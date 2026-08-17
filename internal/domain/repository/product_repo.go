package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id int64) (*models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id int64) error
	ListActive(ctx context.Context) ([]models.Product, error)
	// ListActiveByCategory — активные товары прямо под categoryID; nil — без категории.
	ListActiveByCategory(ctx context.Context, categoryID *int64) ([]models.Product, error)

	// единицы товара
	AddItems(ctx context.Context, productID int64, contents []string) error
	// ReserveItem забирает одну непроданную единицу и сразу помечает её
	// проданной. Только внутри транзакции: иначе единица спишется даже тогда,
	// когда покупка в итоге не состоится.
	ReserveItem(ctx context.Context, productID int64) (*models.ProductItem, error)
	CountAvailableItems(ctx context.Context, productID int64) (int, error)

	// ListAll/CountAll — админ-листинг: все товары, включая неактивные и распроданные.
	ListAll(ctx context.Context, offset, limit int, categoryID *int64) ([]models.ProductAdminSummary, error)
	CountAll(ctx context.Context, categoryID *int64) (int64, error)

	// CountByCategoryID — сколько товаров в категории (проверка перед удалением категории).
	CountByCategoryID(ctx context.Context, categoryID int64) (int64, error)
}
