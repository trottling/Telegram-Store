package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// AnswerCallback подтверждает callback_query до next() — иначе кнопка
// зависает в состоянии «loading» до таймаута Telegram.
func (m *Middlewares) AnswerCallback(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.CallbackQuery != nil {
			if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
			}); err != nil {
				m.log.Errorf("answer_callback: failed to answer callback query %s: %v", update.CallbackQuery.ID, err)
			}
		}
		next(ctx, b, update)
	}
}
