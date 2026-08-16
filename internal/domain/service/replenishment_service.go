package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ReplenishmentService принимает Telegram ID пользователя, не внутренний ключ.
type ReplenishmentService interface {
	// CreateInvoice проверяет merchant включён/сумма в пределах min/max
	// (см. Settings), создаёт счёт у мерчанта и pending-строку Replenishment.
	CreateInvoice(ctx context.Context, telegramID int64, merchant models.Merchant, amount float64) (paymentURL string, err error)

	// Confirm/Fail — вызываются вебхуком мерчанта. Confirm идемпотентен:
	// повторный вызов на уже обработанной строке баланс не трогает.
	Confirm(ctx context.Context, merchant models.Merchant, invoiceID string) error
	Fail(ctx context.Context, merchant models.Merchant, invoiceID string) error

	ListUserReplenishments(ctx context.Context, telegramID int64, offset, limit int) ([]models.Replenishment, error)
	CountUserReplenishments(ctx context.Context, telegramID int64) (int64, error)

	// SumUserMerchantAmount — сколько оплачено юзеру от одного мерчанта
	// (используется для "всего начислено" в реферальной статистике, Merchant=referral).
	SumUserMerchantAmount(ctx context.Context, telegramID int64, merchant models.Merchant) (float64, error)

	// ListAllAdmin/CountAllAdmin — межпользовательский вид для админ-панели, userID nil = без фильтра.
	ListAllAdmin(ctx context.Context, userID *int64, offset, limit int) ([]models.ReplenishmentAdminItem, error)
	CountAllAdmin(ctx context.Context, userID *int64) (int64, error)
}
