package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/TG-Store/bot/texts"
	domainfsm "github.com/trottling/TG-Store/internal/domain/fsm"
)

// RefillBalanceHandler баланс не трогает — только запрашивает сумму,
// счёт создаётся в middleware.FSM после ответа пользователя.
func (h *Handlers) RefillBalanceHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	st := &domainfsm.State{Step: domainfsm.StepAwaitingRefillAmount}
	if err := h.stateStore.SetFSMState(ctx, chatID, st); err != nil {
		h.log.Errorf("RefillBalanceHandler: failed to set refill-amount state for %d: %v", chatID, err)
		return
	}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   texts.AskRefillAmountMsg,
	}); err != nil {
		h.log.Errorf("RefillBalanceHandler: failed to send message to %d: %v", chatID, err)
	}
}
