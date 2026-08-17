// Package fsm — состояние многошагового диалога с ботом (покупка, пополнение).
// В отличие от domain/cache — источник истины сам Redis, без Postgres за ним.
package fsm

import (
	"context"
	"errors"
)

// ErrNotFound — для чата нет ожидаемого шага (обычный случай, не ошибка).
var ErrNotFound = errors.New("fsm: not found")

type step string

const (
	StepAwaitingBuyQuantity     step = "awaiting_buy_quantity"
	StepAwaitingBuyConfirmation step = "awaiting_buy_confirmation"
	StepAwaitingRefillAmount    step = "awaiting_refill_amount"
)

// State — ожидаемый шаг для одного чата. MessageID — сообщение сценария,
// чтобы текстовый ответ (без callback_query) редактировал то же сообщение.
type State struct {
	Step      step   `json:"step"`
	ProductID int64  `json:"product_id,omitempty"`
	Quantity  int    `json:"quantity,omitempty"`
	MessageID int    `json:"message_id,omitempty"`
	Merchant  string `json:"merchant,omitempty"` // выбранный мерчант для StepAwaitingRefillAmount
}

// Store — хранилище FSM. Методы называются с суффиксом FSMState, а не
// просто Get/Set/Clear, поскольку та же Redis-структура реализует и domain/cache.
type Store interface {
	GetFSMState(ctx context.Context, telegramID int64) (*State, error)
	SetFSMState(ctx context.Context, telegramID int64, state *State) error
	ClearFSMState(ctx context.Context, telegramID int64) error

	// ConsumeFSMState атомарно читает и удаляет состояние (Redis GETDEL,
	// ср. adminsession.ConsumeLoginCode). Нужен там, где состояние работает
	// как одноразовый токен: go-telegram/bot обрабатывает каждый update своей
	// горутиной, поэтому Get+Clear двумя вызовами позволяет двум быстрым
	// тапам по одной кнопке пройти проверку дважды. Шаг вызывающий проверяет
	// уже после — состояние к этому моменту снято в любом случае.
	ConsumeFSMState(ctx context.Context, telegramID int64) (*State, error)
}
