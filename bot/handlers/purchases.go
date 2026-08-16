package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/TG-Store/bot/keyboards"
	"github.com/trottling/TG-Store/bot/texts"
	"github.com/trottling/TG-Store/bot/utils"
)

const purchasesPageSize = 10

// PurchasesHandler открывает первую страницу истории покупок.
func (h *Handlers) PurchasesHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.renderPurchases(ctx, b, update.Message.Chat.ID, 0, 0)
}

// PurchasesPageHandler — кнопки «вперёд»/«назад» по страницам истории.
func (h *Handlers) PurchasesPageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	offset, err := utils.ParseCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("PurchasesPageHandler: failed to parse purchases page callback: %v", err)
		return
	}

	h.renderPurchases(ctx, b, chatID, int(offset), messageID)
}

func (h *Handlers) renderPurchases(ctx context.Context, b *bot.Bot, chatID int64, offset int, messageID int) {
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

	text := texts.PurchasesMsg
	if len(batches) == 0 {
		text = texts.PurchasesEmptyMsg
	}

	kb := keyboards.BuildPurchasesKb(batches, offset, purchasesPageSize, total)

	if messageID == 0 {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdownV1,
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
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: kb,
	}); err != nil {
		h.log.Errorf("renderPurchases: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// PurchaseDetailHandler открывает весь батч покупки (все единицы одного Buy()), не одну штуку.
func (h *Handlers) PurchaseDetailHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID

	batchID, err := utils.ParseBatchCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("PurchaseDetailHandler: failed to parse purchase batch callback: %v", err)
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
	var total float64
	for i, p := range purchases {
		content := ""
		if p.Item != nil {
			content = p.Item.Content
		}
		contents[i] = fmt.Sprintf("`%s`", content)
		total += p.Amount
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf(texts.PurchaseDetailMsg, purchases[0].Product.Name, total, len(purchases), purchases[0].CreatedAt.Format("02.01.2006 15:04"), strings.Join(contents, "\n")),
		ParseMode: models.ParseModeMarkdownV1,
	}); err != nil {
		h.log.Errorf("PurchaseDetailHandler: failed to send message to %d: %v", chatID, err)
	}
}
