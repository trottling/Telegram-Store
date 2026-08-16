package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type PurchaseRepository interface {
	Create(ctx context.Context, purchase *models.Purchase) error
	UpdateStatus(ctx context.Context, purchaseID int64, status models.PurchaseStatus) error
	GetByID(ctx context.Context, id int64) (*models.Purchase, error)
	// GetByBatchID — все строки одного Buy(), в рамках userID.
	GetByBatchID(ctx context.Context, userID int64, batchID string) ([]models.Purchase, error)
	GetByUserID(ctx context.Context, userID int64, offset, limit int) ([]models.Purchase, error)
	CountByUserID(ctx context.Context, userID int64) (int64, error)
	// CountByProductID — есть ли история покупок (проверка перед удалением товара).
	CountByProductID(ctx context.Context, productID int64) (int64, error)
	// ListBatchesByUserID/CountBatchesByUserID — по батчам, не по сырым строкам.
	ListBatchesByUserID(ctx context.Context, userID int64, offset, limit int) ([]models.PurchaseBatchSummary, error)
	CountBatchesByUserID(ctx context.Context, userID int64) (int64, error)

	// ListAll/CountAll/GetAdminByID — межпользовательский вид для админ-панели.
	ListAll(ctx context.Context, filter models.PurchaseAdminFilter, offset, limit int) ([]models.PurchaseAdminItem, error)
	CountAll(ctx context.Context, filter models.PurchaseAdminFilter) (int64, error)
	GetAdminByID(ctx context.Context, id int64) (*models.PurchaseAdminItem, error)
}
