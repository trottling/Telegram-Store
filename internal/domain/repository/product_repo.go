package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id models.ProductID) (*models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id models.ProductID) error
	ListActive(ctx context.Context) ([]models.Product, error)
	// ListActiveByCategory — активные товары прямо под categoryID; nil — без категории.
	ListActiveByCategory(ctx context.Context, categoryID *models.CategoryID) ([]models.Product, error)

	// единицы товара
	AddItems(ctx context.Context, productID models.ProductID, contents []string) error
	// ReserveItems забирает до count непроданных единиц и сразу помечает их
	// проданными — одним запросом, не count отдельными. Только внутри
	// транзакции: иначе единицы спишутся даже тогда, когда покупка в итоге не
	// состоится. Если в наличии меньше count — возвращает сколько реально
	// нашлось (может быть меньше count или 0), не ошибку: решать, считать ли
	// это нехваткой стока, дело вызывающего (PurchaseSrv.Buy).
	ReserveItems(ctx context.Context, productID models.ProductID, count int) ([]models.ProductItem, error)
	CountAvailableItems(ctx context.Context, productID models.ProductID) (int, error)

	// ListAll/CountAll — админ-листинг: все товары, включая неактивные и распроданные.
	ListAll(ctx context.Context, offset, limit int, categoryID *models.CategoryID) ([]models.ProductAdminSummary, error)
	CountAll(ctx context.Context, categoryID *models.CategoryID) (int64, error)

	// CountByCategoryID — сколько товаров в категории (проверка перед удалением категории).
	CountByCategoryID(ctx context.Context, categoryID models.CategoryID) (int64, error)
}
