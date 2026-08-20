package postgres

import (
	"context"
	"errors"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewUserRepo(db *gorm.DB, log *zap.SugaredLogger) *UserRepo {
	return &UserRepo{db: db, log: log}
}

func (r *UserRepo) GetByID(ctx context.Context, telegramID int64) (*models.User, error) {
	user, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Where("telegram_id = ?", telegramID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrUserNotFound
		}
		r.log.Errorw("user_repo: get by id failed", "error", err, "telegram_id", telegramID)
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	if err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Create(ctx, user); err != nil {
		r.log.Errorw("user_repo: create failed", "error", err, "telegram_id", user.TelegramID)
		return err
	}
	r.log.Infow("user_repo: user created", "telegram_id", user.TelegramID)
	return nil
}

// Update перезаписывает все колонки — иначе нулевые значения (Select("*") vs
// обычный Updates) молча пропускаются.
func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	_, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).
		Where("telegram_id = ?", user.TelegramID).
		Select("*").
		Updates(ctx, *user)
	if err != nil {
		r.log.Errorw("user_repo: update failed", "error", err, "telegram_id", user.TelegramID)
	}
	return err
}

// UpdateBalance атомарно прибавляет delta; для списания WHERE сама
// проверяет balance >= -delta, защита от гонки при двойном списании.
func (r *UserRepo) UpdateBalance(ctx context.Context, telegramID int64, delta float64) error {
	query := gorm.G[models.User](dbFromCtx(ctx, r.db)).Where("telegram_id = ?", telegramID)
	if delta < 0 {
		query = query.Where("balance >= ?", -delta)
	}

	rows, err := query.Update(ctx, "balance", gorm.Expr("balance + ?", delta))
	if err != nil {
		r.log.Errorw("user_repo: update balance failed", "error", err, "telegram_id", telegramID, "delta", delta)
		return err
	}
	if rows == 0 {
		if delta < 0 {
			return domainerrors.ErrNotEnoughBalance
		}
		return domainerrors.ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context, offset, limit int) ([]models.User, error) {
	users, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Offset(offset).Limit(limit).Order("telegram_id").Find(ctx)
	if err != nil {
		r.log.Errorw("user_repo: list failed", "error", err)
	}
	return users, err
}

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	count, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Count(ctx, "*")
	if err != nil {
		r.log.Errorw("user_repo: count failed", "error", err)
	}
	return count, err
}

func (r *UserRepo) CountReferrals(ctx context.Context, referrerID int64) (int64, error) {
	count, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Where("referrer_id = ?", referrerID).Count(ctx, "*")
	if err != nil {
		r.log.Errorw("user_repo: count referrals failed", "error", err, "referrer_id", referrerID)
	}
	return count, err
}

func (r *UserRepo) ListReferrals(ctx context.Context, referrerID int64, offset, limit int) ([]models.User, error) {
	users, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).
		Where("referrer_id = ?", referrerID).
		Offset(offset).
		Limit(limit).
		Order("telegram_id").
		Find(ctx)
	if err != nil {
		r.log.Errorw("user_repo: list referrals failed", "error", err, "referrer_id", referrerID)
	}
	return users, err
}

// EnsureRootAdminExists выдаёт rootAdminID роль root_admin, создавая
// пользователя при необходимости. Идемпотентно.
func (r *UserRepo) EnsureRootAdminExists(ctx context.Context, rootAdminID int64) error {
	existingUser, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).
		Where("telegram_id = ?", rootAdminID).
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		newUser := &models.User{
			TelegramID: rootAdminID,
			Role:       models.RoleRootAdmin,
		}
		if err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Create(ctx, newUser); err != nil {
			r.log.Errorw("user_repo: failed to create root admin", "error", err, "telegram_id", rootAdminID)
			return err
		}
		r.log.Infow("user_repo: root admin user created", "telegram_id", rootAdminID)
		return nil
	}

	if err != nil {
		r.log.Errorw("user_repo: failed to check root admin existence", "error", err, "telegram_id", rootAdminID)
		return err
	}

	if existingUser.Role == models.RoleRootAdmin {
		r.log.Debugw("user_repo: root admin already has root privileges", "telegram_id", rootAdminID)
		return nil
	}

	_, err = gorm.G[models.User](dbFromCtx(ctx, r.db)).
		Where("telegram_id = ?", rootAdminID).
		Update(ctx, "role", models.RoleRootAdmin)
	if err != nil {
		r.log.Errorw("user_repo: failed to grant root admin privileges", "error", err, "telegram_id", rootAdminID)
		return err
	}

	r.log.Infow("user_repo: granted root admin privileges", "telegram_id", rootAdminID)
	return nil
}
