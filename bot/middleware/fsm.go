package middleware

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/bot/texts"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
)

// maxQuickPickQuantity дублирует handlers.maxQuickPickQuantity.
const maxQuickPickQuantity = 5

// knownButtonTexts — reply-кнопки; если пришла одна из них во время
// ожидания шага, пользователь ушёл из сценария, парсить как число не нужно.
var knownButtonTexts = map[string]bool{
	texts.HelpBtn:          true,
	texts.CatalogBtn:       true,
	texts.ProfileBtn:       true,
	texts.PurchasesBtn:     true,
	texts.RefillBalanceBtn: true,
	texts.StartMenuBtn:     true,
}

func isEscapeHatch(text string) bool {
	return strings.HasPrefix(text, "/") || knownButtonTexts[text]
}

// FSM перехватывает текст, пока чат ждёт ответа на шаг сценария (количество, сумма).
func (m *Middlewares) FSM(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.Text == "" {
			next(ctx, b, update)
			return
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		st, err := m.stateStore.GetFSMState(ctx, chatID)
		if err != nil {
			// ErrNotFound — обычный случай (нет ожидаемого шага), остальные ошибки просто логируем.
			if !errors.Is(err, domainfsm.ErrNotFound) {
				m.log.WithError(err).WithField("telegram_id", chatID).Error("fsm: failed to read state")
			}
			next(ctx, b, update)
			return
		}

		if isEscapeHatch(text) {
			_ = m.stateStore.ClearFSMState(ctx, chatID)
			next(ctx, b, update)
			return
		}

		switch st.Step {
		case domainfsm.StepAwaitingBuyQuantity:
			m.handleBuyQuantity(ctx, b, chatID, text, st)
		case domainfsm.StepAwaitingRefillAmount:
			m.handleRefillAmount(ctx, b, chatID, text)
		default:
			_ = m.stateStore.ClearFSMState(ctx, chatID)
			next(ctx, b, update)
		}
	}
}

// send отправляет ответ и логирует ошибку — общий хвост для каждого шага FSM.
// kb присваивается только если не nil: иначе типизированный nil в
// интерфейсе ReplyMarkup не считается пустым, и Bot API получит "reply_markup": null.
func (m *Middlewares) send(ctx context.Context, b *bot.Bot, chatID int64, text string, kb *models.InlineKeyboardMarkup) {
	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdownV1,
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		m.log.Errorf("fsm: failed to send message to %d: %v", chatID, err)
	}
}

// handleBuyQuantity обрабатывает введённое вручную количество.
func (m *Middlewares) handleBuyQuantity(ctx context.Context, b *bot.Bot, chatID int64, text string, st *domainfsm.State) {
	qty, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || qty <= 0 {
		// Состояние не сбрасываем — даём попробовать ещё раз.
		m.send(ctx, b, chatID, texts.InvalidQuantityMsg, nil)
		return
	}

	m.showBuyConfirmation(ctx, b, chatID, st.MessageID, st.ProductID, qty)
}

// showBuyConfirmation дублирует handlers.Handlers.showBuyConfirmation —
// пакеты Middlewares и Handlers связаны независимо, шарить смысла нет.
func (m *Middlewares) showBuyConfirmation(ctx context.Context, b *bot.Bot, chatID int64, messageID int, productID int64, qty int) {
	product, err := m.productService.GetByID(ctx, productID)
	if err != nil {
		m.log.WithError(err).WithField("product_id", productID).Error("fsm: failed to get product for confirmation")
		return
	}

	available, err := m.productService.GetAvailableCount(ctx, productID)
	if err != nil {
		m.log.WithError(err).WithField("product_id", productID).Error("fsm: failed to get available count for confirmation")
		return
	}
	if qty > available {
		if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        fmt.Sprintf(texts.InsufficientStockMsg, available),
			ParseMode:   models.ParseModeMarkdownV1,
			ReplyMarkup: keyboards.BuildQuantityKb(maxQuickPickQuantity),
		}); err != nil {
			m.log.Errorf("fsm: failed to edit message %d in chat %d: %v", messageID, chatID, err)
		}
		return
	}

	st := &domainfsm.State{Step: domainfsm.StepAwaitingBuyConfirmation, ProductID: productID, Quantity: qty, MessageID: messageID}
	if err = m.stateStore.SetFSMState(ctx, chatID, st); err != nil {
		m.log.WithError(err).WithField("telegram_id", chatID).Error("fsm: failed to set buy-confirmation state")
		return
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        fmt.Sprintf(texts.ConfirmPurchaseMsg, product.Name, qty, product.Price*float64(qty)),
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: keyboards.BuildBuyConfirmKb(),
	}); err != nil {
		m.log.Errorf("fsm: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// handleRefillAmount парсит сумму и запрашивает счёт у paymentService
// (сейчас это заглушка, всегда падает в ветку «недоступно»).
func (m *Middlewares) handleRefillAmount(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	normalized := strings.ReplaceAll(strings.TrimSpace(text), ",", ".")
	amount, err := strconv.ParseFloat(normalized, 64)
	if err != nil || amount <= 0 {
		// Состояние не сбрасываем — даём попробовать ещё раз.
		m.send(ctx, b, chatID, texts.InvalidAmountMsg, nil)
		return
	}

	_ = m.stateStore.ClearFSMState(ctx, chatID)

	paymentURL, _, err := m.paymentService.CreateInvoice(ctx, chatID, amount, "Пополнение баланса")
	if err != nil {
		m.log.WithError(err).WithField("telegram_id", chatID).Info("fsm: refill invoice unavailable")
		m.send(ctx, b, chatID, texts.RefillMsg, nil)
		return
	}

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{{Text: texts.PayBtn, URL: paymentURL}}},
	}
	m.send(ctx, b, chatID, fmt.Sprintf(texts.RefillInvoiceMsg, amount), kb)
}
