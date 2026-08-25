package cache

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// UserCache — read-through кэш для User.
type UserCache interface {
	GetUser(ctx context.Context, telegramID models.TelegramID) (*models.User, error)
	SetUser(ctx context.Context, user *models.User) error
	InvalidateUser(ctx context.Context, telegramID models.TelegramID) error
}
