package service

import (
	"context"
	"errors"
	"time"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	domainservice "github.com/trottling/Telegram-Store/internal/domain/service"
	"go.uber.org/zap"
)

type PurchaseSrv struct {
	userRepo          repository.UserRepository
	productRepo       repository.ProductRepository
	purchaseRepo      repository.PurchaseRepository
	replenishmentRepo repository.ReplenishmentRepository
	transactor        repository.Transactor
	settingsService   domainservice.SettingsService
	cache             MultiCache
	log               *zap.SugaredLogger
}

func NewPurchaseSrv(
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	purchaseRepo repository.PurchaseRepository,
	replenishmentRepo repository.ReplenishmentRepository,
	transactor repository.Transactor,
	settingsService domainservice.SettingsService,
	cache MultiCache,
	log *zap.SugaredLogger,
) *PurchaseSrv {
	return &PurchaseSrv{
		userRepo:          userRepo,
		productRepo:       productRepo,
		purchaseRepo:      purchaseRepo,
		replenishmentRepo: replenishmentRepo,
		transactor:        transactor,
		settingsService:   settingsService,
		cache:             cache,
		log:               log,
	}
}

// Buy покупает count единиц productID для telegramID в одной транзакции.
func (s *PurchaseSrv) Buy(ctx context.Context, telegramID models.TelegramID, productID models.ProductID, count int) ([]*models.Purchase, *models.ReferralCredit, error) {
	logCtx := s.log.With("telegram_id", telegramID, "product_id", productID, "count", count)

	if count <= 0 {
		return nil, nil, domainerrors.ErrInvalidQuantity
	}
	if count > domainservice.MaxBuyQuantity {
		return nil, nil, domainerrors.ErrTooManyProducts
	}

	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return nil, nil, err
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, nil, err
	}
	if !product.IsActive {
		logCtx.Warn("purchase_service: buy rejected, product inactive")
		return nil, nil, domainerrors.ErrProductInactive
	}

	totalPrice := product.Price.Mul(count)
	// Ниже totalPrice пересчитывается внутри транзакции по свежей цене; списание
	// всё равно защищено guard'ом balance >= сумма в UpdateBalance.
	if user.Balance().LessThan(totalPrice) {
		logCtx.Warnw("purchase_service: buy rejected, not enough balance", "balance", user.Balance())
		return nil, nil, domainerrors.ErrNotEnoughBalance
	}

	// Один batchID на все строки этого вызова — история группирует по нему.
	batchID := models.NewBatchID()

	purchases := make([]*models.Purchase, 0, count)
	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		// Товар перечитываем внутри транзакции: между проверками выше и этим
		// моментом админ мог его деактивировать или поменять цену. Проверки
		// снаружи остаются как быстрый отказ, последнее слово — здесь.
		var txErr error
		if product, txErr = s.productRepo.GetByID(ctx, productID); txErr != nil {
			return txErr
		}
		if !product.IsActive {
			return domainerrors.ErrProductInactive
		}
		totalPrice = product.Price.Mul(count)

		// Один запрос на весь count вместо count отдельных: ReserveItems сам
		// резервирует и помечает проданными до count единиц атомарно (тот же
		// FOR UPDATE SKIP LOCKED, что и раньше у поштучного ReserveItem —
		// см. её doc-комментарий). Меньше items, чем count, — стока не
		// хватило; ошибку не возвращаем нарочно (это не ошибка запроса), а
		// проверяем длину сами, чтобы отличить от продукта без единиц вовсе.
		items, itemsErr := s.productRepo.ReserveItems(ctx, productID, count)
		if itemsErr != nil {
			return itemsErr
		}
		if len(items) < count {
			return domainerrors.ErrProductOutOfStock
		}

		now := time.Now()
		rows := make([]models.Purchase, count)
		for i := range items {
			rows[i] = models.Purchase{
				UserID:      telegramID,
				ProductID:   productID,
				ItemID:      &items[i].ID,
				BatchID:     batchID,
				Amount:      product.Price,
				Status:      models.PurchaseStatusCompleted,
				CompletedAt: &now,
			}
		}
		if createErr := s.purchaseRepo.CreateBatch(ctx, rows); createErr != nil {
			return createErr
		}

		for i := range rows {
			p := &rows[i]
			p.Item = &items[i]
			p.Product = *product
			purchases = append(purchases, p)
		}

		return s.userRepo.UpdateBalance(ctx, telegramID, totalPrice.Decimal().Neg())
	})
	if err != nil {
		if errors.Is(err, domainerrors.ErrProductOutOfStock) {
			logCtx.Warn("purchase_service: buy rejected, out of stock")
		} else {
			logCtx.Errorw("purchase_service: buy transaction failed", "error", err)
		}
		return nil, nil, err
	}

	// Реферальный бонус — после коммита покупки, отдельной транзакцией. Внутри
	// той же он быть не может: любая упавшая там команда переводит транзакцию
	// Postgres в aborted, COMMIT выполняет ROLLBACK, и pgx возвращает
	// ErrTxCommitRollback — то есть сбой необязательного бонуса отменял бы
	// полностью оплаченную покупку. А best-effort в этом и состоит, что не
	// должен её ронять.
	credit := s.creditReferral(ctx, user.ReferrerID, totalPrice)

	logInvalidation(s.log, s.cache.InvalidateUser(ctx, telegramID), "user", telegramID)
	_ = s.cache.InvalidateProductAvailableCount(ctx, productID)
	// Листинг скрывает распроданные товары — нужно сбросить и его тоже.
	_ = s.cache.InvalidateActiveProducts(ctx)
	if credit != nil {
		logInvalidation(s.log, s.cache.InvalidateUser(ctx, credit.ReferrerID), "user", credit.ReferrerID)
	}
	logCtx.Infow("purchase_service: purchase completed", "total_price", totalPrice)
	return purchases, credit, nil
}

// creditReferral — best-effort: реферальная система выключена, у реферера
// отключены начисления, либо процент/сумма нулевые — молча ничего не делает.
// Пишет и Replenishment (Merchant=referral), чтобы начисление было видно в
// "Мои пополнения" реферера.
//
// Вызывается после коммита покупки, отдельной транзакцией — см. Buy. Свои две
// записи держит вместе, чтобы у реферера не оказалось начисленных денег без
// строки в истории пополнений.
func (s *PurchaseSrv) creditReferral(ctx context.Context, referrerID *models.TelegramID, purchaseAmount models.Money) *models.ReferralCredit {
	if referrerID == nil {
		return nil
	}

	settings, err := s.settingsService.Get(ctx)
	if err != nil || !settings.Referral.Enabled || settings.Referral.Percent <= 0 {
		return nil
	}

	referrer, err := s.userRepo.GetByID(ctx, *referrerID)
	if err != nil || !referrer.ReferralsEnabled {
		return nil
	}

	credit := purchaseAmount.Percent(settings.Referral.Percent)
	if credit.IsZero() {
		return nil
	}

	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if updErr := s.userRepo.UpdateBalance(ctx, referrer.TelegramID, credit.Decimal()); updErr != nil {
			return updErr
		}

		now := time.Now()
		return s.replenishmentRepo.Create(ctx, &models.Replenishment{
			UserID:      referrer.TelegramID,
			Merchant:    models.MerchantReferral,
			Amount:      credit,
			Status:      models.ReplenishmentStatusPaid,
			CompletedAt: &now,
		})
	})
	if err != nil {
		s.log.Errorw("purchase_service: failed to credit referral", "error", err, "referrer_id", referrer.TelegramID)
		return nil
	}

	return &models.ReferralCredit{ReferrerID: referrer.TelegramID, Amount: credit}
}

func (s *PurchaseSrv) GetUserPurchases(ctx context.Context, telegramID models.TelegramID, offset, limit int) ([]models.PurchaseBatchSummary, error) {
	return s.purchaseRepo.ListBatchesByUserID(ctx, telegramID, offset, limit)
}

func (s *PurchaseSrv) CountUserPurchaseBatches(ctx context.Context, telegramID models.TelegramID) (int64, error) {
	return s.purchaseRepo.CountBatchesByUserID(ctx, telegramID)
}

func (s *PurchaseSrv) GetBatch(ctx context.Context, telegramID models.TelegramID, batchID models.BatchID) ([]models.Purchase, error) {
	return s.purchaseRepo.GetByBatchID(ctx, telegramID, batchID)
}

func (s *PurchaseSrv) GetUserStats(ctx context.Context, telegramID models.TelegramID) (purchaseCount int, totalSpent models.Money, err error) {
	count, totalSpent, err := s.purchaseRepo.StatsByUserID(ctx, telegramID)
	if err != nil {
		return 0, models.Money{}, err
	}
	return int(count), totalSpent, nil
}

func (s *PurchaseSrv) GetByID(ctx context.Context, purchaseID models.PurchaseID) (*models.Purchase, error) {
	return s.purchaseRepo.GetByID(ctx, purchaseID)
}

// ListAllAdmin/CountAllAdmin/GetAdminByID — межпользовательский вид для админ-панели.
func (s *PurchaseSrv) ListAllAdmin(ctx context.Context, filter models.PurchaseAdminFilter, offset, limit int) ([]models.PurchaseAdminItem, error) {
	return s.purchaseRepo.ListAll(ctx, filter, offset, limit)
}

func (s *PurchaseSrv) CountAllAdmin(ctx context.Context, filter models.PurchaseAdminFilter) (int64, error) {
	return s.purchaseRepo.CountAll(ctx, filter)
}

func (s *PurchaseSrv) GetAdminByID(ctx context.Context, id models.PurchaseID) (*models.PurchaseAdminItem, error) {
	return s.purchaseRepo.GetAdminByID(ctx, id)
}
