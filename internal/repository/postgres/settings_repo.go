package postgres

import (
	"context"
	"errors"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SettingsRepo struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewSettingsRepo(db *gorm.DB, log *zap.SugaredLogger) *SettingsRepo {
	return &SettingsRepo{db: db, log: log}
}

func (r *SettingsRepo) Get(ctx context.Context) (*models.Settings, error) {
	settings, err := gorm.G[models.Settings](dbFromCtx(ctx, r.db)).Where("id = ?", models.SettingsID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrSettingsNotFound
		}
		r.log.Errorw("settings_repo: get failed", "error", err)
		return nil, err
	}
	return &settings, nil
}

// Update перезаписывает все колонки, как UserRepo.Update — иначе нулевые
// значения молча пропускаются обычным Updates.
func (r *SettingsRepo) Update(ctx context.Context, settings *models.Settings) error {
	settings.ID = models.SettingsID
	_, err := gorm.G[models.Settings](dbFromCtx(ctx, r.db)).
		Where("id = ?", models.SettingsID).
		Select("*").
		Updates(ctx, *settings)
	if err != nil {
		r.log.Errorw("settings_repo: update failed", "error", err)
	}
	return err
}

// EnsureExists создаёт единственную строку настроек с дефолтами, если её
// ещё нет. Идемпотентно.
func (r *SettingsRepo) EnsureExists(ctx context.Context, defaults *models.Settings) error {
	_, err := gorm.G[models.Settings](dbFromCtx(ctx, r.db)).Where("id = ?", models.SettingsID).First(ctx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Errorw("settings_repo: failed to check settings existence", "error", err)
		return err
	}

	defaults.ID = models.SettingsID
	if err = gorm.G[models.Settings](dbFromCtx(ctx, r.db)).Create(ctx, defaults); err != nil {
		r.log.Errorw("settings_repo: failed to create default settings", "error", err)
		return err
	}
	r.log.Info("settings_repo: default settings row created")
	return nil
}
