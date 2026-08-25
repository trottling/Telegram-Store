package postgres

import (
	"context"
	"errors"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CategoryRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewCategoryRepo(db *gorm.DB, log *zap.SugaredLogger) *CategoryRepo {
	return &CategoryRepo{db: db, log: log}
}

func (r *CategoryRepo) Create(ctx context.Context, category *models.Category) error {
	category.ID = models.NewCategoryID()
	if err := gorm.G[models.Category](dbFromCtx(ctx, r.db)).Create(ctx, category); err != nil {
		r.log.Errorw("category_repo: create failed", "error", err, "name", category.Name)
		return err
	}
	r.log.Infow("category_repo: category created", "category_id", category.ID, "name", category.Name)
	return nil
}

func (r *CategoryRepo) GetByID(ctx context.Context, id models.CategoryID) (*models.Category, error) {
	category, err := gorm.G[models.Category](dbFromCtx(ctx, r.db)).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrCategoryNotFound
		}
		r.log.Errorw("category_repo: get by id failed", "error", err, "category_id", id)
		return nil, err
	}
	return &category, nil
}

// Update перезаписывает все колонки (Select("*")) — иначе ParentID нельзя
// обнулить обратно (перенос в корень).
func (r *CategoryRepo) Update(ctx context.Context, category *models.Category) error {
	_, err := gorm.G[models.Category](dbFromCtx(ctx, r.db)).
		Where("id = ?", category.ID).
		Select("*").
		Updates(ctx, *category)
	if err != nil {
		r.log.Errorw("category_repo: update failed", "error", err, "category_id", category.ID)
	}
	return err
}

func (r *CategoryRepo) Delete(ctx context.Context, id models.CategoryID) error {
	_, err := gorm.G[models.Category](dbFromCtx(ctx, r.db)).Where("id = ?", id).Delete(ctx)
	if err != nil {
		r.log.Errorw("category_repo: delete failed", "error", err, "category_id", id)
		return err
	}
	r.log.Infow("category_repo: category deleted", "category_id", id)
	return nil
}

// ListPath идёт по ParentID от id до корня и возвращает путь от корня.
func (r *CategoryRepo) ListPath(ctx context.Context, id models.CategoryID) ([]models.Category, error) {
	var path []models.Category

	currentID := &id
	for currentID != nil {
		category, err := gorm.G[models.Category](dbFromCtx(ctx, r.db)).Where("id = ?", *currentID).First(ctx)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, domainerrors.ErrCategoryNotFound
			}
			r.log.Errorw("category_repo: list path failed", "error", err, "category_id", *currentID)
			return nil, err
		}

		path = append([]models.Category{category}, path...)
		currentID = category.ParentID
	}

	return path, nil
}

// ListAllFlat — все категории без фильтров, для админ-панели.
func (r *CategoryRepo) ListAllFlat(ctx context.Context) ([]models.Category, error) {
	categories, err := gorm.G[models.Category](dbFromCtx(ctx, r.db)).
		Order("parent_id NULLS FIRST, name").
		Find(ctx)
	if err != nil {
		r.log.Errorw("category_repo: list all flat failed", "error", err)
	}
	return categories, err
}

// CountChildren — сколько прямых потомков у parentID.
func (r *CategoryRepo) CountChildren(ctx context.Context, parentID models.CategoryID) (int64, error) {
	count, err := gorm.G[models.Category](dbFromCtx(ctx, r.db)).Where("parent_id = ?", parentID).Count(ctx, "*")
	if err != nil {
		r.log.Errorw("category_repo: count children failed", "error", err, "parent_id", parentID)
	}
	return count, err
}
