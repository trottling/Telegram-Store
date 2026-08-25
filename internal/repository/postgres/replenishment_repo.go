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

type ReplenishmentRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewReplenishmentRepo(db *gorm.DB, log *zap.SugaredLogger) *ReplenishmentRepo {
	return &ReplenishmentRepo{db: db, log: log}
}

func (r *ReplenishmentRepo) Create(ctx context.Context, replenishment *models.Replenishment) error {
	if err := gorm.G[models.Replenishment](dbFromCtx(ctx, r.db)).Create(ctx, replenishment); err != nil {
		r.log.Errorw("replenishment_repo: create failed", "error", err, "user_id", replenishment.UserID)
		return err
	}
	return nil
}

func (r *ReplenishmentRepo) GetByMerchantInvoiceID(ctx context.Context, merchant models.Merchant, invoiceID string) (*models.Replenishment, error) {
	replenishment, err := gorm.G[models.Replenishment](dbFromCtx(ctx, r.db)).
		Where("merchant = ? AND invoice_id = ?", merchant, invoiceID).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrReplenishmentNotFound
		}
		r.log.Errorw("replenishment_repo: get by merchant invoice id failed", "error", err, "merchant", merchant, "invoice_id", invoiceID)
		return nil, err
	}
	return &replenishment, nil
}

func (r *ReplenishmentRepo) GetByID(ctx context.Context, id int64) (*models.Replenishment, error) {
	replenishment, err := gorm.G[models.Replenishment](dbFromCtx(ctx, r.db)).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrReplenishmentNotFound
		}
		r.log.Errorw("replenishment_repo: get by id failed", "error", err, "id", id)
		return nil, err
	}
	return &replenishment, nil
}

// UpdateStatus — WHERE status = 'pending' защищает от повторной обработки
// вебхука: второй вызов на уже обработанной строке changed=false, err=nil.
func (r *ReplenishmentRepo) UpdateStatus(ctx context.Context, id int64, status models.ReplenishmentStatus, completedAt *time.Time) (bool, error) {
	rows, err := gorm.G[models.Replenishment](dbFromCtx(ctx, r.db)).
		Where("id = ? AND status = ?", id, models.ReplenishmentStatusPending).
		Updates(ctx, models.Replenishment{Status: status, CompletedAt: completedAt})
	if err != nil {
		r.log.Errorw("replenishment_repo: update status failed", "error", err, "replenishment_id", id)
		return false, err
	}
	return rows > 0, nil
}

func (r *ReplenishmentRepo) ListByUserID(ctx context.Context, userID int64, offset, limit int) ([]models.Replenishment, error) {
	replenishments, err := gorm.G[models.Replenishment](dbFromCtx(ctx, r.db)).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(ctx)
	if err != nil {
		r.log.Errorw("replenishment_repo: list by user id failed", "error", err, "user_id", userID)
	}
	return replenishments, err
}

func (r *ReplenishmentRepo) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	count, err := gorm.G[models.Replenishment](dbFromCtx(ctx, r.db)).Where("user_id = ?", userID).Count(ctx, "*")
	if err != nil {
		r.log.Errorw("replenishment_repo: count by user id failed", "error", err, "user_id", userID)
	}
	return count, err
}

// SumPaidByUserMerchant — сумма оплаченных пополнений (COALESCE — 0, если строк нет).
func (r *ReplenishmentRepo) SumPaidByUserMerchant(ctx context.Context, userID int64, merchant models.Merchant) (models.Money, error) {
	var sum models.Money
	err := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Replenishment{}).
		Where("user_id = ? AND merchant = ? AND status = ?", userID, merchant, models.ReplenishmentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sum).Error
	if err != nil {
		r.log.Errorw("replenishment_repo: sum paid by user merchant failed", "error", err, "user_id", userID, "merchant", merchant)
	}
	return sum, err
}

// replenishmentAdminQuery — общая база для ListAllAdmin/CountAllAdmin.
// users.deleted_at фильтруем вручную — авто-скоуп soft-delete не покрывает джойны.
func (r *ReplenishmentRepo) replenishmentAdminQuery(ctx context.Context, filter models.ReplenishmentAdminFilter) *gorm.DB {
	q := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.Replenishment{})
	if filter.UserID != nil {
		q = q.Where("replenishments.user_id = ?", *filter.UserID)
	}
	if filter.Merchant != nil {
		q = q.Where("replenishments.merchant = ?", *filter.Merchant)
	}
	return q
}

func (r *ReplenishmentRepo) ListAllAdmin(ctx context.Context, filter models.ReplenishmentAdminFilter, offset, limit int) ([]models.ReplenishmentAdminItem, error) {
	var items []models.ReplenishmentAdminItem
	err := r.replenishmentAdminQuery(ctx, filter).
		Select("replenishments.id, replenishments.user_id, u.username, replenishments.merchant, replenishments.invoice_id, replenishments.amount, replenishments.status, replenishments.created_at, replenishments.completed_at").
		Joins("JOIN users u ON u.telegram_id = replenishments.user_id AND u.deleted_at IS NULL").
		Order("replenishments.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&items).Error
	if err != nil {
		r.log.Errorw("replenishment_repo: list all admin failed", "error", err)
	}
	return items, err
}

func (r *ReplenishmentRepo) CountAllAdmin(ctx context.Context, filter models.ReplenishmentAdminFilter) (int64, error) {
	var count int64
	err := r.replenishmentAdminQuery(ctx, filter).Count(&count).Error
	if err != nil {
		r.log.Errorw("replenishment_repo: count all admin failed", "error", err)
	}
	return count, err
}
