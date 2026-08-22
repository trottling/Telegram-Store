package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/trottling/Telegram-Store/bot"
	"github.com/trottling/Telegram-Store/internal/config"
)

// webhookShutdownTimeout — свой HTTP-сервер, Redis/покупки не трогает, поэтому
// не завязан на drainTimeout из runBot.
const webhookShutdownTimeout = 5 * time.Second

// RunWebhookServer поднимает HTTP-сервер приёма вебхуков Telegram и
// регистрирует его URL в Telegram (SetWebhook) — только если BOT_WEBHOOK_URL
// задан (см. TelegramBot.Start); иначе бот работает через long polling, и
// этот Invoke — no-op.
//
// Регистрируется в fx.Invoke ПОСЛЕ runBot: при остановке хуки идут в обратном
// порядке, так что сервер вебхуков перестаёт принимать новые апдейты раньше,
// чем runBot отменит ctx и начнёт ждать уже начатые — Telegram сразу видит
// закрытое соединение и ретраит позже, вместо гонки с догребающими
// обработчиками.
func RunWebhookServer(lc fx.Lifecycle, bt *bot.TelegramBot, webhookCfg *config.BotWebhookConfig, log *zap.SugaredLogger) {
	if webhookCfg.URL == "" {
		return
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webhookCfg.Port),
		Handler: bt.WebhookHandler(),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Errorw("bot: webhook server stopped unexpectedly", "error", err)
				}
			}()

			if err := bt.RegisterWebhook(ctx); err != nil {
				return fmt.Errorf("register telegram webhook: %w", err)
			}
			log.Infow("bot: webhook registered", "url", webhookCfg.URL)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, webhookShutdownTimeout)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.Warnw("bot: failed to shut down webhook server", "error", err)
			}
			return nil
		},
	})
}
