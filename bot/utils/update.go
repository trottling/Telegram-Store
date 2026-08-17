package utils

import "github.com/go-telegram/bot/models"

// CallbackQuery.Message — не указатель, а MaybeInaccessibleMessage: у него
// заполнено либо Message, либо InaccessibleMessage. Второй вариант Telegram
// присылает, когда сообщение с кнопкой боту уже недоступно (удалено или
// слишком старое) — тогда Message равен nil, и прямое разыменование роняет
// процесс, потому что recover() в go-telegram/bot нет.

// CallbackChatID — chat ID из callback_query. У недоступного сообщения chat
// всё же есть, поэтому ответить новым сообщением можно.
func CallbackChatID(update *models.Update) (int64, bool) {
	if update.CallbackQuery == nil {
		return 0, false
	}
	if msg := update.CallbackQuery.Message.Message; msg != nil {
		return msg.Chat.ID, true
	}
	if msg := update.CallbackQuery.Message.InaccessibleMessage; msg != nil {
		return msg.Chat.ID, true
	}
	return 0, false
}

// CallbackTarget — chat ID вместе с сообщением, которое хендлер собирается
// редактировать. Недоступное сообщение сюда не годится (править его нельзя),
// поэтому ok=false и хендлер просто выходит.
func CallbackTarget(update *models.Update) (chatID int64, messageID int, ok bool) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return 0, 0, false
	}
	msg := update.CallbackQuery.Message.Message
	return msg.Chat.ID, msg.ID, true
}
