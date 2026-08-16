package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id int64) (*models.Category, error)
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id int64) error

	// ListChildren — прямые потомки parentID (nil — корень). Категория
	// попадает в список только если в её поддереве есть товар в наличии.
	ListChildren(ctx context.Context, parentID *int64) ([]models.Category, error)

	// ListPath — цепочка предков от корня до id (включительно), для хлебных крошек.
	ListPath(ctx context.Context, id int64) ([]models.Category, error)

	// ListAllFlat — все категории без фильтров, для админ-панели.
	ListAllFlat(ctx context.Context) ([]models.Category, error)

	// CountChildren — сколько прямых потомков у parentID (проверка перед удалением).
	CountChildren(ctx context.Context, parentID int64) (int64, error)
}
