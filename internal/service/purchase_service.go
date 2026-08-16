package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	domainerrors "github.com/trottling/TG-Store/internal/domain/errors"
	"github.com/trottling/TG-Store/internal/domain/models"
	"github.com/trottling/TG-Store/internal/domain/repository"
)

// maxBuyQuantity ограничивает покупку за один вызов.
const maxBuyQuantity = 20

type PurchaseSrv struct {
	userRepo     repository.UserRepository
	productRepo  repository.ProductRepository
	purchaseRepo repository.PurchaseRepository
	categoryRepo repository.CategoryRepository
	transactor   repository.Transactor
	cache        multiCache
	log          *logrus.Logger
}

func NewPurchaseSrv(
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	purchaseRepo repository.PurchaseRepository,
	categoryRepo repository.CategoryRepository,
	transactor repository.Transactor,
	cache multiCache,
	log *logrus.Logger,
) *PurchaseSrv {
	return &PurchaseSrv{
		userRepo:     userRepo,
		productRepo:  productRepo,
		purchaseRepo: purchaseRepo,
		categoryRepo: categoryRepo,
		transactor:   transactor,
		cache:        cache,
		log:          log,
	}
}

// Buy покупает count единиц productID для telegramID в одной транзакции.
func (s *PurchaseSrv) Buy(ctx context.Context, telegramID, productID int64, count int) ([]*models.Purchase, error) {
	logCtx := s.log.WithFields(logrus.Fields{"telegram_id": telegramID, "product_id": productID, "count": count})

	if count <= 0 {
		return nil, domainerrors.ErrInvalidQuantity
	}
	if count > maxBuyQuantity {
		return nil, domainerrors.ErrTooManyProducts
	}

	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if !product.IsActive {
		logCtx.Warn("purchase_service: buy rejected, product inactive")
		return nil, domainerrors.ErrProductInactive
	}

	totalPrice := product.Price * float64(count)
	if user.Balance < totalPrice {
		logCtx.WithField("balance", user.Balance).Warn("purchase_service: buy rejected, not enough balance")
		return nil, domainerrors.ErrNotEnoughBalance
	}

	// Один batchID на все строки этого вызова — история группирует по нему.
	batchID := uuid.NewString()

	purchases := make([]*models.Purchase, 0, count)
	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		for range count {
			item, itemErr := s.productRepo.GetAvailableItem(ctx, productID)
			if itemErr != nil {
				return itemErr
			}

			now := time.Now()
			p := &models.Purchase{
				UserID:      telegramID,
				ProductID:   productID,
				ItemID:      &item.ID,
				BatchID:     batchID,
				Amount:      product.Price,
				Status:      models.PurchaseStatusCompleted,
				CompletedAt: &now,
			}
			if createErr := s.purchaseRepo.Create(ctx, p); createErr != nil {
				return createErr
			}
			if markErr := s.productRepo.MarkItemSold(ctx, item.ID, p.ID); markErr != nil {
				return markErr
			}

			p.Item = item
			p.Product = *product
			purchases = append(purchases, p)
		}

		return s.userRepo.UpdateBalance(ctx, telegramID, -totalPrice)
	})
	if err != nil {
		if errors.Is(err, domainerrors.ErrProductOutOfStock) {
			logCtx.Warn("purchase_service: buy rejected, out of stock")
		} else {
			logCtx.WithError(err).Error("purchase_service: buy transaction failed")
		}
		return nil, err
	}

	_ = s.cache.InvalidateUser(ctx, telegramID)
	_ = s.cache.InvalidateProductAvailableCount(ctx, productID)
	// Листинг скрывает распроданные товары — нужно сбросить и его тоже.
	_ = s.cache.InvalidateActiveProducts(ctx)
	if product.CategoryID != nil {
		invalidateCategoryAncestorChain(ctx, s.categoryRepo, s.cache, *product.CategoryID)
	}
	logCtx.WithField("total_price", totalPrice).Info("purchase_service: purchase completed")
	return purchases, nil
}

func (s *PurchaseSrv) GetUserPurchases(ctx context.Context, telegramID int64, offset, limit int) ([]models.PurchaseBatchSummary, error) {
	return s.purchaseRepo.ListBatchesByUserID(ctx, telegramID, offset, limit)
}

func (s *PurchaseSrv) CountUserPurchaseBatches(ctx context.Context, telegramID int64) (int64, error) {
	return s.purchaseRepo.CountBatchesByUserID(ctx, telegramID)
}

func (s *PurchaseSrv) GetBatch(ctx context.Context, telegramID int64, batchID string) ([]models.Purchase, error) {
	return s.purchaseRepo.GetByBatchID(ctx, telegramID, batchID)
}

func (s *PurchaseSrv) GetUserStats(ctx context.Context, telegramID int64) (purchaseCount int, totalSpent float64, err error) {
	count, err := s.purchaseRepo.CountByUserID(ctx, telegramID)
	if err != nil {
		return 0, 0, err
	}
	if count == 0 {
		return 0, 0, nil
	}

	purchases, err := s.purchaseRepo.GetByUserID(ctx, telegramID, 0, int(count))
	if err != nil {
		return 0, 0, err
	}

	for _, p := range purchases {
		if p.Status == models.PurchaseStatusCompleted {
			totalSpent += p.Amount
		}
	}
	return int(count), totalSpent, nil
}

func (s *PurchaseSrv) GetByID(ctx context.Context, purchaseID int64) (*models.Purchase, error) {
	return s.purchaseRepo.GetByID(ctx, purchaseID)
}

// ListAllAdmin/CountAllAdmin/GetAdminByID — межпользовательский вид для админ-панели.
func (s *PurchaseSrv) ListAllAdmin(ctx context.Context, filter models.PurchaseAdminFilter, offset, limit int) ([]models.PurchaseAdminItem, error) {
	return s.purchaseRepo.ListAll(ctx, filter, offset, limit)
}

func (s *PurchaseSrv) CountAllAdmin(ctx context.Context, filter models.PurchaseAdminFilter) (int64, error) {
	return s.purchaseRepo.CountAll(ctx, filter)
}

func (s *PurchaseSrv) GetAdminByID(ctx context.Context, id int64) (*models.PurchaseAdminItem, error) {
	return s.purchaseRepo.GetAdminByID(ctx, id)
}
