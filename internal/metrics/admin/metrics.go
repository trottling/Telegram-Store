// Package adminmetrics — бизнес-метрики admin_backend: счётчик админских
// действий (ActionsTotal) плюс агрегатный Collector поверх Postgres
// (см. collector.go). Package-level переменная, как и в internal/metrics/bot —
// AdminSrv не получает новой DI-зависимости ради одного счётчика.
package adminmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ActionsTotal — action: тот же набор строк, что уже пишется в AdminLog.Action
// (ban/unban/balance_add/make_admin/revoke_admin/product_*/category_*/
// settings_update/referral_*) — инкрементится в единственной точке, где эти
// строки уже собираются, AdminSrv.logAction.
var ActionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "admin_actions_total",
	Help: "Total number of admin actions performed, by action.",
}, []string{"action"})

// HTTPRequestsTotal/HTTPRequestDurationSeconds — в отличие от dbstats.Collector
// (один Collector.New() вызов на процесс, ничего не коллизирует), это
// package-level promauto-переменные: internal/service — один пакет на
// AdminSrv и ReplenishmentSrv разом, поэтому его импортируют все три
// бинарника, и adminmetrics/paymentsmetrics оказываются зарегистрированы в
// одном and том же процессе/registry всегда, не только "в своём" бинарнике.
// Одинаковое имя тут же паникует на MustRegister — отсюда admin_-префикс,
// а не общее с paymentsmetrics имя. route — шаблон маршрута (gin's
// c.FullPath(), ограниченная кардинальность — не сырой путь с
// telegram_id/product_id), не сработавший маршрут — "unmatched".
var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "admin_http_requests_total",
		Help: "Total number of HTTP requests, by route, method and status.",
	}, []string{"route", "method", "status"})
	HTTPRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "admin_http_request_duration_seconds",
		Help: "Time spent handling an HTTP request, by route and method.",
	}, []string{"route", "method"})
)
