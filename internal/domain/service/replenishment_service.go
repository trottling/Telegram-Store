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
	// Confirm зачисляет баланс по оплаченному счёту. paidAmount — сумма, о
	// которой сообщил мерчант; 0, если он её не сообщает. Расхождение с
	// записанной суммой не блокирует зачисление (начисляем всегда своё,
	// согласованное с пользователем), но логируется как сигнал рассинхрона.
	Confirm(ctx context.Context, merchant models.Merchant, invoiceID string, paidAmount float64) error
	Fail(ctx context.Context, merchant models.Merchant, invoiceID string) error

	ListUserReplenishments(ctx context.Context, telegramID int64, offset, limit int) ([]models.Replenishment, error)
	CountUserReplenishments(ctx context.Context, telegramID int64) (int64, error)

	// SumUserMerchantAmount — сколько оплачено юзеру от одного мерчанта
	// (используется для "всего начислено" в реферальной статистике, Merchant=referral).
	SumUserMerchantAmount(ctx context.Context, telegramID int64, merchant models.Merchant) (float64, error)

	// ListAllAdmin/CountAllAdmin — межпользовательский вид для админ-панели.
	ListAllAdmin(ctx context.Context, filter models.ReplenishmentAdminFilter, offset, limit int) ([]models.ReplenishmentAdminItem, error)
	CountAllAdmin(ctx context.Context, filter models.ReplenishmentAdminFilter) (int64, error)
}
