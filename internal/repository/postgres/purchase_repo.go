package postgres

import (
	"context"
	"errors"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PurchaseRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewPurchaseRepo(db *gorm.DB, log *zap.SugaredLogger) *PurchaseRepo {
	return &PurchaseRepo{db: db, log: log}
}

// CreateBatch — один INSERT на весь батч вместо одного на строку. batchSize
// равен len(purchases): count ограничен MaxBuyQuantity (20), так что это
// всегда один запрос, а не несколько сработавших подряд.
func (r *PurchaseRepo) CreateBatch(ctx context.Context, purchases []models.Purchase) error {
	if err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).CreateInBatches(ctx, &purchases, len(purchases)); err != nil {
		r.log.Errorw("purchase_repo: create batch failed", "error", err, "count", len(purchases))
		return err
	}
	return nil
}

func (r *PurchaseRepo) UpdateStatus(ctx context.Context, purchaseID int64, status models.PurchaseStatus) error {
	_, err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).Where("id = ?", purchaseID).Update(ctx, "status", status)
	if err != nil {
		r.log.Errorw("purchase_repo: update status failed", "error", err, "purchase_id", purchaseID)
	}
	return err
}

func (r *PurchaseRepo) GetByID(ctx context.Context, id int64) (*models.Purchase, error) {
	purchase, err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).
		Preload("Product", nil).
		Preload("Item", nil).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrPurchaseNotFound
		}
		r.log.Errorw("purchase_repo: get by id failed", "error", err, "purchase_id", id)
		return nil, err
	}
	return &purchase, nil
}

func (r *PurchaseRepo) GetByBatchID(ctx context.Context, userID int64, batchID string) ([]models.Purchase, error) {
	purchases, err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).
		Preload("Product", nil).
		Preload("Item", nil).
		Where("user_id = ? AND batch_id = ?", userID, batchID).
		Order("id").
		Find(ctx)
	if err != nil {
		r.log.Errorw("purchase_repo: get by batch id failed", "error", err, "batch_id", batchID)
	}
	return purchases, err
}

// StatsByUserID — счётчик и сумма одним запросом. FILTER нужен потому, что
// считаются все покупки, а суммируются только завершённые.
func (r *PurchaseRepo) StatsByUserID(ctx context.Context, userID int64) (int64, models.Money, error) {
	var stats struct {
		Count      int64
		TotalSpent models.Money
	}

	err := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{}).
		Select("COUNT(*) AS count, COALESCE(SUM(amount) FILTER (WHERE status = ?), 0) AS total_spent", models.PurchaseStatusCompleted).
		Where("user_id = ?", userID).
		Scan(&stats).Error
	if err != nil {
		r.log.Errorw("purchase_repo: stats by user id failed", "error", err, "user_id", userID)
		return 0, models.Money{}, err
	}
	return stats.Count, stats.TotalSpent, nil
}

func (r *PurchaseRepo) CountByProductID(ctx context.Context, productID int64) (int64, error) {
	count, err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).Where("product_id = ?", productID).Count(ctx, "*")
	if err != nil {
		r.log.Errorw("purchase_repo: count by product id failed", "error", err, "product_id", productID)
	}
	return count, err
}

// ListBatchesByUserID группирует по batch_id — GROUP BY не укладывается в gorm.G[T].
func (r *PurchaseRepo) ListBatchesByUserID(ctx context.Context, userID int64, offset, limit int) ([]models.PurchaseBatchSummary, error) {
	var summaries []models.PurchaseBatchSummary
	err := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{}).
		Select("purchases.batch_id AS batch_id, purchases.product_id AS product_id, pr.name AS product_name, MIN(purchases.amount) AS unit_price, COUNT(*) AS quantity, SUM(purchases.amount) AS total_amount, MIN(purchases.created_at) AS created_at").
		Joins("JOIN products pr ON pr.id = purchases.product_id").
		Where("purchases.user_id = ?", userID).
		Group("purchases.batch_id, purchases.product_id, pr.name").
		Order("MIN(purchases.created_at) DESC").
		Offset(offset).
		Limit(limit).
		Scan(&summaries).Error
	if err != nil {
		r.log.Errorw("purchase_repo: list batches failed", "error", err, "user_id", userID)
	}
	return summaries, err
}

func (r *PurchaseRepo) CountBatchesByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{}).
		Where("user_id = ?", userID).
		Distinct("batch_id").
		Count(&count).Error
	if err != nil {
		r.log.Errorw("purchase_repo: count batches failed", "error", err, "user_id", userID)
	}
	return count, err
}

// purchaseFilterQuery — общая база для ListAll/CountAll, добавляет WHERE только для непустых полей фильтра.
func (r *PurchaseRepo) purchaseFilterQuery(ctx context.Context, filter models.PurchaseAdminFilter) *gorm.DB {
	q := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{})
	if filter.UserID != nil {
		q = q.Where("user_id = ?", *filter.UserID)
	}
	if filter.Status != nil {
		q = q.Where("status = ?", *filter.Status)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", *filter.To)
	}
	return q
}

// ListAll/CountAll/GetAdminByID — межпользовательский вид с джойном
// username/product name. users.deleted_at фильтруем вручную — авто-скоуп
// soft-delete у GORM не покрывает джойны.
func (r *PurchaseRepo) ListAll(ctx context.Context, filter models.PurchaseAdminFilter, offset, limit int) ([]models.PurchaseAdminItem, error) {
	var items []models.PurchaseAdminItem
	err := r.purchaseFilterQuery(ctx, filter).
		Select("purchases.id, purchases.user_id, u.username, purchases.product_id, pr.name AS product_name, purchases.item_id, purchases.batch_id, purchases.amount, purchases.status, purchases.created_at, purchases.completed_at").
		Joins("JOIN users u ON u.telegram_id = purchases.user_id AND u.deleted_at IS NULL").
		Joins("JOIN products pr ON pr.id = purchases.product_id").
		Order("purchases.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&items).Error
	if err != nil {
		r.log.Errorw("purchase_repo: list all failed", "error", err)
	}
	return items, err
}

func (r *PurchaseRepo) CountAll(ctx context.Context, filter models.PurchaseAdminFilter) (int64, error) {
	var count int64
	err := r.purchaseFilterQuery(ctx, filter).Count(&count).Error
	if err != nil {
		r.log.Errorw("purchase_repo: count all failed", "error", err)
	}
	return count, err
}

func (r *PurchaseRepo) GetAdminByID(ctx context.Context, id int64) (*models.PurchaseAdminItem, error) {
	var item models.PurchaseAdminItem
	result := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Purchase{}).
		Select("purchases.id, purchases.user_id, u.username, purchases.product_id, pr.name AS product_name, purchases.item_id, purchases.batch_id, purchases.amount, purchases.status, purchases.created_at, purchases.completed_at").
		Joins("JOIN users u ON u.telegram_id = purchases.user_id AND u.deleted_at IS NULL").
		Joins("JOIN products pr ON pr.id = purchases.product_id").
		Where("purchases.id = ?", id).
		Scan(&item)
	if result.Error != nil {
		r.log.Errorw("purchase_repo: get admin by id failed", "error", result.Error, "purchase_id", id)
		return nil, result.Error
	}
	// Scan() не возвращает gorm.ErrRecordNotFound — проверяем RowsAffected.
	if result.RowsAffected == 0 {
		return nil, domainerrors.ErrPurchaseNotFound
	}
	return &item, nil
}
