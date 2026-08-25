package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id models.CategoryID) (*models.Category, error)
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id models.CategoryID) error

	// ListPath — цепочка предков от корня до id (включительно), для хлебных крошек.
	ListPath(ctx context.Context, id models.CategoryID) ([]models.Category, error)

	// ListAllFlat — все категории без фильтров: и для админ-панели, и как
	// сырьё для CategorySrv.RefreshCatalogSnapshot (строит дерево сам, в памяти).
	ListAllFlat(ctx context.Context) ([]models.Category, error)

	// CountChildren — сколько прямых потомков у parentID (проверка перед удалением).
	CountChildren(ctx context.Context, parentID models.CategoryID) (int64, error)
}
