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

// refillMerchants — мерчанты, доступные для выбора в боте, в порядке показа.
// MerchantReferral сюда не входит — начисления оттуда не создаются через CreateInvoice.
var refillMerchants = []struct {
	Merchant domain.Merchant
	Btn      string
}{
	{domain.MerchantCrystalPay, texts.CrystalPayBtn},
	{domain.MerchantYooKassa, texts.YooKassaBtn},
	{domain.MerchantTinkoff, texts.TinkoffBtn},
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
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        texts.T(user.Language, texts.AskRefillAmountMsg, map[string]any{"Hint": amountRangeHint(user.Language, mc)}),
		ReplyMarkup: &models.InlineKeyboardMarkup{},
	}); err != nil {
		h.log.Errorf("RefillMerchantHandler: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}
