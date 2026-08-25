package repository

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// UserRepository ключуется по Telegram ID — отдельного внутреннего ID нет.
type UserRepository interface {
	GetByID(ctx context.Context, telegramID int64) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	// UpdateBalance — атомарный UPDATE ... SET balance = balance + delta
	// WHERE balance >= -delta; это единственная безопасная при конкуренции
	// защита от ухода в минус, поэтому delta — знаковая корректировка
	// (decimal.Decimal), а не Money: Money не бывает отрицательным.
	UpdateBalance(ctx context.Context, telegramID int64, delta decimal.Decimal) error
	List(ctx context.Context, offset, limit int) ([]models.User, error)
	Count(ctx context.Context) (int64, error)

	// CountReferrals/ListReferrals — кого пригласил referrerID (User.ReferrerID = referrerID).
	CountReferrals(ctx context.Context, referrerID int64) (int64, error)
	ListReferrals(ctx context.Context, referrerID int64, offset, limit int) ([]models.User, error)

	// EnsureRootAdminExists выдаёт rootAdminID роль root_admin, создавая
	// пользователя при необходимости. Идемпотентно, вызывается из cmd/migrate.
	EnsureRootAdminExists(ctx context.Context, rootAdminID int64) error
}
