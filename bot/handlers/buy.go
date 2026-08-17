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
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
)

const maxQuickPickQuantity = 5

// BuyHandler ничего не покупает — переводит карточку в запрос количества и
// ставит чат в состояние ожидания. Списывает деньги только BuyConfirmHandler.
func (h *Handlers) BuyHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("BuyHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	productID, err := utils.ParseCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("BuyHandler: failed to parse buy callback: %v", err)
		return
	}

	product, err := h.productService.GetByID(ctx, productID)
	if err != nil {
		h.log.Errorf("BuyHandler: failed to get product %d: %v", productID, err)
		if _, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: texts.UserFacingError(user.Language, err)}); sendErr != nil {
			h.log.Errorf("BuyHandler: failed to send message to %d: %v", chatID, sendErr)
		}
		return
	}

	st := &domainfsm.State{Step: domainfsm.StepAwaitingBuyQuantity, ProductID: productID, MessageID: messageID}
	if err = h.stateStore.SetFSMState(ctx, chatID, st); err != nil {
		h.log.Errorf("BuyHandler: failed to set buy-quantity state for %d: %v", chatID, err)
		return
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        texts.T(user.Language, texts.AskQuantityMsg, map[string]any{"Name": utils.EscapeMarkdown(product.Name)}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: keyboards.BuildQuantityKb(user.Language, maxQuickPickQuantity),
	}); err != nil {
		h.log.Errorf("BuyHandler: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// BuyQtyHandler обрабатывает тап по быстрому выбору количества 1..5.
func (h *Handlers) BuyQtyHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	qty, err := utils.ParseCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("BuyQtyHandler: failed to parse buy-quantity callback: %v", err)
		return
	}

	st, err := h.stateStore.GetFSMState(ctx, chatID)
	if err != nil || st.Step != domainfsm.StepAwaitingBuyQuantity {
		// Устаревшая кнопка от старого экрана — игнорируем.
		return
	}

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("BuyQtyHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	h.showBuyConfirmation(ctx, b, chatID, user.Language, messageID, st.ProductID, int(qty))
}

// showBuyConfirmation — экран подтверждения покупки. Проверка остатка тут
// нужна только для UX, реальную гарантию даёт транзакция в PurchaseService.Buy.
func (h *Handlers) showBuyConfirmation(ctx context.Context, b *bot.Bot, chatID int64, lang string, messageID int, productID int64, qty int) {
	product, err := h.productService.GetByID(ctx, productID)
	if err != nil {
		h.log.Errorf("showBuyConfirmation: failed to get product %d: %v", productID, err)
		return
	}

	available, err := h.productService.GetAvailableCount(ctx, productID)
	if err != nil {
		h.log.Errorf("showBuyConfirmation: failed to get product %d available count: %v", productID, err)
		return
	}
	if qty > available {
		if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        texts.T(lang, texts.InsufficientStockMsg, map[string]any{"Available": available}),
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: keyboards.BuildQuantityKb(lang, maxQuickPickQuantity),
		}); err != nil {
			h.log.Errorf("showBuyConfirmation: failed to edit message %d in chat %d: %v", messageID, chatID, err)
		}
		return
	}

	st := &domainfsm.State{Step: domainfsm.StepAwaitingBuyConfirmation, ProductID: productID, Quantity: qty, MessageID: messageID}
	if err = h.stateStore.SetFSMState(ctx, chatID, st); err != nil {
		h.log.Errorf("showBuyConfirmation: failed to set buy-confirmation state for %d: %v", chatID, err)
		return
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text: texts.T(lang, texts.ConfirmPurchaseMsg, map[string]any{
			"Name":     utils.EscapeMarkdown(product.Name),
			"Quantity": qty,
			"Amount":   utils.FormatAmount(product.Price * float64(qty)),
		}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: keyboards.BuildBuyConfirmKb(lang),
	}); err != nil {
		h.log.Errorf("showBuyConfirmation: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// BuyCancelHandler возвращает с экрана количества/подтверждения к карточке товара.
func (h *Handlers) BuyCancelHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	st, err := h.stateStore.GetFSMState(ctx, chatID)
	if err != nil {
		return
	}
	_ = h.stateStore.ClearFSMState(ctx, chatID)

	h.renderProductDetail(ctx, b, chatID, messageID, st.ProductID)
}

// BuyConfirmHandler — единственное место, где реально происходит покупка.
func (h *Handlers) BuyConfirmHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	st, err := h.stateStore.GetFSMState(ctx, chatID)
	if err != nil || st.Step != domainfsm.StepAwaitingBuyConfirmation {
		return
	}
	_ = h.stateStore.ClearFSMState(ctx, chatID)

	if _, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{ChatID: chatID, MessageID: messageID}); err != nil {
		h.log.Errorf("BuyConfirmHandler: failed to strip keyboard from message %d in chat %d: %v", messageID, chatID, err)
	}

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("BuyConfirmHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	purchases, credit, err := h.purchaseService.Buy(ctx, chatID, st.ProductID, st.Quantity)
	if err != nil {
		h.log.Errorf("BuyConfirmHandler: failed to buy product %d x%d for user %d: %v", st.ProductID, st.Quantity, chatID, err)
		if _, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        texts.UserFacingError(user.Language, err),
			ReplyMarkup: h.kb.MainMenuKb(user.Language),
		}); sendErr != nil {
			h.log.Errorf("BuyConfirmHandler: failed to send message to %d: %v", chatID, sendErr)
		}
		return
	}

	contents := make([]string, len(purchases))
	for i, p := range purchases {
		content := ""
		if p.Item != nil {
			content = p.Item.Content
		}
		contents[i] = fmt.Sprintf("`%s`", utils.EscapeMarkdownCode(content))
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: texts.T(user.Language, texts.ProductBoughtMsg, map[string]any{
			"Count":       len(purchases),
			"Description": utils.EscapeMarkdown(purchases[0].Product.Description),
			"Content":     strings.Join(contents, "\n"),
		}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: h.kb.MainMenuKb(user.Language),
	}); err != nil {
		h.log.Errorf("BuyConfirmHandler: failed to send message to %d: %v", chatID, err)
	}

	if credit != nil {
		// Уведомление рефереру — на его собственном языке, не языке покупателя.
		referrerLang := texts.LangRU
		if referrer, refErr := h.userService.GetProfile(ctx, credit.ReferrerID); refErr == nil {
			referrerLang = referrer.Language
		}
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    credit.ReferrerID,
			Text:      texts.T(referrerLang, texts.ReferralCreditMsg, map[string]any{"Amount": utils.FormatAmount(credit.Amount)}),
			ParseMode: models.ParseModeMarkdown,
		}); err != nil {
			h.log.Errorf("BuyConfirmHandler: failed to notify referrer %d: %v", credit.ReferrerID, err)
		}
	}
}
