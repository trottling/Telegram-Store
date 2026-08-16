package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

type ReplenishmentSrv struct {
	replenishmentRepo repository.ReplenishmentRepository
	userRepo          repository.UserRepository
	// providers — по одному на реальный мерчант; MerchantReferral сюда не
	// входит, начисления с рефералов создаются напрямую, без CreateInvoice.
	// В backend-процессе (вебхуки/листинг) может быть nil — CreateInvoice
	// оттуда не вызывается.
	providers map[models.Merchant]payment.PaymentProvider
	cache     domaincache.UserCache
	log       *logrus.Logger
}

func NewReplenishmentSrv(
	replenishmentRepo repository.ReplenishmentRepository,
	userRepo repository.UserRepository,
	providers map[models.Merchant]payment.PaymentProvider,
	cache domaincache.UserCache,
	log *logrus.Logger,
) *ReplenishmentSrv {
	return &ReplenishmentSrv{
		replenishmentRepo: replenishmentRepo,
		userRepo:          userRepo,
		providers:         providers,
		cache:             cache,
		log:               log,
	}
}

const replenishmentDescription = "Пополнение баланса"

func (s *ReplenishmentSrv) CreateInvoice(ctx context.Context, telegramID int64, merchant models.Merchant, amount float64) (string, error) {
	if amount <= 0 {
		return "", domainerrors.ErrInvalidAmount
	}

	provider, ok := s.providers[merchant]
	if !ok {
		return "", domainerrors.ErrInvalidMerchant
	}

	paymentURL, invoiceID, err := provider.CreateInvoice(ctx, telegramID, amount, replenishmentDescription)
	if err != nil {
		return "", err
	}

	replenishment := &models.Replenishment{
		UserID:    telegramID,
		Merchant:  merchant,
		InvoiceID: invoiceID,
		Amount:    amount,
		Status:    models.ReplenishmentStatusPending,
	}
	if err = s.replenishmentRepo.Create(ctx, replenishment); err != nil {
		return "", err
	}

	return paymentURL, nil
}

// Confirm зачисляет баланс и помечает счёт оплаченным. Идемпотентно:
// UpdateStatus меняет строку только из pending, повторный вызов — no-op.
func (s *ReplenishmentSrv) Confirm(ctx context.Context, merchant models.Merchant, invoiceID string) error {
	replenishment, err := s.replenishmentRepo.GetByMerchantInvoiceID(ctx, merchant, invoiceID)
	if err != nil {
		return err
	}

	now := time.Now()
	changed, err := s.replenishmentRepo.UpdateStatus(ctx, replenishment.ID, models.ReplenishmentStatusPaid, &now)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	if err = s.userRepo.UpdateBalance(ctx, replenishment.UserID, replenishment.Amount); err != nil {
		return err
	}
	_ = s.cache.InvalidateUser(ctx, replenishment.UserID)

	s.log.WithFields(logrus.Fields{"user_id": replenishment.UserID, "merchant": merchant, "invoice_id": invoiceID, "amount": replenishment.Amount}).Info("replenishment_service: balance credited")
	return nil
}

func (s *ReplenishmentSrv) Fail(ctx context.Context, merchant models.Merchant, invoiceID string) error {
	replenishment, err := s.replenishmentRepo.GetByMerchantInvoiceID(ctx, merchant, invoiceID)
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = s.replenishmentRepo.UpdateStatus(ctx, replenishment.ID, models.ReplenishmentStatusFailed, &now)
	return err
}

func (s *ReplenishmentSrv) ListUserReplenishments(ctx context.Context, telegramID int64, offset, limit int) ([]models.Replenishment, error) {
	return s.replenishmentRepo.ListByUserID(ctx, telegramID, offset, limit)
}

func (s *ReplenishmentSrv) CountUserReplenishments(ctx context.Context, telegramID int64) (int64, error) {
	return s.replenishmentRepo.CountByUserID(ctx, telegramID)
}

func (s *ReplenishmentSrv) SumUserMerchantAmount(ctx context.Context, telegramID int64, merchant models.Merchant) (float64, error) {
	return s.replenishmentRepo.SumPaidByUserMerchant(ctx, telegramID, merchant)
}

func (s *ReplenishmentSrv) ListAllAdmin(ctx context.Context, userID *int64, offset, limit int) ([]models.ReplenishmentAdminItem, error) {
	return s.replenishmentRepo.ListAllAdmin(ctx, userID, offset, limit)
}

func (s *ReplenishmentSrv) CountAllAdmin(ctx context.Context, userID *int64) (int64, error) {
	return s.replenishmentRepo.CountAllAdmin(ctx, userID)
}
