package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// AnalyticsService — единственный потребитель: internal/metrics/admin.Collector,
// на каждый Prometheus-скрейп бизнес-дашборда admin_backend.
type AnalyticsService interface {
	GetSnapshot(ctx context.Context) (*models.AnalyticsSnapshot, error)
}
