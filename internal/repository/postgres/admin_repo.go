package postgres

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AdminLogRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewAdminLogRepo(db *gorm.DB, log *zap.SugaredLogger) *AdminLogRepo {
	return &AdminLogRepo{db: db, log: log}
}

func (r *AdminLogRepo) Create(ctx context.Context, entry *models.AdminLog) error {
	if err := gorm.G[models.AdminLog](dbFromCtx(ctx, r.db)).Create(ctx, entry); err != nil {
		r.log.Errorw("admin_log_repo: create failed", "error", err, "admin_id", entry.AdminID, "action", entry.Action)
		return err
	}
	return nil
}

func (r *AdminLogRepo) ListByAdmin(ctx context.Context, adminID int64, offset, limit int) ([]models.AdminLog, error) {
	logs, err := gorm.G[models.AdminLog](dbFromCtx(ctx, r.db)).
		Where("admin_id = ?", adminID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(ctx)
	if err != nil {
		r.log.Errorw("admin_log_repo: list by admin failed", "error", err, "admin_id", adminID)
	}
	return logs, err
}

// adminLogFilterQuery — общая база для ListAll/CountAll с опциональным фильтром по adminID.
func (r *AdminLogRepo) adminLogFilterQuery(ctx context.Context, adminID *int64) *gorm.DB {
	q := dbFromCtx(ctx, r.db).WithContext(ctx).Model(&models.AdminLog{})
	if adminID != nil {
		q = q.Where("admin_id = ?", *adminID)
	}
	return q
}

// ListAll — журнал действий по всем админам (или одному, если adminID задан).
func (r *AdminLogRepo) ListAll(ctx context.Context, adminID *int64, offset, limit int) ([]models.AdminLog, error) {
	var logs []models.AdminLog
	err := r.adminLogFilterQuery(ctx, adminID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		r.log.Errorw("admin_log_repo: list all failed", "error", err)
	}
	return logs, err
}

func (r *AdminLogRepo) CountAll(ctx context.Context, adminID *int64) (int64, error) {
	var count int64
	err := r.adminLogFilterQuery(ctx, adminID).Count(&count).Error
	if err != nil {
		r.log.Errorw("admin_log_repo: count all failed", "error", err)
	}
	return count, err
}
