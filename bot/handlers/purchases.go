package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	domainmodels "github.com/trottling/Telegram-Store/internal/domain/models"
)

const purchasesPageSize = 10

// PurchasesHandler открывает первую страницу истории покупок.
func (h *Handlers) PurchasesHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.renderPurchases(ctx, b, update.Message.Chat.ID, 0, 0)
}

// PurchasesPageHandler — кнопки «вперёд»/«назад» по страницам истории.
func (h *Handlers) PurchasesPageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, messageID, ok := utils.CallbackTarget(update)
	if !ok {
		return
	}

	offset, err := utils.ParseCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("PurchasesPageHandler: failed to parse purchases page callback: %v", err)
		return
	}

	h.renderPurchases(ctx, b, chatID, int(offset), messageID)
}

func (h *Handlers) renderPurchases(ctx context.Context, b *bot.Bot, chatID int64, offset int, messageID int) {
	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("renderPurchases: failed to get profile for %d: %v", chatID, err)
		return
	}

	batches, err := h.purchaseService.GetUserPurchases(ctx, chatID, offset, purchasesPageSize)
	if err != nil {
		h.log.Errorf("renderPurchases: failed to get user %d purchases: %v", chatID, err)
		return
	}

	total, err := h.purchaseService.CountUserPurchaseBatches(ctx, chatID)
	if err != nil {
		h.log.Errorf("renderPurchases: failed to count user %d purchase batches: %v", chatID, err)
		return
	}

	text := texts.T(user.Language, texts.PurchasesMsg, nil)
	if len(batches) == 0 {
		text = texts.T(user.Language, texts.PurchasesEmptyMsg, nil)
	}

	kb := keyboards.BuildPurchasesKb(user.Language, batches, offset, purchasesPageSize, total)

	if messageID == 0 {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: kb,
		}); err != nil {
			h.log.Errorf("renderPurchases: failed to send message to %d: %v", chatID, err)
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
		h.log.Errorf("renderPurchases: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// PurchaseDetailHandler открывает весь батч покупки (все единицы одного
// Buy()), редактируя список покупок на месте — «назад» возвращает список той
// же страницы (переиспользует PurchasesPageHandler через тот же callback).
func (h *Handlers) PurchaseDetailHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, messageID, ok := utils.CallbackTarget(update)
	if !ok {
		return
	}

	offset, batchID, err := utils.ParseBatchCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("PurchaseDetailHandler: failed to parse purchase batch callback: %v", err)
		return
	}

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("PurchaseDetailHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	purchases, err := h.purchaseService.GetBatch(ctx, chatID, batchID)
	if err != nil {
		h.log.Errorf("PurchaseDetailHandler: failed to get purchase batch %s: %v", batchID, err)
		return
	}
	if len(purchases) == 0 {
		return
	}

	contents := make([]string, len(purchases))
	total := domainmodels.Money{}
	for i, p := range purchases {
		content := ""
		if p.Item != nil {
			content = p.Item.Content
		}
		contents[i] = fmt.Sprintf("`%s`", utils.EscapeMarkdownCode(content))
		total = total.Add(p.Amount)
	}

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: texts.T(user.Language, texts.BackBtn, nil), CallbackData: utils.BuildPurchasesPageCallback(offset)}},
		},
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text: texts.T(user.Language, texts.PurchaseDetailMsg, map[string]any{
			"Name":        utils.EscapeMarkdown(purchases[0].Product.Name),
			"Amount":      utils.FormatAmount(total),
			"Count":       len(purchases),
			"Date":        utils.FormatDate(purchases[0].CreatedAt),
			"Description": utils.EscapeMarkdown(purchases[0].Product.Description),
			"Content":     strings.Join(contents, "\n"),
		}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: kb,
	}); err != nil {
		h.log.Errorf("PurchaseDetailHandler: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}
