package cache

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// CategoryCache — кэш видимых детей категории, ключ по parentID (nil —
// корень). Не read-through: пишет только фоновый воркер (см.
// CategorySrv.RefreshCatalogSnapshot), читатели никогда не промахиваются в
// Postgres и поэтому не инвалидируют — устаревшее значение просто доживает
// до следующего тика воркера или своего TTL (categoryChildrenTTL).
type CategoryCache interface {
	GetCategoryChildren(ctx context.Context, parentID *models.CategoryID) ([]models.Category, error)
	SetCategoryChildren(ctx context.Context, parentID *models.CategoryID, categories []models.Category) error
}
