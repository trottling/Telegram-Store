package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// userRecord — persistence-модель User для GORM. Отдельно от models.User,
// потому что у того balance/role неэкспортируемые (см. его doc-комментарий) —
// GORM не может ни прочитать, ни записать неэкспортируемое поле. toDomain/
// fromDomain — маппинг на границе репозитория, единственном месте, которое о
// userRecord знает.
type userRecord struct {
	TelegramID       models.TelegramID  `gorm:"primaryKey"`
	Username         string             `gorm:"size:32"`
	Balance          models.Money       `gorm:"type:decimal(12,2);default:0;not null"`
	Role             models.Role        `gorm:"type:varchar(20);default:'user';not null"`
	CreatedAt        time.Time          `gorm:""`
	UpdatedAt        time.Time          `gorm:""`
	DeletedAt        gorm.DeletedAt     `gorm:"index"`
	ReferrerID       *models.TelegramID `gorm:"index"`
	ReferralsEnabled bool               `gorm:"default:true;not null"`
	Language         string             `gorm:"size:8;default:'ru';not null"`
}

func (userRecord) TableName() string { return "users" }

func (r userRecord) toDomain() *models.User {
	return models.HydrateUser(r.TelegramID, r.Username, r.Balance, r.Role, r.CreatedAt, r.UpdatedAt, r.ReferrerID, r.ReferralsEnabled, r.Language)
}

func fromDomain(u *models.User) userRecord {
	return userRecord{
		TelegramID:       u.TelegramID,
		Username:         u.Username,
		Balance:          u.Balance(),
		Role:             u.Role(),
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
		ReferrerID:       u.ReferrerID,
		ReferralsEnabled: u.ReferralsEnabled,
		Language:         u.Language,
	}
}

type UserRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewUserRepo(db *gorm.DB, log *zap.SugaredLogger) *UserRepo {
	return &UserRepo{db: db, log: log}
}

func (r *UserRepo) GetByID(ctx context.Context, telegramID models.TelegramID) (*models.User, error) {
	record, err := gorm.G[userRecord](dbFromCtx(ctx, r.db)).Where("telegram_id = ?", telegramID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrUserNotFound
		}
		r.log.Errorw("user_repo: get by id failed", "error", err, "telegram_id", telegramID)
		return nil, err
	}
	return record.toDomain(), nil
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	record := fromDomain(user)
	if err := gorm.G[userRecord](dbFromCtx(ctx, r.db)).Create(ctx, &record); err != nil {
		r.log.Errorw("user_repo: create failed", "error", err, "telegram_id", user.TelegramID)
		return err
	}
	*user = *record.toDomain()
	r.log.Infow("user_repo: user created", "telegram_id", user.TelegramID)
	return nil
}

// Update перезаписывает все колонки — иначе нулевые значения (Select("*") vs
// обычный Updates) молча пропускаются.
func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	record := fromDomain(user)
	_, err := gorm.G[userRecord](dbFromCtx(ctx, r.db)).
		Where("telegram_id = ?", user.TelegramID).
		Select("*").
		Updates(ctx, record)
	if err != nil {
		r.log.Errorw("user_repo: update failed", "error", err, "telegram_id", user.TelegramID)
	}
	return err
}

// UpdateBalance атомарно прибавляет delta; для списания WHERE сама
// проверяет balance >= -delta, защита от гонки при двойном списании.
func (r *UserRepo) UpdateBalance(ctx context.Context, telegramID models.TelegramID, delta decimal.Decimal) error {
	query := gorm.G[userRecord](dbFromCtx(ctx, r.db)).Where("telegram_id = ?", telegramID)
	negative := delta.IsNegative()
	if negative {
		query = query.Where("balance >= ?", delta.Neg())
	}

	rows, err := query.Update(ctx, "balance", gorm.Expr("balance + ?", delta))
	if err != nil {
		r.log.Errorw("user_repo: update balance failed", "error", err, "telegram_id", telegramID, "delta", delta)
		return err
	}
	if rows == 0 {
		if negative {
			return domainerrors.ErrNotEnoughBalance
		}
		return domainerrors.ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context, offset, limit int) ([]models.User, error) {
	records, err := gorm.G[userRecord](dbFromCtx(ctx, r.db)).Offset(offset).Limit(limit).Order("telegram_id").Find(ctx)
	if err != nil {
		r.log.Errorw("user_repo: list failed", "error", err)
		return nil, err
	}
	return toDomainSlice(records), nil
}

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	count, err := gorm.G[userRecord](dbFromCtx(ctx, r.db)).Count(ctx, "*")
	if err != nil {
		r.log.Errorw("user_repo: count failed", "error", err)
	}
	return count, err
}

func (r *UserRepo) CountReferrals(ctx context.Context, referrerID models.TelegramID) (int64, error) {
	count, err := gorm.G[userRecord](dbFromCtx(ctx, r.db)).Where("referrer_id = ?", referrerID).Count(ctx, "*")
	if err != nil {
		r.log.Errorw("user_repo: count referrals failed", "error", err, "referrer_id", referrerID)
	}
	return count, err
}

func (r *UserRepo) ListReferrals(ctx context.Context, referrerID models.TelegramID, offset, limit int) ([]models.User, error) {
	records, err := gorm.G[userRecord](dbFromCtx(ctx, r.db)).
		Where("referrer_id = ?", referrerID).
		Offset(offset).
		Limit(limit).
		Order("telegram_id").
		Find(ctx)
	if err != nil {
		r.log.Errorw("user_repo: list referrals failed", "error", err, "referrer_id", referrerID)
		return nil, err
	}
	return toDomainSlice(records), nil
}

// EnsureRootAdminExists выдаёт rootAdminID роль root_admin, создавая
// пользователя при необходимости. Идемпотентно.
func (r *UserRepo) EnsureRootAdminExists(ctx context.Context, rootAdminID models.TelegramID, rootAdminUsername string) error {
	existing, err := gorm.G[userRecord](dbFromCtx(ctx, r.db)).
		Where("telegram_id = ?", rootAdminID).
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		newRecord := &userRecord{
			Username:   rootAdminUsername,
			TelegramID: rootAdminID,
			Role:       models.RoleRootAdmin,
		}
		if err = gorm.G[userRecord](dbFromCtx(ctx, r.db)).Create(ctx, newRecord); err != nil {
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

	if existing.Role == models.RoleRootAdmin {
		r.log.Debugw("user_repo: root admin already has root privileges", "telegram_id", rootAdminID)
		return nil
	}

	_, err = gorm.G[userRecord](dbFromCtx(ctx, r.db)).
		Where("telegram_id = ?", rootAdminID).
		Update(ctx, "role", models.RoleRootAdmin)
	if err != nil {
		r.log.Errorw("user_repo: failed to grant root admin privileges", "error", err, "telegram_id", rootAdminID)
		return err
	}

	r.log.Infow("user_repo: granted root admin privileges", "telegram_id", rootAdminID)
	return nil
}

func toDomainSlice(records []userRecord) []models.User {
	users := make([]models.User, len(records))
	for i, record := range records {
		users[i] = *record.toDomain()
	}
	return users
}
