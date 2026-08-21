package handlers

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	botmetrics "github.com/trottling/Telegram-Store/internal/metrics/bot"
)

// StartHandler — единственное место, где создаётся запись пользователя.
// Payload реф-ссылки (t.me/<bot>?start=<id> -> текст "/start <id>") учитывается
// только на ветке создания — см. UserSrv.GetOrCreate.
func (h *Handlers) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	lang := texts.Normalize(update.Message.From.LanguageCode)

	user, created, err := h.userService.GetOrCreate(ctx, chatID, update.Message.Chat.Username, parseStartPayload(update.Message.Text), lang)
	if err != nil {
		h.log.Errorf("StartHandler: failed to get or create user %d: %v", chatID, err)
		return
	}
	if created {
		// user.ReferrerID — уже провалидированный UserSrv.validReferrer результат,
		// не сырой payload: тут не "пришёл по ссылке", а "реально засчитан рефералом".
		source := "organic"
		if user.ReferrerID != nil {
			source = "referral"
		}
		botmetrics.UsersRegisteredTotal.WithLabelValues(source).Inc()
	}

	// user.Language — для существующего пользователя это уже сохранённое
	// (возможно вручную переключённое) значение, не свежая телеграм-локаль.
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        texts.T(user.Language, texts.StartMsg, map[string]any{"Name": utils.EscapeMarkdown(update.Message.From.FirstName)}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: h.kb.MainMenuKb(user.Language),
	}); err != nil {
		h.log.Errorf("StartHandler: failed to send message: %v", err)
	}
}

// parseStartPayload вытаскивает referrerID из "/start <id>" — MatchTypeCommandStartOnly
// матчит и "/start", и "/start 123" (Telegram кладёт payload deep-link'а как
// обычный текст после команды). nil, если payload пуст или не число.
func parseStartPayload(text string) *int64 {
	payload := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
	if payload == "" {
		return nil
	}
	id, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}

// StartMenuHandler отвечает на кнопку «На главную» — только новым сообщением, редактировать нечего.
func (h *Handlers) StartMenuHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("StartMenuHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        texts.T(user.Language, texts.StartMsg, map[string]any{"Name": utils.EscapeMarkdown(update.Message.From.FirstName)}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: h.kb.MainMenuKb(user.Language),
	}); err != nil {
		h.log.Errorf("StartMenuHandler: failed to send message to %d: %v", chatID, err)
	}
}

// MainMenuHandler — кнопка «В главное меню», показывается только в корне каталога.
func (h *Handlers) MainMenuHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, ok := utils.CallbackChatID(update)
	if !ok {
		return
	}

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("MainMenuHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        texts.T(user.Language, texts.StartMsg, map[string]any{"Name": utils.EscapeMarkdown(update.CallbackQuery.From.FirstName)}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: h.kb.MainMenuKb(user.Language),
	}); err != nil {
		h.log.Errorf("MainMenuHandler: failed to send message to %d: %v", chatID, err)
	}
}

// HelpHandler показывает экран помощи с Username поддержки из настроек.
func (h *Handlers) HelpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("HelpHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	settings, err := h.settingsService.Get(ctx)
	if err != nil {
		h.log.Errorf("HelpHandler: failed to get settings: %v", err)
		return
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        texts.T(user.Language, texts.HelpMsg, map[string]any{"SupportUsername": utils.EscapeMarkdown(settings.SupportUsername)}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: h.kb.MainMenuKb(user.Language),
	}); err != nil {
		h.log.Errorf("HelpHandler: failed to send message to %d: %v", chatID, err)
	}
}
