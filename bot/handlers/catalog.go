package handlers

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/TG-Store/bot/keyboards"
	"github.com/trottling/TG-Store/bot/texts"
	"github.com/trottling/TG-Store/bot/utils"
)

// CatalogHandler открывает корень каталога.
func (h *Handlers) CatalogHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.renderCatalog(ctx, b, update.Message.Chat.ID, nil, 0)
}

// CatalogRootHandler — кнопка «в корень каталога» из глубины дерева.
func (h *Handlers) CatalogRootHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	h.renderCatalog(ctx, b, chatID, nil, update.CallbackQuery.Message.Message.ID)
}

// CategoryHandler показывает подкатегории и товары одной категории.
func (h *Handlers) CategoryHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID

	categoryID, err := utils.ParseCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("CategoryHandler: failed to parse category callback: %v", err)
		return
	}

	h.renderCatalog(ctx, b, chatID, &categoryID, update.CallbackQuery.Message.Message.ID)
}

// renderCatalog: messageID == 0 — новое сообщение, иначе редактирует текущее.
func (h *Handlers) renderCatalog(ctx context.Context, b *bot.Bot, chatID int64, categoryID *int64, messageID int) {
	children, err := h.categoryService.ListChildren(ctx, categoryID)
	if err != nil {
		h.log.Errorf("renderCatalog: failed to list category children: %v", err)
		return
	}

	products, err := h.categoryService.ListProducts(ctx, categoryID)
	if err != nil {
		h.log.Errorf("renderCatalog: failed to list category products: %v", err)
		return
	}

	text := texts.CatalogMsg
	var backCallback string

	if categoryID != nil {
		category, catErr := h.categoryService.GetByID(ctx, *categoryID)
		if catErr != nil {
			h.log.Errorf("renderCatalog: failed to get category %d: %v", *categoryID, catErr)
			return
		}

		user, userErr := h.userService.GetProfile(ctx, chatID)
		if userErr != nil {
			h.log.Errorf("renderCatalog: failed to get profile for %d: %v", chatID, userErr)
			return
		}

		text = fmt.Sprintf(texts.CategoryMsg, category.Name, category.Description, user.Balance)
		if category.ParentID != nil {
			backCallback = utils.BuildCategoryCallback(*category.ParentID)
		} else {
			backCallback = utils.CatalogRootCallback
		}
	}

	if len(children) == 0 && len(products) == 0 {
		text = texts.CatalogEmptyMsg
	}

	kb := keyboards.BuildCatalogNavKb(children, products, backCallback)

	if messageID == 0 {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdownV1,
			ReplyMarkup: kb,
		}); err != nil {
			h.log.Errorf("renderCatalog: failed to send message to %d: %v", chatID, err)
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
		h.log.Errorf("renderCatalog: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// ProductHandler открывает карточку товара, редактируя текущее сообщение.
func (h *Handlers) ProductHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID
	callback := update.CallbackQuery.Data

	productID, err := utils.ParseCallbackQuery(callback)
	if err != nil {
		h.log.Errorf("ProductHandler: failed to parse product id from callback %s: %v", callback, err)
		return
	}

	h.renderProductDetail(ctx, b, chatID, messageID, productID)
}

// renderProductDetail шлёт или редактирует карточку товара — общая для
// ProductHandler и возврата из BuyCancelHandler.
func (h *Handlers) renderProductDetail(ctx context.Context, b *bot.Bot, chatID int64, messageID int, productID int64) {
	product, err := h.productService.GetByID(ctx, productID)
	if err != nil {
		h.log.Errorf("renderProductDetail: failed to get product %d: %v", productID, err)
		return
	}

	available, err := h.productService.GetAvailableCount(ctx, productID)
	if err != nil {
		h.log.Errorf("renderProductDetail: failed to get product %d available count: %v", productID, err)
		return
	}

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("renderProductDetail: failed to get profile for %d: %v", chatID, err)
		return
	}

	var backCallback string
	if product.CategoryID != nil {
		backCallback = utils.BuildCategoryCallback(*product.CategoryID)
	} else {
		backCallback = utils.CatalogRootCallback
	}

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: texts.BuyBtn, CallbackData: utils.BuildBuyCallback(productID)}},
			{{Text: texts.BackBtn, CallbackData: backCallback}},
		},
	}
	text := fmt.Sprintf(texts.ProductMsg, product.Name, product.Price, utils.StockIndicator(available), available, product.Description, user.Balance)

	if messageID == 0 {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdownV1,
			ReplyMarkup: kb,
		}); err != nil {
			h.log.Errorf("renderProductDetail: failed to send message to %d: %v", chatID, err)
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
		h.log.Errorf("renderProductDetail: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}
