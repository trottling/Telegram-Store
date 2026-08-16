package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type UserService interface {
	// GetOrCreate: referrerID — из payload "/start <id>", учитывается только
	// при создании строки (см. UserSrv.GetOrCreate) — уже существующий
	// пользователь рефералом не становится, даже если пришёл по ссылке.
	GetOrCreate(ctx context.Context, telegramID int64, username string, referrerID *int64) (*models.User, error)
	GetProfile(ctx context.Context, telegramID int64) (*models.User, error)
	// RefreshProfile — то же, что GetProfile, но всегда читает Postgres напрямую, минуя кэш, и обновляет кэш свежими данными.
	RefreshProfile(ctx context.Context, telegramID int64) (*models.User, error)
	IsBanned(ctx context.Context, telegramID int64) (bool, error)

	// CountReferrals/ListReferrals — кого пригласил telegramID (для админ-панели — таблица рефералов).
	CountReferrals(ctx context.Context, telegramID int64) (int64, error)
	ListReferrals(ctx context.Context, telegramID int64, offset, limit int) ([]models.User, error)

	// ListAdmin/CountAdmin — все пользователи для админ-панели, мимо кэша.
	ListAdmin(ctx context.Context, offset, limit int) ([]models.User, error)
	CountAdmin(ctx context.Context) (int64, error)
}
