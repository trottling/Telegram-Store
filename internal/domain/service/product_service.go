package service

import (
	"context"

	"github.com/trottling/TG-Store/internal/domain/models"
)

type ProductService interface {
	// ListAvailable — все активные товары, плоский список.
	ListAvailable(ctx context.Context) ([]models.Product, error)
	GetByID(ctx context.Context, id int64) (*models.Product, error)
	GetAvailableCount(ctx context.Context, productID int64) (int, error)

	// ListAllAdmin/CountAllAdmin — админ-листинг: все товары, включая неактивные и распроданные.
	ListAllAdmin(ctx context.Context, offset, limit int, categoryID *int64) ([]models.ProductAdminSummary, error)
	CountAllAdmin(ctx context.Context, categoryID *int64) (int64, error)
}
