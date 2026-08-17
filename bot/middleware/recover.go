package middleware

import (
	"context"
	"runtime/debug"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sirupsen/logrus"
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
				m.log.WithFields(logrus.Fields{
					"telegram_id": chatID,
					"update_id":   update.ID,
					"panic":       r,
					"stack":       string(debug.Stack()),
				}).Error("middleware: panic recovered")
			}
		}()
		next(ctx, b, update)
	}
}
