package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"

	"github.com/trottling/Telegram-Store/backend"
)

// runServer регистрирует старт/стоп admin API в fx.Lifecycle. webServer.Start
// блокируется до Shutdown — запускается в горутине, ошибку логируем сами
// (fx.Lifecycle.OnStart не место для долгоживущего листенера).
func runServer(lc fx.Lifecycle, webServer *backend.Server, redisClient *redis.Client, log *logrus.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := webServer.Start(); err != nil {
					log.Errorf("backend: admin API server stopped unexpectedly: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := webServer.Shutdown(shutdownCtx); err != nil {
				log.Warnf("backend: failed to shut down admin API server: %v", err)
			}
			if err := redisClient.Close(); err != nil {
				log.Warnf("backend: failed to close redis client: %v", err)
			}
			return nil
		},
	})
}
