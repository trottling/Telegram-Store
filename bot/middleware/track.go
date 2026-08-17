package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Track делает возможным корректное завершение работы.
//
// go-telegram/bot запускает обработку каждого update отдельной горутиной и в
// свой WaitGroup их не берёт: Bot.Start дожидается только цикла поллинга.
// Причём ctx у хендлеров тот же, что у поллинга, поэтому его отмена при
// SIGTERM рвала обработку на середине — покупка откатывалась, а сброс кэша
// баланса после уже закоммиченной покупки не успевал.
//
// Поэтому здесь два действия: считаем update'ы в обработке (WaitInFlight ждёт
// их при остановке) и отвязываем их ctx от отмены поллинга — так же, как
// payments_backend отвязывает вебхуки от разрыва соединения мерчантом.
// Ограничитель — таймаут дренажа в cmd/bot/lifecycle.go.
func (m *Middlewares) Track(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		m.inFlight.Add(1)
		defer m.inFlight.Done()

		next(context.WithoutCancel(ctx), b, update)
	}
}

// WaitInFlight ждёт, пока догребут уже начатые update'ы, но не дольше ctx.
func (m *Middlewares) WaitInFlight(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		m.inFlight.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
