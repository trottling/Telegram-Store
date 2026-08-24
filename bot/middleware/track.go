package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// maxConcurrentUpdates — потолок одновременно обрабатываемых update'ов.
// go-telegram/bot запускает каждый update в своей горутине без пула вообще
// (ProcessUpdate: `go r(...)`) — без ограничителя всплеск активности рождает
// неограниченное число горутин, одновременно бьющих в Postgres/Redis. Число
// подобрано под целевой деплой (1 CPU/2 GB VPS) в паре с пулом соединений к
// Postgres — см. internal/repository/postgres.dbMaxOpenConns.
const maxConcurrentUpdates = 15

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

		// Ждём свободный слот семафора, но не дольше отмены ctx — на
		// остановке update'ы, ещё не начавшие реально обрабатываться, не
		// зависают, а просто отваливаются (WaitInFlight всё равно ограничен
		// drainTimeout в cmd/bot/lifecycle.go). nil sem (Middlewares{} без
		// New(), как в track_test.go) — лимит просто выключен, а не паника
		// или дедлок: отправка в nil-канал блокировалась бы навсегда.
		if m.sem != nil {
			select {
			case m.sem <- struct{}{}:
				defer func() { <-m.sem }()
			case <-ctx.Done():
				return
			}
		}

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
