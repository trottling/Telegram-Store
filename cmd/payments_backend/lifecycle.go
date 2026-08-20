package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"

	paymentsbackend "github.com/trottling/Telegram-Store/payments_backend"
)

// runServer регистрирует старт/стоп webhooks-сервера в fx.Lifecycle.
// webServer.Start блокируется до Shutdown — запускается в горутине, ошибку
// логируем сами (fx.Lifecycle.OnStart не место для долгоживущего листенера).
func runServer(lc fx.Lifecycle, webServer *paymentsbackend.Server, redisClient *redis.Client, log *zap.SugaredLogger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := webServer.Start(); err != nil {
					log.Errorf("payments_backend: webhooks server stopped unexpectedly: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := webServer.Shutdown(shutdownCtx); err != nil {
				log.Warnf("payments_backend: failed to shut down webhooks server: %v", err)
			}
			if err := redisClient.Close(); err != nil {
				log.Warnf("payments_backend: failed to close redis client: %v", err)
			}
			return nil
		},
	})
}
