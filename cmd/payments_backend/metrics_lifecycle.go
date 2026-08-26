package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/trottling/Telegram-Store/internal/metrics/dbstats"
)

// metricsShutdownTimeout — этот сервер не трогает Redis/вебхуки, поэтому не
// завязан на таймаут основного webServer.Shutdown — короткого достаточно.
const metricsShutdownTimeout = 5 * time.Second

// RunMetricsServer регистрирует старт/стоп /metrics-сервера в fx.Lifecycle,
// тем же паттерном, что и cmd/bot/metrics_lifecycle.go. _ *dbstats.Collector
// форсирует его сборку — см. cmd/bot/metrics_lifecycle.go.
func RunMetricsServer(lc fx.Lifecycle, srv *http.Server, _ *dbstats.Collector, log *zap.SugaredLogger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Errorw("payments_backend: metrics server stopped unexpectedly", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, metricsShutdownTimeout)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.Warnw("payments_backend: failed to shut down metrics server", "error", err)
			}
			return nil
		},
	})
}
