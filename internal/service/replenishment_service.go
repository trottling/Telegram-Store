package service

import (
	"context"
	"time"

	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service/payment"
	paymentsmetrics "github.com/trottling/Telegram-Store/internal/metrics/payments"
	"go.uber.org/zap"
)

type ReplenishmentSrv struct {
	replenishmentRepo repository.ReplenishmentRepository
	userRepo          repository.UserRepository
	// providers — по одному на реальный мерчант; MerchantReferral сюда не
	// входит, начисления с рефералов создаются напрямую, без CreateInvoice.
	// В backend-процессе (вебхуки/листинг) может быть nil — CreateInvoice
	// оттуда не вызывается.
	providers  map[models.Merchant]payment.PaymentProvider
	transactor repository.Transactor
	cache      domaincache.UserCache
	// checkCooldown — минимальный интервал между CheckStatus по одному счёту
	// (см. CheckInvoice). CreateInvoice его не вызывает, поэтому, как и
	// providers, в backend-процессе может быть nil.
	checkCooldown domaincache.ReplenishmentCheckCooldown
	log           *zap.SugaredLogger
}

func NewReplenishmentSrv(
	replenishmentRepo repository.ReplenishmentRepository,
	userRepo repository.UserRepository,
	transactor repository.Transactor,
	providers map[models.Merchant]payment.PaymentProvider,
	cache domaincache.UserCache,
	checkCooldown domaincache.ReplenishmentCheckCooldown,
	log *zap.SugaredLogger,
) *ReplenishmentSrv {
	return &ReplenishmentSrv{
		replenishmentRepo: replenishmentRepo,
		userRepo:          userRepo,
		transactor:        transactor,
		providers:         providers,
		cache:             cache,
		checkCooldown:     checkCooldown,
		log:               log,
	}
}

const replenishmentDescription = "Пополнение баланса"

func (s *ReplenishmentSrv) CreateInvoice(ctx context.Context, telegramID int64, merchant models.Merchant, amount models.Money) (string, int64, error) {
	if amount.IsZero() {
		return "", 0, domainerrors.ErrInvalidAmount
	}

	provider, ok := s.providers[merchant]
	if !ok {
		return "", 0, domainerrors.ErrInvalidMerchant
	}

	paymentURL, invoiceID, err := provider.CreateInvoice(ctx, telegramID, amount, replenishmentDescription)
	if err != nil {
		return "", 0, err
	}

	replenishment := &models.Replenishment{
		UserID:    telegramID,
		Merchant:  merchant,
		InvoiceID: invoiceID,
		Amount:    amount,
		Status:    models.ReplenishmentStatusPending,
	}
	if err = s.replenishmentRepo.Create(ctx, replenishment); err != nil {
		// Счёт у мерчанта уже создан, а у нас записи о нём нет: оплата придёт
		// вебхуком на неизвестный invoice_id и будет отброшена. Компенсации
		// нет, поэтому оставляем в логе всё нужное для ручного разбора.
		s.log.Errorw("replenishment_service: invoice created at merchant but not recorded",
			"error", err, "user_id", telegramID, "merchant", merchant,
			"invoice_id", invoiceID, "amount", amount,
		)
		return "", 0, err
	}

	return paymentURL, replenishment.ID, nil
}

// CheckInvoice — см. doc-комментарий интерфейса. CheckStatus у мерчанта
// вызывается только если счёт всё ещё pending — иначе лишний внешний запрос
// за уже известным ответом.
func (s *ReplenishmentSrv) CheckInvoice(ctx context.Context, telegramID int64, replenishmentID int64) (models.ReplenishmentStatus, models.Money, error) {
	replenishment, err := s.replenishmentRepo.GetByID(ctx, replenishmentID)
	if err != nil {
		return "", models.Money{}, err
	}
	if replenishment.UserID != telegramID {
		// Чужой счёт — ведём себя как с несуществующим, а не 403: не палим,
		// что id вообще существует.
		return "", models.Money{}, domainerrors.ErrReplenishmentNotFound
	}
	if replenishment.Status != models.ReplenishmentStatusPending {
		return replenishment.Status, replenishment.Amount, nil
	}

	// Кулдаун — не поход к мерчанту при каждом тапе по кнопке. Счёт остаётся
	// pending, ошибки тут нет: пользователь просто проверял слишком часто.
	if s.checkCooldown != nil {
		acquired, err := s.checkCooldown.TryAcquire(ctx, replenishmentID)
		if err != nil {
			return "", models.Money{}, err
		}
		if !acquired {
			return models.ReplenishmentStatusPending, replenishment.Amount, nil
		}
	}

	provider, ok := s.providers[replenishment.Merchant]
	if !ok {
		return "", models.Money{}, domainerrors.ErrInvalidMerchant
	}

	status, err := provider.CheckStatus(ctx, replenishment.InvoiceID)
	if err != nil {
		return "", models.Money{}, err
	}

	switch status {
	case payment.PaymentStatusPaid:
		// paidAmount — нулевое значение: CheckStatus большинства мерчантов
		// сумму не возвращает; Confirm в этом случае зачисляет записанную
		// сумму (см. её собственный doc-комментарий).
		if err = s.Confirm(ctx, replenishment.Merchant, replenishment.InvoiceID, models.Money{}); err != nil {
			return "", models.Money{}, err
		}
		return models.ReplenishmentStatusPaid, replenishment.Amount, nil
	case payment.PaymentStatusFailed, payment.PaymentStatusCancelled:
		if err = s.Fail(ctx, replenishment.Merchant, replenishment.InvoiceID); err != nil {
			return "", models.Money{}, err
		}
		return models.ReplenishmentStatusFailed, replenishment.Amount, nil
	default:
		return models.ReplenishmentStatusPending, replenishment.Amount, nil
	}
}

// Confirm зачисляет баланс и помечает счёт оплаченным одной транзакцией.
// Идемпотентно: UpdateStatus меняет строку только из pending, повторный
// вебхук мерчанта до UpdateBalance не доходит.
//
// Транзакция здесь обязательна именно из-за идемпотентности: если пометить
// счёт оплаченным отдельным коммитом, а начисление упадёт, то ретрай вебхука
// получит changed=false и молча вернёт 200 — деньги списаны у клиента и
// потеряны без следа.
func (s *ReplenishmentSrv) Confirm(ctx context.Context, merchant models.Merchant, invoiceID string, paidAmount models.Money) error {
	replenishment, err := s.replenishmentRepo.GetByMerchantInvoiceID(ctx, merchant, invoiceID)
	if err != nil {
		return err
	}

	// Начисляем всегда записанную сумму — именно её подтверждал пользователь.
	// Расхождение само по себе зачислению не мешает, но означает, что наша
	// запись и данные мерчанта разошлись, и это стоит увидеть в логах.
	if !paidAmount.IsZero() && !paidAmount.Equal(replenishment.Amount) {
		s.log.Warnw("replenishment_service: merchant reported a different amount",
			"merchant", merchant, "invoice_id", invoiceID,
			"recorded_amount", replenishment.Amount, "reported_amount", paidAmount,
		)
	}

	var credited bool
	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		now := time.Now()
		changed, txErr := s.replenishmentRepo.UpdateStatus(ctx, replenishment.ID, models.ReplenishmentStatusPaid, &now)
		if txErr != nil || !changed {
			return txErr
		}
		if txErr = s.userRepo.UpdateBalance(ctx, replenishment.UserID, replenishment.Amount.Decimal()); txErr != nil {
			return txErr
		}
		credited = true
		return nil
	})
	if err != nil || !credited {
		return err
	}

	// credited==true — этот вызов реально провёл платёж, не ретрай уже
	// обработанного вебхука (см. doc-комментарий Confirm). Считать по err==nil
	// без этой проверки задваивало бы метрику на каждом ретрае мерчанта.
	paymentsmetrics.ReplenishmentsTotal.WithLabelValues(string(merchant), "paid").Inc()
	paymentsmetrics.ReplenishmentAmountTotal.WithLabelValues(string(merchant)).Add(replenishment.Amount.Float64())

	// Инвалидация только после коммита: до него в Postgres ещё старый баланс,
	// и параллельный читатель залил бы его обратно в кэш.
	logInvalidation(s.log, s.cache.InvalidateUser(ctx, replenishment.UserID), "user", replenishment.UserID)

	s.log.Infow("replenishment_service: balance credited", "user_id", replenishment.UserID, "merchant", merchant, "invoice_id", invoiceID, "amount", replenishment.Amount)
	return nil
}

func (s *ReplenishmentSrv) Fail(ctx context.Context, merchant models.Merchant, invoiceID string) error {
	replenishment, err := s.replenishmentRepo.GetByMerchantInvoiceID(ctx, merchant, invoiceID)
	if err != nil {
		return err
	}

	now := time.Now()
	changed, err := s.replenishmentRepo.UpdateStatus(ctx, replenishment.ID, models.ReplenishmentStatusFailed, &now)
	if err != nil {
		return err
	}
	if !changed {
		// Строка была не pending. Обычно это повторный вебхук, но если счёт уже
		// оплачен, то мерчант считает его неудачным, а мы — оплаченным, и деньги
		// уже зачислены. Разбираться придётся руками, поэтому пишем в лог.
		if replenishment.Status == models.ReplenishmentStatusPaid {
			s.log.Warnw("replenishment_service: merchant reported failure for an already paid invoice",
				"user_id", replenishment.UserID, "merchant", merchant, "invoice_id", invoiceID,
			)
		}
		return nil
	}

	paymentsmetrics.ReplenishmentsTotal.WithLabelValues(string(merchant), "failed").Inc()
	return nil
}

func (s *ReplenishmentSrv) ListUserReplenishments(ctx context.Context, telegramID int64, offset, limit int) ([]models.Replenishment, error) {
	return s.replenishmentRepo.ListByUserID(ctx, telegramID, offset, limit)
}

func (s *ReplenishmentSrv) GetUserReplenishment(ctx context.Context, telegramID int64, id int64) (*models.Replenishment, error) {
	replenishment, err := s.replenishmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if replenishment.UserID != telegramID {
		return nil, domainerrors.ErrReplenishmentNotFound
	}
	return replenishment, nil
}

func (s *ReplenishmentSrv) CountUserReplenishments(ctx context.Context, telegramID int64) (int64, error) {
	return s.replenishmentRepo.CountByUserID(ctx, telegramID)
}

func (s *ReplenishmentSrv) SumUserMerchantAmount(ctx context.Context, telegramID int64, merchant models.Merchant) (models.Money, error) {
	return s.replenishmentRepo.SumPaidByUserMerchant(ctx, telegramID, merchant)
}

func (s *ReplenishmentSrv) ListAllAdmin(ctx context.Context, filter models.ReplenishmentAdminFilter, offset, limit int) ([]models.ReplenishmentAdminItem, error) {
	return s.replenishmentRepo.ListAllAdmin(ctx, filter, offset, limit)
}

func (s *ReplenishmentSrv) CountAllAdmin(ctx context.Context, filter models.ReplenishmentAdminFilter) (int64, error) {
	return s.replenishmentRepo.CountAllAdmin(ctx, filter)
}
