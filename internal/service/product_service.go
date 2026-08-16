package service

import (
	"context"

	"github.com/sirupsen/logrus"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
)

type ProductSrv struct {
	productRepo repository.ProductRepository
	cache       domaincache.ProductCache
	log         *logrus.Logger
}

func NewProductSrv(productRepo repository.ProductRepository, cache domaincache.ProductCache, log *logrus.Logger) *ProductSrv {
	return &ProductSrv{productRepo: productRepo, cache: cache, log: log}
}

func (s *ProductSrv) ListAvailable(ctx context.Context) ([]models.Product, error) {
	if products, err := s.cache.GetActiveProducts(ctx); err == nil {
		return products, nil
	}
	s.log.Debug("product_service: active products cache miss")

	products, err := s.productRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetActiveProducts(ctx, products)
	return products, nil
}

func (s *ProductSrv) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	if product, err := s.cache.GetProduct(ctx, id); err == nil {
		return product, nil
	}

	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetProduct(ctx, product)
	return product, nil
}

func (s *ProductSrv) GetAvailableCount(ctx context.Context, productID int64) (int, error) {
	if count, err := s.cache.GetProductAvailableCount(ctx, productID); err == nil {
		return count, nil
	}

	count, err := s.productRepo.CountAvailableItems(ctx, productID)
	if err != nil {
		return 0, err
	}

	_ = s.cache.SetProductAvailableCount(ctx, productID, count)
	return count, nil
}

// ListAllAdmin/CountAllAdmin намеренно мимо кэша.
func (s *ProductSrv) ListAllAdmin(ctx context.Context, offset, limit int, categoryID *int64) ([]models.ProductAdminSummary, error) {
	return s.productRepo.ListAll(ctx, offset, limit, categoryID)
}

func (s *ProductSrv) CountAllAdmin(ctx context.Context, categoryID *int64) (int64, error) {
	return s.productRepo.CountAll(ctx, categoryID)
}
