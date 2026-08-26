// Package paymentsmetrics — бизнес-метрики пополнений баланса. Инкрементятся
// не из payments_backend/handlers, а прямо внутри ReplenishmentSrv.Confirm/Fail
// (internal/service) — идемпотентность (повторный вебхук мерчанта не должен
// зачислить дважды) там завязана на bool changed/credited, а хендлеру эта
// информация не видна: он получает от Confirm/Fail только err. Считать по
// err == nil в хендлере значило бы задваивать метрику на каждом ретрае вебхука.
package paymentsmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ReplenishmentsTotal — merchant: crystalpay/yookassa/tinkoff, status: paid/failed.
	ReplenishmentsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "replenishments_total",
		Help: "Total number of balance top-up attempts confirmed or failed, by merchant and outcome.",
	}, []string{"merchant", "status"})

	// ReplenishmentAmountTotal — сумма только по paid, по мерчантам.
	ReplenishmentAmountTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "replenishment_amount_total",
		Help: "Total amount credited to user balances, by merchant.",
	}, []string{"merchant"})

	// WebhookSignatureInvalidTotal — вебхук отвергнут ещё до похода в
	// ReplenishmentSrv, подпись не сошлась. Только crystalpay/tinkoff — ЮKassa
	// уведомления не подписывает вовсе (см. YooKassaWebhook), считать там нечего.
	WebhookSignatureInvalidTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "payments_webhook_signature_invalid_total",
		Help: "Total number of merchant webhooks rejected for a bad signature, by merchant.",
	}, []string{"merchant"})

	// HTTPRequestsTotal/HTTPRequestDurationSeconds — payments_-префикс, не
	// общее с adminmetrics'им тёзкой имя: internal/service (AdminSrv и
	// ReplenishmentSrv) импортируют оба пакета метрик разом, так что оба
	// оказываются зарегистрированы в одном registry в любом из трёх
	// бинарников — общее имя тут же паникует на MustRegister (см. её
	// doc-комментарий в internal/metrics/admin). route — шаблон маршрута
	// (gin's c.FullPath(), не сырой путь с invoice_id).
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "payments_http_requests_total",
		Help: "Total number of HTTP requests, by route, method and status.",
	}, []string{"route", "method", "status"})
	HTTPRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "payments_http_request_duration_seconds",
		Help: "Time spent handling an HTTP request, by route and method.",
	}, []string{"route", "method"})
)
