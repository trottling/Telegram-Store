package service

import (
	"context"

	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"go.uber.org/zap"
)

type SettingsSrv struct {
	settingsRepo repository.SettingsRepository
	cache        domaincache.SettingsCache
	log          *zap.SugaredLogger
}

func NewSettingsSrv(settingsRepo repository.SettingsRepository, cache domaincache.SettingsCache, log *zap.SugaredLogger) *SettingsSrv {
	return &SettingsSrv{settingsRepo: settingsRepo, cache: cache, log: log}
}

func (s *SettingsSrv) Get(ctx context.Context) (*models.Settings, error) {
	if settings, err := s.cache.GetSettings(ctx); err == nil {
		return settings, nil
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetSettings(ctx, settings)
	return settings, nil
}
