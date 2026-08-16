package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type CategoryService interface {
	// ListChildren — прямые потомки parentID; nil — верхний уровень каталога.
	ListChildren(ctx context.Context, parentID *int64) ([]models.Category, error)
	GetByID(ctx context.Context, id int64) (*models.Category, error)
	// ListPath — хлебные крошки от корня до id.
	ListPath(ctx context.Context, id int64) ([]models.Category, error)
	// ListProducts — активные товары под categoryID; nil — без категории.
	ListProducts(ctx context.Context, categoryID *int64) ([]models.Product, error)

	// ListAllFlat — все категории без фильтров, для админ-панели.
	ListAllFlat(ctx context.Context) ([]models.Category, error)
}
