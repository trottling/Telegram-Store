package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// SettingsService — чтение единственных настроек бота (кэшируется). Правки —
// через AdminService.UpdateSettings, чтобы не терять аудит-лог.
type SettingsService interface {
	Get(ctx context.Context) (*models.Settings, error)
}
