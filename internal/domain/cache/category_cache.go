package cache

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// CategoryCache — read-through кэш для Category, ключ по parentID (nil — корень).
type CategoryCache interface {
	GetCategoryChildren(ctx context.Context, parentID *int64) ([]models.Category, error)
	SetCategoryChildren(ctx context.Context, parentID *int64, categories []models.Category) error
	InvalidateCategoryChildren(ctx context.Context, parentID *int64) error
}
