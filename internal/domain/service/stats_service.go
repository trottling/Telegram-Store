package service

import (
	"context"
	"time"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// StatsService — данные для экрана статистики.
type StatsService interface {
	// GetDashboard: from/to nil — за всё время; график выручки всегда получает конкретный период.
	GetDashboard(ctx context.Context, from, to *time.Time) (*models.DashboardStats, error)
}
