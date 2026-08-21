package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
)

// analyticsTopN — сколько строк топа продуктов/категорий отдавать в
// Prometheus. Фиксировано и небольшое намеренно: это лейбл метрики, а не
// параметр листинга — незачем неограниченному числу товаров превращаться в
// неограниченное число time series.
const analyticsTopN = 10

type AnalyticsSrv struct {
	repo repository.AnalyticsRepository
}

func NewAnalyticsSrv(repo repository.AnalyticsRepository) *AnalyticsSrv {
	return &AnalyticsSrv{repo: repo}
}

func (s *AnalyticsSrv) GetSnapshot(ctx context.Context) (*models.AnalyticsSnapshot, error) {
	return s.repo.GetSnapshot(ctx, analyticsTopN)
}
