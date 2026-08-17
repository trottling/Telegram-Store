package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	domain "github.com/trottling/Telegram-Store/internal/domain/models"
)

// ProfileHandler и ProfileCallbackHandler ведут на один и тот же экран профиля.
func (h *Handlers) ProfileHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.sendProfile(ctx, b, update.Message.Chat.ID, false)
}

func (h *Handlers) ProfileCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.sendProfile(ctx, b, update.CallbackQuery.Message.Message.Chat.ID, false)
}

// ProfileRefreshHandler — кнопка «Обновить» на клавиатуре профиля: та же
// карточка, но профиль перечитывается напрямую из Postgres, минуя кэш
// (статистика покупок и так всегда считается мимо кэша, см. PurchaseSrv.GetUserStats).
func (h *Handlers) ProfileRefreshHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.sendProfile(ctx, b, update.Message.Chat.ID, true)
}

func (h *Handlers) sendProfile(ctx context.Context, b *bot.Bot, chatID int64, bypassCache bool) {
	count, spent, err := h.purchaseService.GetUserStats(ctx, chatID)
	if err != nil {
		h.log.Errorf("sendProfile: failed to get user %d stats: %v", chatID, err)
		return
	}

	var user *domain.User
	if bypassCache {
		user, err = h.userService.RefreshProfile(ctx, chatID)
	} else {
		user, err = h.userService.GetProfile(ctx, chatID)
	}
	if err != nil {
		h.log.Errorf("sendProfile: failed to get user %d profile: %v", chatID, err)
		return
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: texts.T(user.Language, texts.ProfileMsg, map[string]any{
			"Username":   utils.EscapeMarkdown(user.Username),
			"TelegramID": user.TelegramID,
			"Balance":    utils.FormatAmount(user.Balance),
			"Count":      count,
			"Spent":      utils.FormatAmount(spent),
		}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: h.kb.ProfileKb(user.Language),
	}); err != nil {
		h.log.Errorf("sendProfile: failed to send message to %d: %v", chatID, err)
	}
}
