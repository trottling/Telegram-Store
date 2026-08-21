package adminmetrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewServer — отдельный HTTP-сервер только под /metrics, тем же паттерном,
// что и у бота (internal/metrics/bot). Порт — ADMIN_METRICS_PORT, только на
// backend-network, наружу не торчит.
func NewServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}
