package utils

import (
	"encoding/json"
	"testing"

	"github.com/go-telegram/bot/models"
)

// callbackUpdate собирает Update из сырого JSON, а не из литерала структуры:
// MaybeInaccessibleMessage выбирает Message/InaccessibleMessage в своём
// UnmarshalJSON по полю date, и именно это поведение тут и проверяется.
func callbackUpdate(t *testing.T, messageJSON string) *models.Update {
	t.Helper()

	var update models.Update
	if err := json.Unmarshal([]byte(`{"update_id":1,"callback_query":{"id":"q","data":"x","message":`+messageJSON+`}}`), &update); err != nil {
		t.Fatalf("не удалось разобрать update: %v", err)
	}
	return &update
}

const (
	accessibleMessage   = `{"message_id":10,"date":1700000000,"chat":{"id":42,"type":"private"}}`
	inaccessibleMessage = `{"message_id":10,"date":0,"chat":{"id":42,"type":"private"}}`
)

func TestCallbackChatID(t *testing.T) {
	tests := []struct {
		name       string
		update     *models.Update
		wantChatID int64
		wantOK     bool
	}{
		{"обычное сообщение", callbackUpdate(t, accessibleMessage), 42, true},
		{"недоступное сообщение — chat всё равно есть", callbackUpdate(t, inaccessibleMessage), 42, true},
		{"не callback_query", &models.Update{ID: 1}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatID, ok := CallbackChatID(tt.update)
			if chatID != tt.wantChatID || ok != tt.wantOK {
				t.Errorf("CallbackChatID() = (%d, %v), ожидалось (%d, %v)", chatID, ok, tt.wantChatID, tt.wantOK)
			}
		})
	}
}

func TestCallbackTarget(t *testing.T) {
	tests := []struct {
		name          string
		update        *models.Update
		wantChatID    int64
		wantMessageID int
		wantOK        bool
	}{
		{"обычное сообщение", callbackUpdate(t, accessibleMessage), 42, 10, true},
		// Недоступное сообщение редактировать нельзя, поэтому ok=false.
		{"недоступное сообщение", callbackUpdate(t, inaccessibleMessage), 0, 0, false},
		{"не callback_query", &models.Update{ID: 1}, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatID, messageID, ok := CallbackTarget(tt.update)
			if chatID != tt.wantChatID || messageID != tt.wantMessageID || ok != tt.wantOK {
				t.Errorf("CallbackTarget() = (%d, %d, %v), ожидалось (%d, %d, %v)",
					chatID, messageID, ok, tt.wantChatID, tt.wantMessageID, tt.wantOK)
			}
		})
	}
}

// TestInaccessibleMessageIsNil фиксирует предпосылку всего фикса: у
// недоступного сообщения Message равен nil, и прямое разыменование
// .Message.Message.Chat.ID паникует.
func TestInaccessibleMessageIsNil(t *testing.T) {
	update := callbackUpdate(t, inaccessibleMessage)

	if update.CallbackQuery.Message.Message != nil {
		t.Fatal("ожидался nil в Message у недоступного сообщения — предпосылка фикса изменилась")
	}
	if update.CallbackQuery.Message.InaccessibleMessage == nil {
		t.Fatal("ожидался заполненный InaccessibleMessage")
	}
}
