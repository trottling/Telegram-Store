package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// SettingsRepository — единственная строка настроек бота (models.SettingsID).
type SettingsRepository interface {
	Get(ctx context.Context) (*models.Settings, error)
	Update(ctx context.Context, settings *models.Settings) error

	// EnsureExists создаёт строку настроек с дефолтами, если её ещё нет.
	// Идемпотентно, вызывается из cmd/migrate.
	EnsureExists(ctx context.Context, defaults *models.Settings) error
}
