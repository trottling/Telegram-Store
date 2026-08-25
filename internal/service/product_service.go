package service

import (
	"context"

	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

type ProductSrv struct {
	productRepo repository.ProductRepository
	cache       domaincache.ProductCache
	log         *zap.SugaredLogger
	// sf схлопывает параллельные промахи кэша на один и тот же ключ в один
	// запрос к БД — иначе при истечении горячего ключа (популярный товар)
	// все параллельные читатели идут в БД одновременно. Нулевое значение
	// готово к использованию, отдельной инициализации не требует.
	sf singleflight.Group
}

func NewProductSrv(productRepo repository.ProductRepository, cache domaincache.ProductCache, log *zap.SugaredLogger) *ProductSrv {
	return &ProductSrv{productRepo: productRepo, cache: cache, log: log}
}

func (s *ProductSrv) ListAvailable(ctx context.Context) ([]models.Product, error) {
	if products, err := s.cache.GetActiveProducts(ctx); err == nil {
		return products, nil
	}
	s.log.Debug("product_service: active products cache miss")

	v, err, _ := s.sf.Do("active", func() (any, error) {
		products, err := s.productRepo.ListActive(ctx)
		if err != nil {
			return nil, err
		}
		_ = s.cache.SetActiveProducts(ctx, products)
		return products, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]models.Product), nil
}

func (s *ProductSrv) GetByID(ctx context.Context, id models.ProductID) (*models.Product, error) {
	if product, err := s.cache.GetProduct(ctx, id); err == nil {
		return product, nil
	}

	v, err, _ := s.sf.Do("product:"+id.String(), func() (any, error) {
		product, err := s.productRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		_ = s.cache.SetProduct(ctx, product)
		return product, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.Product), nil
}

func (s *ProductSrv) GetAvailableCount(ctx context.Context, productID models.ProductID) (int, error) {
	if count, err := s.cache.GetProductAvailableCount(ctx, productID); err == nil {
		return count, nil
	}

	v, err, _ := s.sf.Do("count:"+productID.String(), func() (any, error) {
		count, err := s.productRepo.CountAvailableItems(ctx, productID)
		if err != nil {
			return nil, err
		}
		_ = s.cache.SetProductAvailableCount(ctx, productID, count)
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

// ListAllAdmin/CountAllAdmin намеренно мимо кэша.
func (s *ProductSrv) ListAllAdmin(ctx context.Context, offset, limit int, categoryID *models.CategoryID) ([]models.ProductAdminSummary, error) {
	return s.productRepo.ListAll(ctx, offset, limit, categoryID)
}

func (s *ProductSrv) CountAllAdmin(ctx context.Context, categoryID *models.CategoryID) (int64, error) {
	return s.productRepo.CountAll(ctx, categoryID)
}
