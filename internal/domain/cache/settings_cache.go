package cache

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// SettingsCache — read-through кэш для единственной строки настроек бота.
type SettingsCache interface {
	GetSettings(ctx context.Context) (*models.Settings, error)
	SetSettings(ctx context.Context, settings *models.Settings) error
	InvalidateSettings(ctx context.Context) error
}
