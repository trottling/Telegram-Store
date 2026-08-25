package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	domainmodels "github.com/trottling/Telegram-Store/internal/domain/models"
)

const replenishmentsPageSize = 10

// ReplenishmentsHandler открывает первую страницу истории пополнений.
func (h *Handlers) ReplenishmentsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.renderReplenishments(ctx, b, domainmodels.TelegramID(update.Message.Chat.ID), 0, 0)
}

// ReplenishmentsPageHandler — кнопки «вперёд»/«назад» по страницам истории.
func (h *Handlers) ReplenishmentsPageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, messageID, ok := utils.CallbackTarget(update)
	if !ok {
		return
	}

	offset, err := utils.ParseCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("ReplenishmentsPageHandler: failed to parse replenishments page callback: %v", err)
		return
	}

	h.renderReplenishments(ctx, b, chatID, int(offset), messageID)
}

// renderReplenishments: по кнопке на пополнение, как у покупок — раскрывается
// по тапу в ReplenishmentDetailHandler.
func (h *Handlers) renderReplenishments(ctx context.Context, b *bot.Bot, chatID domainmodels.TelegramID, offset int, messageID int) {
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

	text := texts.T(user.Language, texts.ReplenishmentsMsg, nil)
	if len(items) == 0 {
		text = texts.T(user.Language, texts.ReplenishmentsEmptyMsg, nil)
	}

	kb := keyboards.BuildReplenishmentsKb(user.Language, items, offset, replenishmentsPageSize, total)

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

// ReplenishmentDetailHandler открывает карточку одного пополнения — тот же
// паттерн, что у PurchaseDetailHandler: редактирует список на месте, «назад»
// возвращает ту же страницу через ReplenishmentsPageHandler.
func (h *Handlers) ReplenishmentDetailHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, messageID, ok := utils.CallbackTarget(update)
	if !ok {
		return
	}

	offset, id, err := utils.ParseReplenishmentDetailCallback(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("ReplenishmentDetailHandler: failed to parse callback: %v", err)
		return
	}

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("ReplenishmentDetailHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	r, err := h.replenishmentService.GetUserReplenishment(ctx, chatID, id)
	if err != nil {
		h.log.Errorf("ReplenishmentDetailHandler: failed to get replenishment %d for %d: %v", id, chatID, err)
		return
	}

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: texts.T(user.Language, texts.BackBtn, nil), CallbackData: utils.BuildReplenishmentsPageCallback(offset)}},
		},
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text: texts.T(user.Language, texts.ReplenishmentDetailMsg, map[string]any{
			"Amount":   utils.FormatAmount(r.Amount),
			"Status":   texts.ReplenishmentStatusName(user.Language, r.Status),
			"Date":     utils.FormatDate(r.CreatedAt),
			"Merchant": texts.MerchantName(user.Language, r.Merchant),
		}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: kb,
	}); err != nil {
		h.log.Errorf("ReplenishmentDetailHandler: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}
