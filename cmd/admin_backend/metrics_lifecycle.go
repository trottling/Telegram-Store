package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	adminmetrics "github.com/trottling/Telegram-Store/internal/metrics/admin"
)

// metricsShutdownTimeout — этот сервер не трогает Redis/сессии, поэтому не
// завязан на таймаут основного webServer.Shutdown — короткого достаточно.
const metricsShutdownTimeout = 5 * time.Second

// RunMetricsServer регистрирует старт/стоп /metrics-сервера в fx.Lifecycle,
// тем же паттерном, что и cmd/bot/metrics_lifecycle.go. Параметр collector не
// используется в теле напрямую — он тут ради самого факта присутствия в
// сигнатуре: fx строит его только если что-то требует его как зависимость, а
// NewAnalyticsCollector регистрирует себя в Prometheus прямо в конструкторе
// (см. internal/metrics/admin/collector.go), поэтому важно лишь, чтобы fx его
// вообще построил.
func RunMetricsServer(lc fx.Lifecycle, srv *http.Server, _ *adminmetrics.AnalyticsCollector, log *zap.SugaredLogger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Errorw("admin_backend: metrics server stopped unexpectedly", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, metricsShutdownTimeout)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.Warnw("admin_backend: failed to shut down metrics server", "error", err)
			}
			return nil
		},
	})
}
