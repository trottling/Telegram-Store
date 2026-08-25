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
	// попадает в список только если в её поддереве есть товар в наличии
	// (фильтр по HasStock, поддерживаемому RecomputeStock).
	ListChildren(ctx context.Context, parentID *int64) ([]models.Category, error)

	// RecomputeStock пересчитывает HasStock для одной категории: true, если
	// у неё самой есть активный товар в наличии, или это уже true у прямого
	// потомка. Предполагает, что потомки уже пересчитаны (агрегат снизу
	// вверх) — вызывающий идёт по дереву от изменившейся категории к корню.
	// Возвращает, изменилось ли значение, чтобы вызывающий знал, стоит ли
	// продолжать выше.
	RecomputeStock(ctx context.Context, categoryID int64) (bool, error)

	// ListPath — цепочка предков от корня до id (включительно), для хлебных крошек.
	ListPath(ctx context.Context, id int64) ([]models.Category, error)

	// ListAllFlat — все категории без фильтров, для админ-панели.
	ListAllFlat(ctx context.Context) ([]models.Category, error)

	// CountChildren — сколько прямых потомков у parentID (проверка перед удалением).
	CountChildren(ctx context.Context, parentID int64) (int64, error)
}
