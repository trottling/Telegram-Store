package postgres

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	domainerrors "github.com/trottling/TG-Store/internal/domain/errors"
	"github.com/trottling/TG-Store/internal/domain/models"
	"gorm.io/gorm"
)

type PurchaseRepo struct {
	db  *gorm.DB
	log *logrus.Logger
}

func NewPurchaseRepo(db *gorm.DB, log *logrus.Logger) *PurchaseRepo {
	return &PurchaseRepo{db: db, log: log}
}

func (r *PurchaseRepo) Create(ctx context.Context, purchase *models.Purchase) error {
	if err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).Create(ctx, purchase); err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{"user_id": purchase.UserID, "product_id": purchase.ProductID}).Error("purchase_repo: create failed")
		return err
	}
	return nil
}

func (r *PurchaseRepo) UpdateStatus(ctx context.Context, purchaseID int64, status models.PurchaseStatus) error {
	_, err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).Where("id = ?", purchaseID).Update(ctx, "status", status)
	if err != nil {
		r.log.WithError(err).WithField("purchase_id", purchaseID).Error("purchase_repo: update status failed")
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
		r.log.WithError(err).WithField("purchase_id", id).Error("purchase_repo: get by id failed")
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
		r.log.WithError(err).WithField("batch_id", batchID).Error("purchase_repo: get by batch id failed")
	}
	return purchases, err
}

func (r *PurchaseRepo) GetByUserID(ctx context.Context, userID int64, offset, limit int) ([]models.Purchase, error) {
	purchases, err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).
		Preload("Product", nil).
		Preload("Item", nil).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(ctx)
	if err != nil {
		r.log.WithError(err).WithField("user_id", userID).Error("purchase_repo: get by user id failed")
	}
	return purchases, err
}

func (r *PurchaseRepo) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	count, err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).Where("user_id = ?", userID).Count(ctx, "*")
	if err != nil {
		r.log.WithError(err).WithField("user_id", userID).Error("purchase_repo: count by user id failed")
	}
	return count, err
}

func (r *PurchaseRepo) CountByProductID(ctx context.Context, productID int64) (int64, error) {
	count, err := gorm.G[models.Purchase](dbFromCtx(ctx, r.db)).Where("product_id = ?", productID).Count(ctx, "*")
	if err != nil {
		r.log.WithError(err).WithField("product_id", productID).Error("purchase_repo: count by product id failed")
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
		r.log.WithError(err).WithField("user_id", userID).Error("purchase_repo: list batches failed")
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
		r.log.WithError(err).WithField("user_id", userID).Error("purchase_repo: count batches failed")
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
		r.log.WithError(err).Error("purchase_repo: list all failed")
	}
	return items, err
}

func (r *PurchaseRepo) CountAll(ctx context.Context, filter models.PurchaseAdminFilter) (int64, error) {
	var count int64
	err := r.purchaseFilterQuery(ctx, filter).Count(&count).Error
	if err != nil {
		r.log.WithError(err).Error("purchase_repo: count all failed")
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
		r.log.WithError(result.Error).WithField("purchase_id", id).Error("purchase_repo: get admin by id failed")
		return nil, result.Error
	}
	// Scan() не возвращает gorm.ErrRecordNotFound — проверяем RowsAffected.
	if result.RowsAffected == 0 {
		return nil, domainerrors.ErrPurchaseNotFound
	}
	return &item, nil
}
