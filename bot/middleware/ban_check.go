package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	domainservice "github.com/trottling/Telegram-Store/internal/domain/service"
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
		lang := texts.Normalize(extractLanguageCode(update))

		// GetFreshProfile, не GetProfile: бан должен сработать на этом же
		// update'е, а не спустя до userTTL, если инвалидация кэша после
		// BanUser почему-то не сработала. Заодно кладём результат в ctx —
		// GetProfile ниже по цепочке (см. domainservice.WithUser) отдаст его
		// без повторного похода в кэш/Postgres.
		user, err := m.userService.GetFreshProfile(ctx, chatID)
		if err != nil {
			if errors.Is(err, domainerrors.ErrUserNotFound) {
				m.reply(ctx, b, chatID, texts.T(lang, texts.PleaseStartMsg, nil))
				return
			}
			m.log.Errorw("ban_check: failed to load user", "error", err, "telegram_id", chatID)
			return
		}

		if user.IsBanned() {
			m.reply(ctx, b, chatID, texts.T(lang, texts.BannedMsg, nil))
			return
		}

		next(domainservice.WithUser(ctx, user), b, update)
	}
}

func (m *Middlewares) reply(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		m.log.Errorf("ban_check: failed to send message to %d: %v", chatID, err)
	}
}
