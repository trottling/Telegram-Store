package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type CategoryService interface {
	// ListChildren — прямые потомки parentID; nil — верхний уровень каталога.
	ListChildren(ctx context.Context, parentID *models.CategoryID) ([]models.Category, error)
	GetByID(ctx context.Context, id models.CategoryID) (*models.Category, error)
	// ListPath — хлебные крошки от корня до id.
	ListPath(ctx context.Context, id models.CategoryID) ([]models.Category, error)
	// ListProducts — активные товары под categoryID; nil — без категории.
	ListProducts(ctx context.Context, categoryID *models.CategoryID) ([]models.Product, error)

	// ListAllFlat — все категории без фильтров, для админ-панели.
	ListAllFlat(ctx context.Context) ([]models.Category, error)

	// RefreshCatalogSnapshot пересчитывает видимость всего дерева и
	// публикует её в кэш — вызывается только фоновым воркером cmd/bot, не
	// на пути запроса. См. реализацию в internal/service.
	RefreshCatalogSnapshot(ctx context.Context) error
}
