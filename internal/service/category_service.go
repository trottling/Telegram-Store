package service

import (
	"context"

	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

type CategorySrv struct {
	categoryRepo repository.CategoryRepository
	productRepo  repository.ProductRepository
	cache        domaincache.CategoryCache
	log          *zap.SugaredLogger
	// sf — см. комментарий у ProductSrv.sf; здесь защищает от громового стада
	// на популярную категорию (её ListChildren зовётся на каждый заход в
	// раздел каталога).
	sf singleflight.Group
}

func NewCategorySrv(
	categoryRepo repository.CategoryRepository,
	productRepo repository.ProductRepository,
	cache domaincache.CategoryCache,
	log *zap.SugaredLogger,
) *CategorySrv {
	return &CategorySrv{categoryRepo: categoryRepo, productRepo: productRepo, cache: cache, log: log}
}

func (s *CategorySrv) ListChildren(ctx context.Context, parentID *models.CategoryID) ([]models.Category, error) {
	if children, err := s.cache.GetCategoryChildren(ctx, parentID); err == nil {
		return children, nil
	}
	s.log.Debugw("category_service: children cache miss", "parent_id", parentID)

	key := "root"
	if parentID != nil {
		key = parentID.String()
	}
	v, err, _ := s.sf.Do(key, func() (any, error) {
		children, err := s.categoryRepo.ListChildren(ctx, parentID)
		if err != nil {
			return nil, err
		}
		_ = s.cache.SetCategoryChildren(ctx, parentID, children)
		return children, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]models.Category), nil
}

func (s *CategorySrv) GetByID(ctx context.Context, id models.CategoryID) (*models.Category, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *CategorySrv) ListPath(ctx context.Context, id models.CategoryID) ([]models.Category, error) {
	return s.categoryRepo.ListPath(ctx, id)
}

func (s *CategorySrv) ListProducts(ctx context.Context, categoryID *models.CategoryID) ([]models.Product, error) {
	return s.productRepo.ListActiveByCategory(ctx, categoryID)
}

// ListAllFlat намеренно мимо кэша — всегда читает из Postgres.
func (s *CategorySrv) ListAllFlat(ctx context.Context) ([]models.Category, error) {
	return s.categoryRepo.ListAllFlat(ctx)
}
