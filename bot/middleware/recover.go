package middleware

import (
	"context"
	"runtime/debug"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	botmetrics "github.com/trottling/Telegram-Store/internal/metrics/bot"
)

// Recover ловит панику в обработке update. go-telegram/bot запускает каждый
// update отдельной горутиной (ProcessUpdate: go r(...)) и recover() не ставит
// нигде — непойманная паника роняет весь процесс, а не одно сообщение.
// Первый в цепочке: applyMiddlewares оборачивает в обратном порядке, поэтому
// m[0] оказывается снаружи и накрывает в том числе Logging.
func (m *Middlewares) Recover(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		defer func() {
			if r := recover(); r != nil {
				chatID, _ := extractChatID(update)
				botmetrics.PanicsTotal.WithLabelValues(classifyKind(update)).Inc()
				m.log.Errorw("middleware: panic recovered",
					"telegram_id", chatID,
					"update_id", update.ID,
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		next(ctx, b, update)
	}
}
