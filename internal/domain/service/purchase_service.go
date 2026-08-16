package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// PurchaseService принимает Telegram ID покупателя, не внутренний ключ.
type PurchaseService interface {
	// Buy — count строк Purchase за одну транзакцию (одна Purchase на единицу
	// товара). credit != nil, если покупателя пригласили и рефералке начислили
	// процент — вызывающий (бот) сам шлёт уведомление рефереру.
	Buy(ctx context.Context, telegramID, productID int64, count int) (purchases []*models.Purchase, credit *models.ReferralCredit, err error)
	// GetUserPurchases/CountUserPurchaseBatches — по батчам, не по сырым строкам.
	GetUserPurchases(ctx context.Context, telegramID int64, offset, limit int) ([]models.PurchaseBatchSummary, error)
	CountUserPurchaseBatches(ctx context.Context, telegramID int64) (int64, error)
	GetBatch(ctx context.Context, telegramID int64, batchID string) ([]models.Purchase, error)
	GetUserStats(ctx context.Context, telegramID int64) (purchaseCount int, totalSpent float64, err error)
	GetByID(ctx context.Context, purchaseID int64) (*models.Purchase, error)

	// ListAllAdmin/CountAllAdmin/GetAdminByID — межпользовательский вид для админ-панели.
	ListAllAdmin(ctx context.Context, filter models.PurchaseAdminFilter, offset, limit int) ([]models.PurchaseAdminItem, error)
	CountAllAdmin(ctx context.Context, filter models.PurchaseAdminFilter) (int64, error)
	GetAdminByID(ctx context.Context, id int64) (*models.PurchaseAdminItem, error)
}
