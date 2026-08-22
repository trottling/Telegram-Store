package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ReplenishmentService принимает Telegram ID пользователя, не внутренний ключ.
type ReplenishmentService interface {
	// CreateInvoice проверяет merchant включён/сумма в пределах min/max
	// (см. Settings), создаёт счёт у мерчанта и pending-строку Replenishment.
	// replenishmentID — её ID, нужен боту для кнопки «Проверить оплату» (см. CheckInvoice).
	CreateInvoice(ctx context.Context, telegramID int64, merchant models.Merchant, amount float64) (paymentURL string, replenishmentID int64, err error)

	// CheckInvoice — ручная проверка статуса по кнопке в боте, на случай если
	// вебхук мерчанта ещё не пришёл или потерялся. Не подменяет вебхук: если
	// счёт уже не pending, просто возвращает записанный статус; иначе
	// спрашивает CheckStatus у самого мерчанта и, если тот сообщает
	// paid/failed, доводит дело тем же Confirm/Fail. telegramID — чей это
	// запрос, чтобы один пользователь не мог дёргать чужой счёт по id. amount —
	// записанная сумма счёта, боту нужна перерисовать текст карточки без
	// лишнего похода в БД.
	CheckInvoice(ctx context.Context, telegramID int64, replenishmentID int64) (status models.ReplenishmentStatus, amount float64, err error)

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
