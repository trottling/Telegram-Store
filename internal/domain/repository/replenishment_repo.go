package repository

import (
	"context"
	"time"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type ReplenishmentRepository interface {
	Create(ctx context.Context, replenishment *models.Replenishment) error
	GetByMerchantInvoiceID(ctx context.Context, merchant models.Merchant, invoiceID string) (*models.Replenishment, error)

	// UpdateStatus переводит строку FROM pending -> status; changed=false,
	// если строка уже была не pending (повторный вебхук — не двойное начисление).
	UpdateStatus(ctx context.Context, id int64, status models.ReplenishmentStatus, completedAt *time.Time) (changed bool, err error)

	ListByUserID(ctx context.Context, userID int64, offset, limit int) ([]models.Replenishment, error)
	CountByUserID(ctx context.Context, userID int64) (int64, error)

	// ListAllAdmin/CountAllAdmin — межпользовательский вид для админ-панели.
	ListAllAdmin(ctx context.Context, userID *int64, offset, limit int) ([]models.ReplenishmentAdminItem, error)
	CountAllAdmin(ctx context.Context, userID *int64) (int64, error)
}
