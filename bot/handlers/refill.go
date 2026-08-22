package handlers

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
	domain "github.com/trottling/Telegram-Store/internal/domain/models"
)

// emptyInlineKeyboard — см. комментарий в RefillMerchantHandler: nil-слайс в
// InlineKeyboardMarkup уходит в JSON как null, Telegram API его не принимает.
var emptyInlineKeyboard = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{}}

// refillMerchants — мерчанты, доступные для выбора в боте, в порядке показа.
// MerchantReferral сюда не входит — начисления оттуда не создаются через CreateInvoice.
var refillMerchants = []struct {
	Merchant domain.Merchant
	Btn      string
}{
	{domain.MerchantCrystalPay, texts.CrystalPayBtn},
	{domain.MerchantYooKassa, texts.YooKassaBtn},
	{domain.MerchantTinkoff, texts.TinkoffBtn},
	{domain.MerchantDummy, texts.DummyBtn},
}

// merchantLimits — общая для всех мерчантов часть настроек (у каждого свои
// креды, но Enabled/Min/MaxAmount совпадают по смыслу).
type merchantLimits struct {
	Enabled bool
	Min     float64
	Max     float64
}

func merchantConfig(settings *domain.Settings, merchant domain.Merchant) merchantLimits {
	switch merchant {
	case domain.MerchantCrystalPay:
		return merchantLimits{settings.CrystalPay.Enabled, settings.CrystalPay.MinAmount, settings.CrystalPay.MaxAmount}
	case domain.MerchantYooKassa:
		return merchantLimits{settings.YooKassa.Enabled, settings.YooKassa.MinAmount, settings.YooKassa.MaxAmount}
	case domain.MerchantTinkoff:
		return merchantLimits{settings.Tinkoff.Enabled, settings.Tinkoff.MinAmount, settings.Tinkoff.MaxAmount}
	case domain.MerchantDummy:
		return merchantLimits{settings.Dummy.Enabled, settings.Dummy.MinAmount, settings.Dummy.MaxAmount}
	default:
		return merchantLimits{}
	}
}

// amountRangeHint — подсказка допустимой суммы для AskRefillAmountMsg.
func amountRangeHint(lang string, mc merchantLimits) string {
	switch {
	case mc.Min > 0 && mc.Max > 0:
		return texts.T(lang, texts.AmountRangeBothMsg, map[string]any{"Min": fmt.Sprintf("%.2f", mc.Min), "Max": fmt.Sprintf("%.2f", mc.Max)})
	case mc.Min > 0:
		return texts.T(lang, texts.AmountRangeMinMsg, map[string]any{"Min": fmt.Sprintf("%.2f", mc.Min)})
	case mc.Max > 0:
		return texts.T(lang, texts.AmountRangeMaxMsg, map[string]any{"Max": fmt.Sprintf("%.2f", mc.Max)})
	default:
		return texts.T(lang, texts.AmountRangeAnyMsg, nil)
	}
}

// RefillBalanceHandler показывает включённых мерчантов для пополнения.
func (h *Handlers) RefillBalanceHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.renderMerchantPicker(ctx, b, update.Message.Chat.ID)
}

func (h *Handlers) renderMerchantPicker(ctx context.Context, b *bot.Bot, chatID int64) {
	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("renderMerchantPicker: failed to get profile for %d: %v", chatID, err)
		return
	}

	settings, err := h.settingsService.Get(ctx)
	if err != nil {
		h.log.Errorf("renderMerchantPicker: failed to get settings: %v", err)
		return
	}

	var rows [][]models.InlineKeyboardButton
	for _, mc := range refillMerchants {
		if merchantConfig(settings, mc.Merchant).Enabled {
			rows = append(rows, []models.InlineKeyboardButton{{Text: texts.T(user.Language, mc.Btn, nil), CallbackData: utils.BuildRefillMerchantCallback(string(mc.Merchant))}})
		}
	}

	if len(rows) == 0 {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: texts.T(user.Language, texts.RefillMsg, nil)}); err != nil {
			h.log.Errorf("renderMerchantPicker: failed to send message to %d: %v", chatID, err)
		}
		return
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        texts.T(user.Language, texts.RefillMerchantPickerMsg, nil),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
	}); err != nil {
		h.log.Errorf("renderMerchantPicker: failed to send message to %d: %v", chatID, err)
	}
}

// RefillMerchantHandler — мерчант выбран, спрашиваем сумму (с подсказкой min/max).
func (h *Handlers) RefillMerchantHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, messageID, ok := utils.CallbackTarget(update)
	if !ok {
		return
	}

	merchantStr, err := utils.ParseRefillMerchantCallback(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("RefillMerchantHandler: failed to parse merchant callback: %v", err)
		return
	}
	merchant := domain.Merchant(merchantStr)

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("RefillMerchantHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	settings, err := h.settingsService.Get(ctx)
	if err != nil {
		h.log.Errorf("RefillMerchantHandler: failed to get settings: %v", err)
		return
	}

	mc := merchantConfig(settings, merchant)
	if !mc.Enabled {
		// Могли выключить между показом списка и тапом — покажем список заново.
		h.renderMerchantPicker(ctx, b, chatID)
		return
	}

	st := &domainfsm.State{Step: domainfsm.StepAwaitingRefillAmount, Merchant: string(merchant), MessageID: messageID}
	if err = h.stateStore.SetFSMState(ctx, chatID, st); err != nil {
		h.log.Errorf("RefillMerchantHandler: failed to set refill-amount state for %d: %v", chatID, err)
		return
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text: texts.T(user.Language, texts.AskRefillAmountMsg, map[string]any{"Hint": amountRangeHint(user.Language, mc)}),
		// emptyInlineKeyboard, не nil: правка убирает клавиатуру карточки
		// мерчанта, а nil в этом поле уходит в JSON как null и Telegram API
		// отвечает "inline_keyboard must be of type Array" — FSM-состояние на
		// ввод суммы при этом уже выставлено, и повтор с ним не сработает молча.
		ReplyMarkup: emptyInlineKeyboard,
	}); err != nil {
		h.log.Errorf("RefillMerchantHandler: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// CheckPaymentHandler — кнопка «Проверить оплату» под счётом. Вебхук мерчанта
// остаётся основным путём подтверждения; это подстраховка на случай, если он
// ещё не пришёл или потерялся (см. ReplenishmentService.CheckInvoice).
// Само сообщение со счётом не трогаем — ни текст, ни кнопки: результат
// показываем только всплывающим уведомлением поверх экрана (AnswerCallbackQuery,
// см. AnswerCallback — CheckPaymentCallbackPrefix там как раз исключён из
// автоматического пустого ack ради этого). Кнопки должны остаться рабочими и
// после "оплачено": пользователь мог оплатить ещё раз по ошибке или мимо кассы.
func (h *Handlers) CheckPaymentHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, ok := utils.CallbackChatID(update)
	if !ok {
		return
	}

	answer := func(text string) {
		if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            text,
		}); err != nil {
			h.log.Errorf("CheckPaymentHandler: failed to answer callback query %s: %v", update.CallbackQuery.ID, err)
		}
	}

	replenishmentID, err := utils.ParseCallbackQuery(update.CallbackQuery.Data)
	if err != nil {
		h.log.Errorf("CheckPaymentHandler: failed to parse callback: %v", err)
		return
	}

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("CheckPaymentHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	status, amount, err := h.replenishmentService.CheckInvoice(ctx, chatID, replenishmentID)
	if err != nil {
		h.log.Errorw("CheckPaymentHandler: check failed", "error", err, "telegram_id", chatID, "replenishment_id", replenishmentID)
		answer(texts.T(user.Language, texts.RefillCheckErrorMsg, nil))
		return
	}

	switch status {
	case domain.ReplenishmentStatusPaid:
		answer(texts.T(user.Language, texts.CheckPaymentPaidMsg, map[string]any{"Amount": utils.FormatAmount(amount)}))
	case domain.ReplenishmentStatusFailed, domain.ReplenishmentStatusCancelled:
		answer(texts.T(user.Language, texts.CheckPaymentFailedMsg, nil))
	default:
		answer(texts.T(user.Language, texts.CheckPaymentPendingMsg, nil))
	}
}
