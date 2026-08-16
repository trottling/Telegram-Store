package cache

import (
	"context"

	"github.com/trottling/TG-Store/internal/domain/models"
)

// UserCache — read-through кэш для User.
type UserCache interface {
	GetUser(ctx context.Context, telegramID int64) (*models.User, error)
	SetUser(ctx context.Context, user *models.User) error
	InvalidateUser(ctx context.Context, telegramID int64) error
}
