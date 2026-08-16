package service

import (
	"context"

	"github.com/sirupsen/logrus"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
)

type CategorySrv struct {
	categoryRepo repository.CategoryRepository
	productRepo  repository.ProductRepository
	cache        domaincache.CategoryCache
	log          *logrus.Logger
}

func NewCategorySrv(
	categoryRepo repository.CategoryRepository,
	productRepo repository.ProductRepository,
	cache domaincache.CategoryCache,
	log *logrus.Logger,
) *CategorySrv {
	return &CategorySrv{categoryRepo: categoryRepo, productRepo: productRepo, cache: cache, log: log}
}

func (s *CategorySrv) ListChildren(ctx context.Context, parentID *int64) ([]models.Category, error) {
	if children, err := s.cache.GetCategoryChildren(ctx, parentID); err == nil {
		return children, nil
	}
	s.log.WithField("parent_id", parentID).Debug("category_service: children cache miss")

	children, err := s.categoryRepo.ListChildren(ctx, parentID)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetCategoryChildren(ctx, parentID, children)
	return children, nil
}

func (s *CategorySrv) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *CategorySrv) ListPath(ctx context.Context, id int64) ([]models.Category, error) {
	return s.categoryRepo.ListPath(ctx, id)
}

func (s *CategorySrv) ListProducts(ctx context.Context, categoryID *int64) ([]models.Product, error) {
	return s.productRepo.ListActiveByCategory(ctx, categoryID)
}

// ListAllFlat намеренно мимо кэша — всегда читает из Postgres.
func (s *CategorySrv) ListAllFlat(ctx context.Context) ([]models.Category, error) {
	return s.categoryRepo.ListAllFlat(ctx)
}
