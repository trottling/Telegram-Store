package botmetrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/trottling/Telegram-Store/internal/config"
)

// NewServer — отдельный HTTP-сервер только под /metrics, не тот же процесс,
// что admin_backend/payments_backend (у бота своего HTTP-сервера раньше не
// было вовсе). Порт — BOT_METRICS_PORT, только на backend-network, наружу не
// торчит (см. docker-compose.yml, сервис prometheus).
func NewServer(cfg *config.MetricsConfig) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}
}
