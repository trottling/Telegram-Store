package repository

import (
	"context"
	"time"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type ReplenishmentRepository interface {
	Create(ctx context.Context, replenishment *models.Replenishment) error
	GetByMerchantInvoiceID(ctx context.Context, merchant models.Merchant, invoiceID string) (*models.Replenishment, error)
	// GetByID — для кнопки «Проверить оплату» в боте: callback несёт
	// внутренний ID (компактнее, чем merchant+invoiceID в одной строке).
	GetByID(ctx context.Context, id int64) (*models.Replenishment, error)

	// UpdateStatus переводит строку FROM pending -> status; changed=false,
	// если строка уже была не pending (повторный вебхук — не двойное начисление).
	UpdateStatus(ctx context.Context, id int64, status models.ReplenishmentStatus, completedAt *time.Time) (changed bool, err error)

	ListByUserID(ctx context.Context, userID int64, offset, limit int) ([]models.Replenishment, error)
	CountByUserID(ctx context.Context, userID int64) (int64, error)

	// SumPaidByUserMerchant — сумма оплаченных пополнений юзера от одного
	// мерчанта (используется для статистики "всего начислено с рефералки").
	SumPaidByUserMerchant(ctx context.Context, userID int64, merchant models.Merchant) (models.Money, error)

	// ListAllAdmin/CountAllAdmin — межпользовательский вид для админ-панели.
	ListAllAdmin(ctx context.Context, filter models.ReplenishmentAdminFilter, offset, limit int) ([]models.ReplenishmentAdminItem, error)
	CountAllAdmin(ctx context.Context, filter models.ReplenishmentAdminFilter) (int64, error)
}
