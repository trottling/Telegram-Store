package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type UserService interface {
	GetOrCreate(ctx context.Context, telegramID int64, username string) (*models.User, error)
	GetProfile(ctx context.Context, telegramID int64) (*models.User, error)
	IsBanned(ctx context.Context, telegramID int64) (bool, error)

	// ListAdmin/CountAdmin — все пользователи для админ-панели, мимо кэша.
	ListAdmin(ctx context.Context, offset, limit int) ([]models.User, error)
	CountAdmin(ctx context.Context) (int64, error)
}
