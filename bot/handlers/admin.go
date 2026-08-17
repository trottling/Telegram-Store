package handlers

import (
	"context"
	"errors"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
)

// AdminHandler отвечает на /admin ссылкой на панель и одноразовым кодом входа (30 секунд).
func (h *Handlers) AdminHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	user, err := h.userService.GetProfile(ctx, chatID)
	if err != nil {
		h.log.Errorf("AdminHandler: failed to get profile for %d: %v", chatID, err)
		return
	}

	if !user.IsAdmin() {
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   texts.T(user.Language, texts.NotAdminMsg, nil),
		}); err != nil {
			h.log.Errorf("AdminHandler: failed to send message to %d: %v", chatID, err)
		}
		return
	}

	code, err := h.adminAuthService.IssueLoginCode(ctx, chatID)
	if err != nil {
		h.log.Errorf("AdminHandler: failed to issue login code for %d: %v", chatID, err)
		if _, err = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: texts.T(user.Language, texts.AdminCodeErrMsg, nil)}); err != nil {
			h.log.Errorf("AdminHandler: failed to send message to %d: %v", chatID, err)
		}
		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ReplyMarkup: h.kb.AdminKb(user.Language),
		ChatID:      chatID,
		Text:        texts.T(user.Language, texts.AdminMsg, map[string]any{"Code": utils.EscapeMarkdownCode(code)}),
		ParseMode:   models.ParseModeMarkdown,
	})

	if err == nil {
		return
	}

	// Фолбэк: если кнопку отправить нельзя, шлём ссылку текстом.
	if !errors.Is(err, bot.ErrorBadRequest) {
		h.log.Errorf("AdminHandler: failed to send message to %d: %v", chatID, err)
		return
	}

	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: texts.T(user.Language, texts.AdminMsgWithLink, map[string]any{
			"Code": utils.EscapeMarkdownCode(code),
			"URL":  utils.EscapeMarkdown(h.adminPanelConfig.FrontendURL),
		}),
		ParseMode: models.ParseModeMarkdown,
	}); err != nil {
		h.log.Errorf("AdminHandler: failed to send message to %d: %v", chatID, err)
		return
	}
}
