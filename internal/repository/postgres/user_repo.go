package postgres

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	domainerrors "github.com/trottling/TG-Store/internal/domain/errors"
	"github.com/trottling/TG-Store/internal/domain/models"
	"gorm.io/gorm"
)

type UserRepo struct {
	db  *gorm.DB
	log *logrus.Logger
}

func NewUserRepo(db *gorm.DB, log *logrus.Logger) *UserRepo {
	return &UserRepo{db: db, log: log}
}

func (r *UserRepo) GetByID(ctx context.Context, telegramID int64) (*models.User, error) {
	user, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Where("telegram_id = ?", telegramID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrUserNotFound
		}
		r.log.WithError(err).WithField("telegram_id", telegramID).Error("user_repo: get by id failed")
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	if err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Create(ctx, user); err != nil {
		r.log.WithError(err).WithField("telegram_id", user.TelegramID).Error("user_repo: create failed")
		return err
	}
	r.log.WithField("telegram_id", user.TelegramID).Info("user_repo: user created")
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
		r.log.WithError(err).WithField("telegram_id", user.TelegramID).Error("user_repo: update failed")
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
		r.log.WithError(err).WithFields(logrus.Fields{"telegram_id": telegramID, "delta": delta}).Error("user_repo: update balance failed")
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
		r.log.WithError(err).Error("user_repo: list failed")
	}
	return users, err
}

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	count, err := gorm.G[models.User](dbFromCtx(ctx, r.db)).Count(ctx, "*")
	if err != nil {
		r.log.WithError(err).Error("user_repo: count failed")
	}
	return count, err
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
			r.log.WithError(err).WithField("telegram_id", rootAdminID).Error("user_repo: failed to create root admin")
			return err
		}
		r.log.WithField("telegram_id", rootAdminID).Info("user_repo: root admin user created")
		return nil
	}

	if err != nil {
		r.log.WithError(err).WithField("telegram_id", rootAdminID).Error("user_repo: failed to check root admin existence")
		return err
	}

	if existingUser.Role == models.RoleRootAdmin {
		r.log.WithField("telegram_id", rootAdminID).Debug("user_repo: root admin already has root privileges")
		return nil
	}

	_, err = gorm.G[models.User](dbFromCtx(ctx, r.db)).
		Where("telegram_id = ?", rootAdminID).
		Update(ctx, "role", models.RoleRootAdmin)
	if err != nil {
		r.log.WithError(err).WithField("telegram_id", rootAdminID).Error("user_repo: failed to grant root admin privileges")
		return err
	}

	r.log.WithField("telegram_id", rootAdminID).Info("user_repo: granted root admin privileges")
	return nil
}
