package postgres

import (
	"context"
	"errors"
	"time"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProductRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewProductRepo(db *gorm.DB, log *zap.SugaredLogger) *ProductRepo {
	return &ProductRepo{db: db, log: log}
}

func (r *ProductRepo) Create(ctx context.Context, product *models.Product) error {
	if err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).Create(ctx, product); err != nil {
		r.log.Errorw("product_repo: create failed", "error", err, "name", product.Name)
		return err
	}
	r.log.Infow("product_repo: product created", "product_id", product.ID, "name", product.Name)
	return nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	product, err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrProductNotFound
		}
		r.log.Errorw("product_repo: get by id failed", "error", err, "product_id", id)
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
		r.log.Errorw("product_repo: update failed", "error", err, "product_id", product.ID)
	}
	return err
}

func (r *ProductRepo) Delete(ctx context.Context, id int64) error {
	_, err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).Where("id = ?", id).Delete(ctx)
	if err != nil {
		r.log.Errorw("product_repo: delete failed", "error", err, "product_id", id)
		return err
	}
	r.log.Infow("product_repo: product deleted", "product_id", id)
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
		r.log.Errorw("product_repo: list active failed", "error", err)
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
		r.log.Errorw("product_repo: list active by category failed", "error", err, "category_id", categoryID)
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
		r.log.Errorw("product_repo: add items failed", "error", err, "product_id", productID)
		return err
	}
	r.log.Infow("product_repo: items added", "product_id", productID, "count", len(items))
	return nil
}

// reserveItemSQL — забрать одну непроданную единицу и сразу пометить её
// проданной, одним запросом. Подзапрос с FOR UPDATE SKIP LOCKED — та же защита
// от overselling, что и раньше: параллельная покупка пропускает заблокированную
// строку, а не ждёт её. Раздельные SELECT и UPDATE давали три запроса на
// единицу (до 60 на покупку) внутри транзакции, удерживающей локи.
//
// Это второй в репозитории случай .Raw() (первый — рекурсивный CTE в
// category_repo): ни gorm.G[T], ни chainable-билдер не выражают
// UPDATE ... WHERE id = (SELECT ... FOR UPDATE SKIP LOCKED) RETURNING.
const reserveItemSQL = `
UPDATE product_items
SET is_sold = true, sold_at = ?
WHERE id = (
	SELECT id FROM product_items
	WHERE product_id = ? AND is_sold = false
	ORDER BY id
	LIMIT 1
	FOR UPDATE SKIP LOCKED
)
RETURNING *`

// ReserveItem резервирует единицу товара и помечает её проданной — вызывать
// только внутри Transactor.WithinTransaction: без охватывающей транзакции
// единица окажется списанной даже при неудачной покупке.
func (r *ProductRepo) ReserveItem(ctx context.Context, productID int64) (*models.ProductItem, error) {
	var item models.ProductItem

	result := dbFromCtx(ctx, r.db).WithContext(ctx).
		Raw(reserveItemSQL, time.Now(), productID).
		Scan(&item)
	if result.Error != nil {
		r.log.Errorw("product_repo: reserve item failed", "error", result.Error, "product_id", productID)
		return nil, result.Error
	}
	// Подзапрос ничего не нашёл — свободных единиц не осталось. Raw+Scan не
	// возвращает ErrRecordNotFound, поэтому смотрим на RowsAffected.
	if result.RowsAffected == 0 {
		return nil, domainerrors.ErrProductOutOfStock
	}
	return &item, nil
}

func (r *ProductRepo) CountAvailableItems(ctx context.Context, productID int64) (int, error) {
	count, err := gorm.G[models.ProductItem](dbFromCtx(ctx, r.db)).
		Where("product_id = ? AND is_sold = ?", productID, false).
		Count(ctx, "*")
	if err != nil {
		r.log.Errorw("product_repo: count available items failed", "error", err, "product_id", productID)
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
		r.log.Errorw("product_repo: list all failed", "error", err, "category_id", categoryID)
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
		r.log.Errorw("product_repo: count all failed", "error", err, "category_id", categoryID)
	}
	return count, err
}

// CountByCategoryID — сколько товаров прямо в категории (любой статус).
func (r *ProductRepo) CountByCategoryID(ctx context.Context, categoryID int64) (int64, error) {
	count, err := gorm.G[models.Product](dbFromCtx(ctx, r.db)).Where("category_id = ?", categoryID).Count(ctx, "*")
	if err != nil {
		r.log.Errorw("product_repo: count by category id failed", "error", err, "category_id", categoryID)
	}
	return count, err
}
