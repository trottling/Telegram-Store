package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type ProductService interface {
	// ListAvailable — все активные товары, плоский список.
	ListAvailable(ctx context.Context) ([]models.Product, error)
	GetByID(ctx context.Context, id models.ProductID) (*models.Product, error)
	GetAvailableCount(ctx context.Context, productID models.ProductID) (int, error)

	// ListAllAdmin/CountAllAdmin — админ-листинг: все товары, включая неактивные и распроданные.
	ListAllAdmin(ctx context.Context, offset, limit int, categoryID *models.CategoryID) ([]models.ProductAdminSummary, error)
	CountAllAdmin(ctx context.Context, categoryID *models.CategoryID) (int64, error)
}
