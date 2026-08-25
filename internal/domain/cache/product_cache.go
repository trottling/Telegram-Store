package cache

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ProductCache — read-through кэш для Product.
type ProductCache interface {
	GetActiveProducts(ctx context.Context) ([]models.Product, error)
	SetActiveProducts(ctx context.Context, products []models.Product) error
	InvalidateActiveProducts(ctx context.Context) error

	GetProduct(ctx context.Context, productID models.ProductID) (*models.Product, error)
	SetProduct(ctx context.Context, product *models.Product) error
	InvalidateProduct(ctx context.Context, productID models.ProductID) error

	GetProductAvailableCount(ctx context.Context, productID models.ProductID) (int, error)
	SetProductAvailableCount(ctx context.Context, productID models.ProductID, count int) error
	InvalidateProductAvailableCount(ctx context.Context, productID models.ProductID) error
}
