package service

import (
	"context"

	"github.com/sirupsen/logrus"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
)

type SettingsSrv struct {
	settingsRepo repository.SettingsRepository
	cache        domaincache.SettingsCache
	log          *logrus.Logger
}

func NewSettingsSrv(settingsRepo repository.SettingsRepository, cache domaincache.SettingsCache, log *logrus.Logger) *SettingsSrv {
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
