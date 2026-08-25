package models

import "strconv"

// TelegramID — отдельный тип поверх int64 для Telegram-идентификаторов
// пользователей (User.TelegramID и всё, что на него ссылается), чтобы
// компилятор не путал его с ID других сущностей в сигнатурах вида
// Foo(ctx, id1, id2). GORM обрабатывает именованный int64-тип нативно, без
// Scanner/Valuer; JSON маршалится как обычное число — тот же формат, что и
// раньше у голого int64.
type TelegramID int64

func (id TelegramID) String() string {
	return strconv.FormatInt(int64(id), 10)
}
