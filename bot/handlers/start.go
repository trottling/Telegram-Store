package handlers

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/TG-Store/bot/texts"
)

// StartHandler — единственное место, где создаётся запись пользователя.
func (h *Handlers) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	if _, err := h.userService.GetOrCreate(ctx, chatID, update.Message.Chat.Username); err != nil {
		h.log.Errorf("StartHandler: failed to get or create user %d: %v", chatID, err)
		return
	}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf(texts.StartMsg, update.Message.From.FirstName),
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: h.kb.MainMenuKb,
	}); err != nil {
		h.log.Errorf("StartHandler: failed to send message: %v", err)
	}
}

// StartMenuHandler отвечает на кнопку «На главную» — только новым сообщением, редактировать нечего.
func (h *Handlers) StartMenuHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf(texts.StartMsg, update.Message.From.FirstName),
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: h.kb.MainMenuKb,
	}); err != nil {
		h.log.Errorf("StartMenuHandler: failed to send message to %d: %v", chatID, err)
	}
}

// MainMenuHandler — кнопка «В главное меню», показывается только в корне каталога.
func (h *Handlers) MainMenuHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf(texts.StartMsg, update.CallbackQuery.From.FirstName),
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: h.kb.MainMenuKb,
	}); err != nil {
		h.log.Errorf("MainMenuHandler: failed to send message to %d: %v", chatID, err)
	}
}

// HelpHandler показывает экран помощи.
func (h *Handlers) HelpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        texts.HelpMsg,
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: h.kb.MainMenuKb,
	}); err != nil {
		h.log.Errorf("HelpHandler: failed to send message to %d: %v", chatID, err)
	}
}
