package handlers

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
)

const replenishmentsPageSize = 10

// ReplenishmentsHandler открывает первую страницу истории пополнений.
func (h *Handlers) ReplenishmentsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.renderReplenishments(ctx, b, update.Message.Chat.ID, 0, 0)
}

// ReplenishmentsPageHandler — кнопки «вперёд»/«назад» по страницам истории.
func (h *Handlers) ReplenishmentsPageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	offset, err := utils.ParseCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("ReplenishmentsPageHandler: failed to parse replenishments page callback: %v", err)
		return
	}

	h.renderReplenishments(ctx, b, chatID, int(offset), messageID)
}

// renderReplenishments: в отличие от покупок, тут нечего раскрывать по тапу
// (нет выдаваемого товара) — список рисуется одним текстовым блоком, без
// инлайн-кнопки на строку, только пагинация.
func (h *Handlers) renderReplenishments(ctx context.Context, b *bot.Bot, chatID int64, offset int, messageID int) {
	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("renderReplenishments: failed to get profile for %d: %v", chatID, err)
		return
	}

	items, err := h.replenishmentService.ListUserReplenishments(ctx, chatID, offset, replenishmentsPageSize)
	if err != nil {
		h.log.Errorf("renderReplenishments: failed to get user %d replenishments: %v", chatID, err)
		return
	}

	total, err := h.replenishmentService.CountUserReplenishments(ctx, chatID)
	if err != nil {
		h.log.Errorf("renderReplenishments: failed to count user %d replenishments: %v", chatID, err)
		return
	}

	text := texts.T(user.Language, texts.ReplenishmentsEmptyMsg, nil)
	if len(items) > 0 {
		lines := make([]string, len(items))
		for i, r := range items {
			lines[i] = texts.T(user.Language, texts.ReplenishmentLineMsg, map[string]any{
				"Amount":   utils.FormatAmount(r.Amount),
				"Merchant": texts.MerchantName(user.Language, r.Merchant),
				"Status":   texts.ReplenishmentStatusName(user.Language, r.Status),
				"Date":     utils.FormatDate(r.CreatedAt),
			})
		}
		text = texts.T(user.Language, texts.ReplenishmentsMsg, nil) + "\n\n" + strings.Join(lines, "\n")
	}

	kb := keyboards.BuildReplenishmentsKb(user.Language, offset, replenishmentsPageSize, total)

	if messageID == 0 {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: kb,
		}); err != nil {
			h.log.Errorf("renderReplenishments: failed to send message to %d: %v", chatID, err)
		}
		return
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: kb,
	}); err != nil {
		h.log.Errorf("renderReplenishments: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}
