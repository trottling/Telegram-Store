package handlers

import (
	"context"
	"errors"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
)

// AdminHandler отвечает на /admin кнопками панели и статистики — обе открываются
// как Telegram Mini App (web_app), логин по initData (см. AdminAuthSrv.ExchangeInitData),
// никакого кода тут больше не выдаётся.
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

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ReplyMarkup: h.kb.AdminKb(user.Language),
		ChatID:      chatID,
		Text:        texts.T(user.Language, texts.AdminMsg, nil),
	})
	if err == nil {
		return
	}

	// web_app-кнопки требуют https (тот же запрет, что и у обычных URL-кнопок) —
	// в дебаге, где ADMIN_PANEL_FRONTEND_URL смотрит на localhost, отправка
	// падает. Текстовой ссылкой это не обойти, как раньше с кодом: initData
	// доступна только внутри самого запуска Mini App, а не по прямой ссылке в
	// обычном браузере — так что просто сообщаем, что панель недоступна.
	if !errors.Is(err, bot.ErrorBadRequest) {
		h.log.Errorf("AdminHandler: failed to send message to %d: %v", chatID, err)
		return
	}

	h.log.Warnf("AdminHandler: web_app buttons rejected for %d, ADMIN_PANEL_FRONTEND_URL is likely not https: %v", chatID, err)
	if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   texts.T(user.Language, texts.AdminPanelUnavailableMsg, nil),
	}); err != nil {
		h.log.Errorf("AdminHandler: failed to send message to %d: %v", chatID, err)
	}
}
