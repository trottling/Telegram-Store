package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// MaxBuyQuantity — потолок количества за одну покупку. Живёт в домене, потому
// что смотрят на него двое: PurchaseSrv.Buy (последнее слово, возвращает
// ErrTooManyProducts) и бот, чтобы отказать сразу при вводе, а не после экрана
// подтверждения. Значение упомянуто и в текстах ErrTooManyProductsMsg.
const MaxBuyQuantity = 20

// PurchaseService принимает Telegram ID покупателя, не внутренний ключ.
type PurchaseService interface {
	// Buy — count строк Purchase за одну транзакцию (одна Purchase на единицу
	// товара). credit != nil, если покупателя пригласили и рефералке начислили
	// процент — вызывающий (бот) сам шлёт уведомление рефереру.
	Buy(ctx context.Context, telegramID models.TelegramID, productID models.ProductID, count int) (purchases []*models.Purchase, credit *models.ReferralCredit, err error)
	// GetUserPurchases/CountUserPurchaseBatches — по батчам, не по сырым строкам.
	GetUserPurchases(ctx context.Context, telegramID models.TelegramID, offset, limit int) ([]models.PurchaseBatchSummary, error)
	CountUserPurchaseBatches(ctx context.Context, telegramID models.TelegramID) (int64, error)
	GetBatch(ctx context.Context, telegramID models.TelegramID, batchID models.BatchID) ([]models.Purchase, error)
	GetUserStats(ctx context.Context, telegramID models.TelegramID) (purchaseCount int, totalSpent models.Money, err error)
	GetByID(ctx context.Context, purchaseID models.PurchaseID) (*models.Purchase, error)

	// ListAllAdmin/CountAllAdmin/GetAdminByID — межпользовательский вид для админ-панели.
	ListAllAdmin(ctx context.Context, filter models.PurchaseAdminFilter, offset, limit int) ([]models.PurchaseAdminItem, error)
	CountAllAdmin(ctx context.Context, filter models.PurchaseAdminFilter) (int64, error)
	GetAdminByID(ctx context.Context, id models.PurchaseID) (*models.PurchaseAdminItem, error)
}
