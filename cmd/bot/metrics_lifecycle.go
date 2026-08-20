package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// metricsShutdownTimeout — этот сервер не трогает Redis/покупки (в отличие от
// runBot), поэтому не завязан на 15s drainTimeout — короткого таймаута
// достаточно.
const metricsShutdownTimeout = 5 * time.Second

// RunMetricsServer регистрирует старт/стоп /metrics-сервера в fx.Lifecycle,
// тем же паттерном, что и runBot.
func RunMetricsServer(lc fx.Lifecycle, srv *http.Server, log *zap.SugaredLogger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Errorw("bot: metrics server stopped unexpectedly", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, metricsShutdownTimeout)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.Warnw("bot: failed to shut down metrics server", "error", err)
			}
			return nil
		},
	})
}
