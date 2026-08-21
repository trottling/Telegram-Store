package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// AnalyticsRepository — агрегаты для бизнес-дашборда Grafana. Отдельно от
// остальных *Admin-репозиториев: единственный потребитель — internal/metrics/admin
// (Prometheus custom Collector), которому не нужны пагинация/фильтры.
type AnalyticsRepository interface {
	// GetSnapshot — topN ограничивает топ продуктов/категорий (кардинальность
	// лейбла в Prometheus должна быть маленькой и фиксированной).
	GetSnapshot(ctx context.Context, topN int) (*models.AnalyticsSnapshot, error)
}
