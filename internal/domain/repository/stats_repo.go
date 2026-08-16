package repository

import (
	"context"
	"time"

	"github.com/trottling/TG-Store/internal/domain/models"
)

// StatsRepository — агрегаты для экрана статистики, только чтение, без кэша.
type StatsRepository interface {
	GetSalesOverview(ctx context.Context, from, to *time.Time) (*models.SalesOverview, error)
	// GetRevenueTimeSeries — выручка по дням, оба края периода обязательны.
	GetRevenueTimeSeries(ctx context.Context, from, to time.Time) ([]models.RevenuePoint, error)
	GetTopProducts(ctx context.Context, from, to *time.Time, limit int, orderBy models.StatsOrderBy) ([]models.ProductStat, error)
	GetTopCategories(ctx context.Context, from, to *time.Time, limit int, orderBy models.StatsOrderBy) ([]models.CategoryStat, error)
	GetUserStats(ctx context.Context) (*models.UserStats, error)
}
