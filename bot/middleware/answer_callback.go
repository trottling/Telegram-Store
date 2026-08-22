package middleware

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/utils"
)

// AnswerCallback подтверждает callback_query до next() — иначе кнопка
// зависает в состоянии «loading» до таймаута Telegram. Callback можно
// подтвердить только один раз, поэтому CheckPaymentCallbackPrefix — исключение:
// CheckPaymentHandler сам отвечает всплывающим уведомлением о статусе оплаты
// (см. bot/handlers/refill.go), обычный пустой ack тут был бы лишним.
func (m *Middlewares) AnswerCallback(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.CallbackQuery != nil && !strings.HasPrefix(update.CallbackQuery.Data, utils.CheckPaymentCallbackPrefix) {
			if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
			}); err != nil {
				m.log.Errorf("answer_callback: failed to answer callback query %s: %v", update.CallbackQuery.ID, err)
			}
		}
		next(ctx, b, update)
	}
}
