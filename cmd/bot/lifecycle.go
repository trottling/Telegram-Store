package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/trottling/Telegram-Store/bot"
)

// drainTimeout — сколько ждём догребающие update'ы при остановке.
const drainTimeout = 15 * time.Second

// runBot регистрирует старт/стоп бота в fx.Lifecycle. bt.Start блокируется
// до отмены ctx (long-polling), поэтому запускается в горутине — OnStart
// должен вернуться быстро, иначе fx посчитает старт приложения не завершённым.
func runBot(lc fx.Lifecycle, bt *bot.TelegramBot, redisClient *redis.Client, log *zap.SugaredLogger) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go bt.Start(ctx)
			return nil
		},
		// Порядок обязателен: сначала останавливаем поллинг, потом даём
		// догрестись начатым update'ам и только затем закрываем Redis. Иначе
		// покупка, попавшая под SIGTERM, коммитится, а сброс кэша баланса
		// падает на закрытом клиенте — пользователь до истечения TTL видит
		// старый баланс.
		OnStop: func(ctx context.Context) error {
			cancel()

			drainCtx, drainCancel := context.WithTimeout(ctx, drainTimeout)
			defer drainCancel()
			if err := bt.WaitInFlight(drainCtx); err != nil {
				log.Warnf("bot: in-flight updates did not finish in time: %v", err)
			}

			if err := redisClient.Close(); err != nil {
				log.Warnf("bot: failed to close redis client: %v", err)
			}
			return nil
		},
	})
}
