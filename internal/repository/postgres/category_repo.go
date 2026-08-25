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

// ListChildren — прямые потомки parentID, у которых (или у кого-то в их
// поддереве) есть товар в наличии. Фильтр по HasStock — денормализованному
// агрегату, который поддерживает RecomputeStock; сам ListChildren больше не
// обходит поддерево рекурсивно на каждое чтение.
func (r *CategoryRepo) ListChildren(ctx context.Context, parentID *models.CategoryID) ([]models.Category, error) {
	var (
		children []models.Category
		err      error
	)
	if parentID == nil {
		children, err = gorm.G[models.Category](dbFromCtx(ctx, r.db)).
			Where("parent_id IS NULL AND has_stock = ?", true).
			Order("name").
			Find(ctx)
	} else {
		children, err = gorm.G[models.Category](dbFromCtx(ctx, r.db)).
			Where("parent_id = ? AND has_stock = ?", *parentID, true).
			Order("name").
			Find(ctx)
	}
	if err != nil {
		r.log.Errorw("category_repo: list children failed", "error", err, "parent_id", parentID)
	}
	return children, err
}

// recomputeStockSQL — агрегат снизу вверх: собственные товары категории
// (активные, с непроданной единицей) ИЛИ уже пересчитанный HasStock у
// прямого потомка. RETURNING отдаёт новое значение одним запросом, без
// отдельного SELECT после UPDATE.
const recomputeStockSQL = `
	UPDATE categories SET has_stock = (
		EXISTS (
			SELECT 1 FROM products p
			WHERE p.category_id = categories.id AND p.is_active = true
			  AND EXISTS (SELECT 1 FROM product_items pi WHERE pi.product_id = p.id AND pi.is_sold = false)
		)
		OR EXISTS (SELECT 1 FROM categories child WHERE child.parent_id = categories.id AND child.has_stock = true)
	)
	WHERE id = ?
	RETURNING has_stock`

// RecomputeStock — см. доменный интерфейс.
func (r *CategoryRepo) RecomputeStock(ctx context.Context, categoryID models.CategoryID) (bool, error) {
	db := dbFromCtx(ctx, r.db).WithContext(ctx)

	var before bool
	if err := db.Raw(`SELECT has_stock FROM categories WHERE id = ?`, categoryID).Scan(&before).Error; err != nil {
		r.log.Errorw("category_repo: recompute stock read failed", "error", err, "category_id", categoryID)
		return false, err
	}

	var after bool
	if err := db.Raw(recomputeStockSQL, categoryID).Scan(&after).Error; err != nil {
		r.log.Errorw("category_repo: recompute stock write failed", "error", err, "category_id", categoryID)
		return false, err
	}
	return before != after, nil
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
