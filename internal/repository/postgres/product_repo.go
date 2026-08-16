package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductRepo struct {
	db  *gorm.DB
	log *logrus.Logger
}

func NewProductRepo(db *gorm.DB, log *logrus.Logger) *ProductRepo {
	return &ProductRepo{db: db, log: log}
}

func (r *ProductRepo) Create(ctx context.Context, product *models.Product) error {
	if err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).Create(ctx, product); err != nil {
		r.log.WithError(err).WithField("name", product.Name).Error("product_repo: create failed")
		return err
	}
	r.log.WithFields(logrus.Fields{"product_id": product.ID, "name": product.Name}).Info("product_repo: product created")
	return nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	product, err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrProductNotFound
		}
		r.log.WithError(err).WithField("product_id", id).Error("product_repo: get by id failed")
		return nil, err
	}
	return &product, nil
}

// Update перезаписывает все колонки — см. UserRepo.Update.
func (r *ProductRepo) Update(ctx context.Context, product *models.Product) error {
	_, err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).
		Where("id = ?", product.ID).
		Select("*").
		Updates(ctx, *product)
	if err != nil {
		r.log.WithError(err).WithField("product_id", product.ID).Error("product_repo: update failed")
	}
	return err
}

func (r *ProductRepo) Delete(ctx context.Context, id int64) error {
	_, err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).Where("id = ?", id).Delete(ctx)
	if err != nil {
		r.log.WithError(err).WithField("product_id", id).Error("product_repo: delete failed")
		return err
	}
	r.log.WithField("product_id", id).Info("product_repo: product deleted")
	return nil
}

// inStockClause — товар без остатка не показываем в клиентских листингах.
const inStockClause = "EXISTS (SELECT 1 FROM product_items pi WHERE pi.product_id = products.id AND pi.is_sold = false)"

func (r *ProductRepo) ListActive(ctx context.Context) ([]models.Product, error) {
	products, err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).
		Where("is_active = ? AND "+inStockClause, true).
		Order("id").
		Find(ctx)
	if err != nil {
		r.log.WithError(err).Error("product_repo: list active failed")
	}
	return products, err
}

func (r *ProductRepo) ListActiveByCategory(ctx context.Context, categoryID *int64) ([]models.Product, error) {
	var (
		products []models.Product
		err      error
	)
	if categoryID == nil {
		products, err = gorm.G[models.Product](dbFromCtx(ctx, r.db)).
			Where("is_active = ? AND category_id IS NULL AND "+inStockClause, true).
			Order("id").
			Find(ctx)
	} else {
		products, err = gorm.G[models.Product](dbFromCtx(ctx, r.db)).
			Where("is_active = ? AND category_id = ? AND "+inStockClause, true, *categoryID).
			Order("id").
			Find(ctx)
	}
	if err != nil {
		r.log.WithError(err).WithField("category_id", categoryID).Error("product_repo: list active by category failed")
	}
	return products, err
}

func (r *ProductRepo) AddItems(ctx context.Context, productID int64, contents []string) error {
	if len(contents) == 0 {
		return nil
	}
	items := make([]models.ProductItem, 0, len(contents))
	for _, c := range contents {
		items = append(items, models.ProductItem{ProductID: productID, Content: c})
	}
	if err := gorm.G[models.ProductItem](dbFromCtx(ctx, r.db)).CreateInBatches(ctx, &items, len(items)); err != nil {
		r.log.WithError(err).WithField("product_id", productID).Error("product_repo: add items failed")
		return err
	}
	r.log.WithFields(logrus.Fields{"product_id": productID, "count": len(items)}).Info("product_repo: items added")
	return nil
}

// GetAvailableItem резервирует единицу товара под блокировкой (FOR UPDATE
// SKIP LOCKED) — вызывать только внутри Transactor.WithinTransaction.
func (r *ProductRepo) GetAvailableItem(ctx context.Context, productID int64) (*models.ProductItem, error) {
	item, err := gorm.G[models.ProductItem](
		dbFromCtx(ctx, r.db),
		clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"},
	).
		Where("product_id = ? AND is_sold = ?", productID, false).
		Order("id").
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrProductOutOfStock
		}
		r.log.WithError(err).WithField("product_id", productID).Error("product_repo: get available item failed")
		return nil, err
	}
	return &item, nil
}

// MarkItemSold помечает единицу проданной; purchaseID нужен только для логов.
func (r *ProductRepo) MarkItemSold(ctx context.Context, itemID int64, purchaseID int64) error {
	now := time.Now()
	_, err := gorm.G[models.ProductItem](dbFromCtx(ctx, r.db)).
		Where("id = ?", itemID).
		Updates(ctx, models.ProductItem{IsSold: true, SoldAt: &now})
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{"item_id": itemID, "purchase_id": purchaseID}).Error("product_repo: mark item sold failed")
	}
	return err
}

func (r *ProductRepo) CountAvailableItems(ctx context.Context, productID int64) (int, error) {
	count, err := gorm.G[models.ProductItem](dbFromCtx(ctx, r.db)).
		Where("product_id = ? AND is_sold = ?", productID, false).
		Count(ctx, "*")
	if err != nil {
		r.log.WithError(err).WithField("product_id", productID).Error("product_repo: count available items failed")
	}
	return int(count), err
}

// ListAll — админ-листинг с джойном имени категории и живым счётчиком остатка.
func (r *ProductRepo) ListAll(ctx context.Context, offset, limit int, categoryID *int64) ([]models.ProductAdminSummary, error) {
	q := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Product{}).
		Select("products.id, products.category_id, c.name AS category_name, products.name, products.description, products.price, products.is_active, products.created_at, COUNT(pi.id) FILTER (WHERE pi.is_sold = false) AS available_count").
		Joins("LEFT JOIN categories c ON c.id = products.category_id").
		Joins("LEFT JOIN product_items pi ON pi.product_id = products.id")
	if categoryID != nil {
		q = q.Where("products.category_id = ?", *categoryID)
	}

	var items []models.ProductAdminSummary
	err := q.Group("products.id, c.name").
		Order("products.id").
		Offset(offset).
		Limit(limit).
		Scan(&items).Error
	if err != nil {
		r.log.WithError(err).WithField("category_id", categoryID).Error("product_repo: list all failed")
	}
	return items, err
}

func (r *ProductRepo) CountAll(ctx context.Context, categoryID *int64) (int64, error) {
	var (
		count int64
		err   error
	)
	if categoryID != nil {
		count, err = gorm.G[models.Product](dbFromCtx(ctx, r.db)).Where("category_id = ?", *categoryID).Count(ctx, "*")
	} else {
		count, err = gorm.G[models.Product](dbFromCtx(ctx, r.db)).Count(ctx, "*")
	}
	if err != nil {
		r.log.WithError(err).WithField("category_id", categoryID).Error("product_repo: count all failed")
	}
	return count, err
}

// CountByCategoryID — сколько товаров прямо в категории (любой статус).
func (r *ProductRepo) CountByCategoryID(ctx context.Context, categoryID int64) (int64, error) {
	count, err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).Where("category_id = ?", categoryID).Count(ctx, "*")
	if err != nil {
		r.log.WithError(err).WithField("category_id", categoryID).Error("product_repo: count by category id failed")
	}
	return count, err
}
