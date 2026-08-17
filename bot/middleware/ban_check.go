package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
)

// BanCheck — единственный шлюз для всех update, кроме /start: нет
// пользователя → просим /start; забанен → сообщаем и стоп; ошибка — тоже
// стоп (fail closed).
func (m *Middlewares) BanCheck(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		// HasPrefix, не точное совпадение — "/start" пропускает и
		// "/start <id>" (payload реф-ссылки), иначе новый приглашённый
		// пользователь никогда не доходит до StartHandler.
		if update.Message != nil && strings.HasPrefix(update.Message.Text, "/start") {
			next(ctx, b, update)
			return
		}

		chatID, ok := extractChatID(update)
		if !ok {
			next(ctx, b, update)
			return
		}

		banned, err := m.userService.IsBanned(ctx, chatID)
		if err != nil {
			if errors.Is(err, domainerrors.ErrUserNotFound) {
				m.reply(ctx, b, chatID, texts.PleaseStartMsg)
				return
			}
			m.log.WithError(err).WithField("telegram_id", chatID).Error("ban_check: failed to check ban status")
			return
		}

		if banned {
			m.reply(ctx, b, chatID, texts.BannedMsg)
			return
		}

		next(ctx, b, update)
	}
}

func (m *Middlewares) reply(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		m.log.Errorf("ban_check: failed to send message to %d: %v", chatID, err)
	}
}
