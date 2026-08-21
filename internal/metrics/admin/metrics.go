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
