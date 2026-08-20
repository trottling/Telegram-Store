// Package botmetrics — Prometheus-метрики бота. promauto регистрирует их на
// дефолтном registry, поэтому promhttp.Handler() (см. server.go) и бесплатные
// go_*/process_*-коллекторы работают без отдельной проводки. Package-level
// переменные, а не поля на каком-то сервисе — метрики вызываются из
// bot/middleware и bot/handlers, которым не нужен новый DI ради одного счётчика.
package botmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// UpdatesTotal — kind: message/callback_query/other, та же классификация,
	// что в bot/middleware.classifyKind.
	UpdatesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bot_updates_total",
		Help: "Total number of Telegram updates received, by kind.",
	}, []string{"kind"})

	// UpdateDurationSeconds — полное время обработки update'а по всей
	// миддлварь-цепочке и хендлеру. Дефолтные бакеты подходят — хендлеры бота
	// подсекундные.
	UpdateDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "bot_update_duration_seconds",
		Help: "Time spent handling a single Telegram update, by kind.",
	}, []string{"kind"})

	// PanicsTotal — инкрементится внутри middleware.Recover, где уже есть
	// recovered-значение и kind в скоупе. Снаружи панику не увидеть:
	// bot.HandlerFunc ничего не возвращает.
	PanicsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bot_panics_total",
		Help: "Total number of panics recovered while handling updates, by kind.",
	}, []string{"kind"})

	// PurchasesTotal — status: success/failed.
	PurchasesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bot_purchases_total",
		Help: "Total number of purchase attempts, by outcome.",
	}, []string{"status"})

	// PurchaseUnitsTotal — сумма купленных единиц на успешных покупках.
	PurchaseUnitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bot_purchase_units_total",
		Help: "Total number of product units sold across successful purchases.",
	})

	// PurchaseAmountTotal — сумма Purchase.Amount на успешных покупках.
	PurchaseAmountTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bot_purchase_amount_total",
		Help: "Total amount charged across successful purchases.",
	})
)
