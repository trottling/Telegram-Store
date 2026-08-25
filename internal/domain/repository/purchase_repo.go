package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type PurchaseRepository interface {
	// CreateBatch вставляет все строки одной покупки (см. PurchaseSrv.Buy)
	// одним запросом, не по одной строке за round-trip.
	CreateBatch(ctx context.Context, purchases []models.Purchase) error
	UpdateStatus(ctx context.Context, purchaseID models.PurchaseID, status models.PurchaseStatus) error
	GetByID(ctx context.Context, id models.PurchaseID) (*models.Purchase, error)
	// GetByBatchID — все строки одного Buy(), в рамках userID.
	GetByBatchID(ctx context.Context, userID models.TelegramID, batchID models.BatchID) ([]models.Purchase, error)
	// StatsByUserID — количество покупок и сумма завершённых, одним агрегатом.
	// Считать это выборкой всех строк нельзя: карточку профиля открывают часто,
	// а у активного покупателя строк тысячи.
	StatsByUserID(ctx context.Context, userID models.TelegramID) (count int64, totalSpent models.Money, err error)
	// CountByProductID — есть ли история покупок (проверка перед удалением товара).
	CountByProductID(ctx context.Context, productID models.ProductID) (int64, error)
	// ListBatchesByUserID/CountBatchesByUserID — по батчам, не по сырым строкам.
	ListBatchesByUserID(ctx context.Context, userID models.TelegramID, offset, limit int) ([]models.PurchaseBatchSummary, error)
	CountBatchesByUserID(ctx context.Context, userID models.TelegramID) (int64, error)

	// ListAll/CountAll/GetAdminByID — межпользовательский вид для админ-панели.
	ListAll(ctx context.Context, filter models.PurchaseAdminFilter, offset, limit int) ([]models.PurchaseAdminItem, error)
	CountAll(ctx context.Context, filter models.PurchaseAdminFilter) (int64, error)
	GetAdminByID(ctx context.Context, id models.PurchaseID) (*models.PurchaseAdminItem, error)
}
