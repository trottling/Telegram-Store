package handlers

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
)

// ProfileHandler и ProfileCallbackHandler ведут на один и тот же экран профиля.
func (h *Handlers) ProfileHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.sendProfile(ctx, b, update.Message.Chat.ID)
}

func (h *Handlers) ProfileCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.sendProfile(ctx, b, update.CallbackQuery.Message.Message.Chat.ID)
}

func (h *Handlers) sendProfile(ctx context.Context, b *bot.Bot, chatID int64) {
	count, spent, err := h.purchaseService.GetUserStats(ctx, chatID)
	if err != nil {
		h.log.Errorf("sendProfile: failed to get user %d stats: %v", chatID, err)
		return
	}

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("sendProfile: failed to get user %d profile: %v", chatID, err)
		return
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf(texts.ProfileMsg, user.Username, user.TelegramID, user.Balance, count, spent),
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: h.kb.ProfileKb,
	}); err != nil {
		h.log.Errorf("sendProfile: failed to send message to %d: %v", chatID, err)
	}
}
