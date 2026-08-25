package adminmetrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"go.uber.org/zap"
)

// AnalyticsCollector — custom Prometheus Collector: агрегаты (SUM/COUNT по
// Postgres, через AnalyticsService) считаются заново на каждый скрейп, а не
// на каждое событие в коде. Идиома для "длиннохвостых" данных вроде топа
// продуктов: кардинальность лейбла ограничена LIMIT в самом SQL-запросе
// (AnalyticsSrv.GetSnapshot), а не размером каталога — ни один лейбл тут не
// завязан на user_id/product_id/telegram_id напрямую.
type AnalyticsCollector struct {
	svc service.AnalyticsService
	log *zap.SugaredLogger

	totalRevenue   *prometheus.Desc
	totalPurchases *prometheus.Desc
	totalUsers     *prometheus.Desc
	bannedUsers    *prometheus.Desc
	adminUsers     *prometheus.Desc
	totalBalance   *prometheus.Desc
	availableStock *prometheus.Desc
	topProduct     *prometheus.Desc
	topCategory    *prometheus.Desc
}

// NewAnalyticsCollector сразу регистрирует себя в дефолтном registry (том же,
// что promhttp.Handler() в server.go отдаёт по /metrics) — конструктор
// вызывается ровно один раз через fx, повторной регистрации не будет.
func NewAnalyticsCollector(svc service.AnalyticsService, log *zap.SugaredLogger) *AnalyticsCollector {
	c := &AnalyticsCollector{
		svc: svc,
		log: log,

		totalRevenue:   prometheus.NewDesc("shop_total_revenue", "Total revenue from completed purchases.", nil, nil),
		totalPurchases: prometheus.NewDesc("shop_total_purchases", "Total number of completed purchases.", nil, nil),
		totalUsers:     prometheus.NewDesc("shop_total_users", "Total number of registered users.", nil, nil),
		bannedUsers:    prometheus.NewDesc("shop_banned_users", "Total number of banned users.", nil, nil),
		adminUsers:     prometheus.NewDesc("shop_admin_users", "Total number of admin/root admin users.", nil, nil),
		totalBalance:   prometheus.NewDesc("shop_total_balance", "Sum of all user balances.", nil, nil),
		availableStock: prometheus.NewDesc("shop_available_stock_total", "Total number of unsold product items.", nil, nil),
		topProduct:     prometheus.NewDesc("shop_top_products_revenue", "Revenue of top products by revenue.", []string{"product_name"}, nil),
		topCategory:    prometheus.NewDesc("shop_top_categories_revenue", "Revenue of top categories by revenue.", []string{"category_name"}, nil),
	}
	prometheus.MustRegister(c)
	return c
}

func (c *AnalyticsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalRevenue
	ch <- c.totalPurchases
	ch <- c.totalUsers
	ch <- c.bannedUsers
	ch <- c.adminUsers
	ch <- c.totalBalance
	ch <- c.availableStock
	ch <- c.topProduct
	ch <- c.topCategory
}

// Collect выполняется на каждый Prometheus-скрейп — гоняет агрегаты в
// Postgres заново, ничего не кэширует между вызовами.
func (c *AnalyticsCollector) Collect(ch chan<- prometheus.Metric) {
	snap, err := c.svc.GetSnapshot(context.Background())
	if err != nil {
		c.log.Errorw("admin_metrics: failed to collect analytics snapshot", "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalRevenue, prometheus.GaugeValue, snap.TotalRevenue.Float64())
	ch <- prometheus.MustNewConstMetric(c.totalPurchases, prometheus.GaugeValue, float64(snap.TotalPurchases))
	ch <- prometheus.MustNewConstMetric(c.totalUsers, prometheus.GaugeValue, float64(snap.TotalUsers))
	ch <- prometheus.MustNewConstMetric(c.bannedUsers, prometheus.GaugeValue, float64(snap.BannedUsers))
	ch <- prometheus.MustNewConstMetric(c.adminUsers, prometheus.GaugeValue, float64(snap.AdminUsers))
	ch <- prometheus.MustNewConstMetric(c.totalBalance, prometheus.GaugeValue, snap.TotalBalance.Float64())
	ch <- prometheus.MustNewConstMetric(c.availableStock, prometheus.GaugeValue, float64(snap.AvailableStock))
	for _, p := range snap.TopProducts {
		ch <- prometheus.MustNewConstMetric(c.topProduct, prometheus.GaugeValue, p.Revenue.Float64(), p.Name)
	}
	for _, cat := range snap.TopCategories {
		ch <- prometheus.MustNewConstMetric(c.topCategory, prometheus.GaugeValue, cat.Revenue.Float64(), cat.Name)
	}
}
