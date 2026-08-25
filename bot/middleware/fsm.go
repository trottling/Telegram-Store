package middleware

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
	domain "github.com/trottling/Telegram-Store/internal/domain/models"
	domainservice "github.com/trottling/Telegram-Store/internal/domain/service"
)

// maxQuickPickQuantity дублирует handlers.maxQuickPickQuantity.
const maxQuickPickQuantity = 5

// knownButtonIDs — reply-кнопки; если во время ожидания шага пришла подпись
// одной из них (на любом поддерживаемом языке), пользователь ушёл из
// сценария, парсить текст как число/сумму не нужно. Список должен покрывать
// все reply-кнопки, зарегистрированные в bot.go — пропущенная тут кнопка
// вместо своего действия попадала бы в handleBuyQuantity/handleRefillAmount
// и парсилась бы как число/сумма, с закономерным "введите положительное число".
var knownButtonIDs = []string{
	texts.HelpBtn, texts.CatalogBtn, texts.ProfileBtn,
	texts.PurchasesBtn, texts.RefillBalanceBtn, texts.StartMenuBtn,
	texts.ProfileRefreshBtn, texts.ReplenishmentsBtn, texts.ReferralBtn, texts.SettingsBtn,
}

var knownButtonTexts = buildKnownButtonTexts()

func buildKnownButtonTexts() map[string]bool {
	m := make(map[string]bool, len(knownButtonIDs)*len(texts.SupportedLanguages))
	for _, id := range knownButtonIDs {
		for _, lang := range texts.SupportedLanguages {
			m[texts.T(lang, id, nil)] = true
		}
	}
	return m
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

		chatID := domain.TelegramID(update.Message.Chat.ID)
		text := update.Message.Text

		st, err := m.stateStore.GetFSMState(ctx, chatID)
		if err != nil {
			// ErrNotFound — обычный случай (нет ожидаемого шага), остальные ошибки просто логируем.
			if !errors.Is(err, domainfsm.ErrNotFound) {
				m.log.Errorw("fsm: failed to read state", "error", err, "telegram_id", chatID)
			}
			next(ctx, b, update)
			return
		}

		if isEscapeHatch(text) {
			_ = m.stateStore.ClearFSMState(ctx, chatID)
			next(ctx, b, update)
			return
		}

		lang := m.resolveLang(ctx, chatID)

		switch st.Step {
		case domainfsm.StepAwaitingBuyQuantity:
			m.handleBuyQuantity(ctx, b, chatID, lang, text, st)
		case domainfsm.StepAwaitingRefillAmount:
			m.handleRefillAmount(ctx, b, chatID, lang, text, st)
		default:
			_ = m.stateStore.ClearFSMState(ctx, chatID)
			next(ctx, b, update)
		}
	}
}

// resolveLang читает язык из профиля пользователя; при ошибке — ru
// (fail-safe, не должен блокировать сценарий FSM).
func (m *Middlewares) resolveLang(ctx context.Context, chatID domain.TelegramID) string {
	user, err := m.userService.GetProfile(ctx, chatID)
	if err != nil {
		return texts.LangRU
	}
	return user.Language
}

// send отправляет ответ и логирует ошибку — общий хвост для каждого шага FSM.
// kb присваивается только если не nil: иначе типизированный nil в
// интерфейсе ReplyMarkup не считается пустым, и Bot API получит "reply_markup": null.
func (m *Middlewares) send(ctx context.Context, b *bot.Bot, chatID domain.TelegramID, text string, kb *models.InlineKeyboardMarkup) {
	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		m.log.Errorf("fsm: failed to send message to %d: %v", chatID, err)
	}
}

// handleBuyQuantity обрабатывает введённое вручную количество.
func (m *Middlewares) handleBuyQuantity(ctx context.Context, b *bot.Bot, chatID domain.TelegramID, lang, text string, st *domainfsm.State) {
	qty, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || qty <= 0 {
		// Состояние не сбрасываем — даём попробовать ещё раз.
		m.send(ctx, b, chatID, texts.T(lang, texts.InvalidQuantityMsg, nil), nil)
		return
	}
	// Потолок проверяем здесь же, а не только в Buy: иначе пользователь видел
	// экран подтверждения с суммой за 500 штук и получал отказ лишь после того,
	// как нажмёт «Подтвердить».
	if qty > domainservice.MaxBuyQuantity {
		m.send(ctx, b, chatID, texts.T(lang, texts.ErrTooManyProductsMsg, nil), nil)
		return
	}

	m.showBuyConfirmation(ctx, b, chatID, lang, st.MessageID, st.ProductID, qty)
}

// showBuyConfirmation дублирует handlers.Handlers.showBuyConfirmation —
// пакеты Middlewares и Handlers связаны независимо, шарить смысла нет.
func (m *Middlewares) showBuyConfirmation(ctx context.Context, b *bot.Bot, chatID domain.TelegramID, lang string, messageID int, productID domain.ProductID, qty int) {
	// При ошибке отвечаем пользователю: он только что ввёл количество, и молчание
	// в ответ выглядит как проглоченный ввод.
	product, err := m.productService.GetByID(ctx, productID)
	if err != nil {
		m.log.Errorw("fsm: failed to get product for confirmation", "error", err, "product_id", productID)
		m.send(ctx, b, chatID, texts.UserFacingError(lang, err), nil)
		return
	}

	available, err := m.productService.GetAvailableCount(ctx, productID)
	if err != nil {
		m.log.Errorw("fsm: failed to get available count for confirmation", "error", err, "product_id", productID)
		m.send(ctx, b, chatID, texts.UserFacingError(lang, err), nil)
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
			m.log.Errorf("fsm: failed to edit message %d in chat %d: %v", messageID, chatID, err)
		}
		return
	}

	st := &domainfsm.State{Step: domainfsm.StepAwaitingBuyConfirmation, ProductID: productID, Quantity: qty, MessageID: messageID}
	if err = m.stateStore.SetFSMState(ctx, chatID, st); err != nil {
		m.log.Errorw("fsm: failed to set buy-confirmation state", "error", err, "telegram_id", chatID)
		return
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text: texts.T(lang, texts.ConfirmPurchaseMsg, map[string]any{
			"Name":     utils.EscapeMarkdown(product.Name),
			"Quantity": qty,
			"Amount":   utils.FormatAmount(product.Price.Mul(qty)),
		}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: keyboards.BuildBuyConfirmKb(lang),
	}); err != nil {
		m.log.Errorf("fsm: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// handleRefillAmount парсит сумму и создаёт счёт через replenishmentService
// у мерчанта, выбранного на предыдущем шаге (st.Merchant).
func (m *Middlewares) handleRefillAmount(ctx context.Context, b *bot.Bot, chatID domain.TelegramID, lang, text string, st *domainfsm.State) {
	normalized := strings.ReplaceAll(strings.TrimSpace(text), ",", ".")
	amount, err := domain.NewMoney(normalized)
	if err != nil || amount.IsZero() {
		// Состояние не сбрасываем — даём попробовать ещё раз.
		m.send(ctx, b, chatID, texts.T(lang, texts.InvalidAmountMsg, nil), nil)
		return
	}

	merchant := domain.Merchant(st.Merchant)
	paymentURL, replenishmentID, err := m.replenishmentService.CreateInvoice(ctx, chatID, merchant, amount)
	if err != nil {
		if errors.Is(err, domainerrors.ErrAmountOutOfRange) {
			// Сумма вне min/max — состояние не сбрасываем, дадим ввести другую.
			m.send(ctx, b, chatID, texts.T(lang, texts.InvalidAmountMsg, nil), nil)
			return
		}
		m.log.Infow("fsm: refill invoice unavailable", "error", err, "telegram_id", chatID)
		_ = m.stateStore.ClearFSMState(ctx, chatID)
		m.send(ctx, b, chatID, texts.T(lang, texts.RefillMsg, nil), nil)
		return
	}

	_ = m.stateStore.ClearFSMState(ctx, chatID)

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: texts.T(lang, texts.PayBtn, nil), URL: paymentURL}},
			{{Text: texts.T(lang, texts.CheckPaymentBtn, nil), CallbackData: utils.BuildCheckPaymentCallback(replenishmentID)}},
		},
	}
	m.send(ctx, b, chatID, texts.T(lang, texts.RefillInvoiceMsg, map[string]any{"Amount": utils.FormatAmount(amount)}), kb)
}
