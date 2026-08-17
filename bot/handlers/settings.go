package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
)

// SettingsHandler — кнопка "⚙️ Настройки" в профиле, показывает инлайн-меню настроек.
func (h *Handlers) SettingsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("SettingsHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        texts.T(user.Language, texts.SettingsMsg, nil),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: h.kb.SettingsKb(user.Language),
	}); err != nil {
		h.log.Errorf("SettingsHandler: failed to send message to %d: %v", chatID, err)
	}
}

// SettingsLanguageHandler — пункт "🌐 Язык" в меню настроек, открывает выбор RU/EN.
func (h *Handlers) SettingsLanguageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("SettingsLanguageHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: texts.T(user.Language, texts.LanguageRUBtn, nil), CallbackData: utils.BuildLanguageCallback(texts.LangRU)}},
			{{Text: texts.T(user.Language, texts.LanguageENBtn, nil), CallbackData: utils.BuildLanguageCallback(texts.LangEN)}},
		},
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        texts.T(user.Language, texts.LanguagePickerMsg, nil),
		ReplyMarkup: kb,
	}); err != nil {
		h.log.Errorf("SettingsLanguageHandler: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}
}

// LanguageSetHandler — выбор языка сделан: сохраняем, подтверждаем на новом
// языке, возвращаем в главное меню (новой reply-клавиатурой на выбранном языке).
func (h *Handlers) LanguageSetHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	lang, err := utils.ParseLanguageCallback(update.CallbackQuery.Data)
	if err != nil || !texts.IsSupported(lang) {
		h.log.Errorf("LanguageSetHandler: invalid language callback %q: %v", update.CallbackQuery.Data, err)
		return
	}

	if err = h.userService.SetLanguage(ctx, chatID, lang); err != nil {
		h.log.Errorf("LanguageSetHandler: failed to set language for %d: %v", chatID, err)
		return
	}

	if _, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        texts.T(lang, texts.LanguageSetMsg, nil),
		ReplyMarkup: nil,
	}); err != nil {
		h.log.Errorf("LanguageSetHandler: failed to edit message %d in chat %d: %v", messageID, chatID, err)
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        texts.T(lang, texts.StartMsg, map[string]any{"Name": utils.EscapeMarkdown(update.CallbackQuery.From.FirstName)}),
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: h.kb.MainMenuKb(lang),
	}); err != nil {
		h.log.Errorf("LanguageSetHandler: failed to send menu to %d: %v", chatID, err)
	}
}
