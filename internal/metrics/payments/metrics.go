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
)
