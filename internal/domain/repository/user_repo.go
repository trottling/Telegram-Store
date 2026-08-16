package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// UserRepository ключуется по Telegram ID — отдельного внутреннего ID нет.
type UserRepository interface {
	GetByID(ctx context.Context, telegramID int64) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	UpdateBalance(ctx context.Context, telegramID int64, delta float64) error
	List(ctx context.Context, offset, limit int) ([]models.User, error)
	Count(ctx context.Context) (int64, error)

	// CountReferrals — сколько пользователей пригласил referrerID (User.ReferrerID = referrerID).
	CountReferrals(ctx context.Context, referrerID int64) (int64, error)

	// EnsureRootAdminExists выдаёт rootAdminID роль root_admin, создавая
	// пользователя при необходимости. Идемпотентно, вызывается из cmd/migrate.
	EnsureRootAdminExists(ctx context.Context, rootAdminID int64) error
}
