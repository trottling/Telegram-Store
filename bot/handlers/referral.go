package handlers

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	domain "github.com/trottling/Telegram-Store/internal/domain/models"
)

// ReferralHandler показывает реферальную программу: процент, ссылку и личную
// статистику. Если Referral.Enabled=false в настройках — программа выключена
// целиком, независимо от процента.
func (h *Handlers) ReferralHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	settings, err := h.settingsService.Get(ctx)
	if err != nil {
		h.log.Errorf("ReferralHandler: failed to get settings: %v", err)
		return
	}
	if !settings.Referral.Enabled {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: texts.ReferralUnavailableMsg}); err != nil {
			h.log.Errorf("ReferralHandler: failed to send message to %d: %v", chatID, err)
		}
		return
	}

	invited, err := h.userService.CountReferrals(ctx, chatID)
	if err != nil {
		h.log.Errorf("ReferralHandler: failed to count referrals for %d: %v", chatID, err)
		return
	}

	totalCredited, err := h.replenishmentService.SumUserMerchantAmount(ctx, chatID, domain.MerchantReferral)
	if err != nil {
		h.log.Errorf("ReferralHandler: failed to sum referral credits for %d: %v", chatID, err)
		return
	}

	link := fmt.Sprintf("https://t.me/%s?start=%d", h.botUsername, chatID)

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: texts.CloseBtn, CallbackData: utils.ReferralCloseCallback},
				{Text: texts.ShareBtn, URL: "https://t.me/share/url?url=" + url.QueryEscape(link)},
			},
		},
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf(texts.ReferralMsg, settings.Referral.Percent, link, invited, totalCredited),
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: kb,
	}); err != nil {
		h.log.Errorf("ReferralHandler: failed to send message to %d: %v", chatID, err)
	}
}

// ReferralCloseHandler — инлайн-кнопка «Закрыть».
func (h *Handlers) ReferralCloseHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	if _, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: chatID, MessageID: messageID}); err != nil {
		h.log.Errorf("ReferralCloseHandler: failed to delete message %d in chat %d: %v", messageID, chatID, err)
	}
}
