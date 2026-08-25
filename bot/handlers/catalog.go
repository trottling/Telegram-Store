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

// CatalogHandler открывает корень каталога.
func (h *Handlers) CatalogHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.renderCatalog(ctx, b, domainmodels.TelegramID(update.Message.Chat.ID), nil, 0)
}

// CatalogRootHandler — кнопка «в корень каталога» из глубины дерева.
func (h *Handlers) CatalogRootHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, messageID, ok := utils.CallbackTarget(update)
	if !ok {
		return
	}
	h.renderCatalog(ctx, b, chatID, nil, messageID)
}

// CategoryHandler показывает подкатегории и товары одной категории.
func (h *Handlers) CategoryHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, messageID, ok := utils.CallbackTarget(update)
	if !ok {
		return
	}

	categoryID, err := utils.ParseCategoryCallback(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("CategoryHandler: failed to parse category callback: %v", err)
		return
	}

	h.renderCatalog(ctx, b, chatID, &categoryID, messageID)
}

// renderCatalog: messageID == 0 — новое сообщение, иначе редактирует текущее.
func (h *Handlers) renderCatalog(ctx context.Context, b *bot.Bot, chatID domainmodels.TelegramID, categoryID *domainmodels.CategoryID, messageID int) {
	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("renderCatalog: failed to get profile for %d: %v", chatID, err)
		return
	}

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

	text := texts.T(user.Language, texts.CatalogMsg, nil)
	var backCallback string

	if categoryID != nil {
		category, catErr := h.categoryService.GetByID(ctx, *categoryID)
		if catErr != nil {
			h.log.Errorf("renderCatalog: failed to get category %d: %v", *categoryID, catErr)
			return
		}

		text = texts.T(user.Language, texts.CategoryMsg, map[string]any{
			"Name":        utils.EscapeMarkdown(category.Name),
			"Description": utils.EscapeMarkdown(category.Description),
			"Balance":     utils.FormatAmount(user.Balance()),
		})
		if category.ParentID != nil {
			backCallback = utils.BuildCategoryCallback(*category.ParentID)
		} else {
			backCallback = utils.CatalogRootCallback
		}
	}

	if len(children) == 0 && len(products) == 0 {
		text = texts.T(user.Language, texts.CatalogEmptyMsg, nil)
	}

	kb := keyboards.BuildCatalogNavKb(user.Language, children, products, backCallback)

	if messageID == 0 {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdown,
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
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: kb,
	}); err != nil {
		h.log.Errorf("renderCatalog: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// ProductHandler открывает карточку товара, редактируя текущее сообщение.
func (h *Handlers) ProductHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, messageID, ok := utils.CallbackTarget(update)
	if !ok {
		return
	}
	callback := update.CallbackQuery.Data

	productID, err := utils.ParseProductCallback(callback)
	if err != nil {
		h.log.Errorf("ProductHandler: failed to parse product id from callback %s: %v", callback, err)
		return
	}

	h.renderProductDetail(ctx, b, chatID, messageID, productID)
}

// renderProductDetail шлёт или редактирует карточку товара — общая для
// ProductHandler и возврата из BuyCancelHandler.
func (h *Handlers) renderProductDetail(ctx context.Context, b *bot.Bot, chatID domainmodels.TelegramID, messageID int, productID domainmodels.ProductID) {
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
			{{Text: texts.T(user.Language, texts.BuyBtn, nil), CallbackData: utils.BuildBuyCallback(productID)}},
			{{Text: texts.T(user.Language, texts.BackBtn, nil), CallbackData: backCallback}},
		},
	}
	text := texts.T(user.Language, texts.ProductMsg, map[string]any{
		"Name":           utils.EscapeMarkdown(product.Name),
		"Price":          utils.FormatAmount(product.Price),
		"StockIndicator": utils.StockIndicator(available),
		"Available":      available,
		"Description":    utils.EscapeMarkdown(product.Description),
		"Balance":        utils.FormatAmount(user.Balance()),
	})

	if messageID == 0 {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdown,
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
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: kb,
	}); err != nil {
		h.log.Errorf("renderProductDetail: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}
