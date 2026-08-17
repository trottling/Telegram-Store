package main

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"

	"github.com/trottling/Telegram-Store/bot"
)

// runBot регистрирует старт/стоп бота в fx.Lifecycle. bt.Start блокируется
// до отмены ctx (long-polling), поэтому запускается в горутине — OnStart
// должен вернуться быстро, иначе fx посчитает старт приложения не завершённым.
func runBot(lc fx.Lifecycle, bt *bot.TelegramBot, redisClient *redis.Client, log *logrus.Logger) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go bt.Start(ctx)
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			if err := redisClient.Close(); err != nil {
				log.Warnf("bot: failed to close redis client: %v", err)
			}
			return nil
		},
	})
}
