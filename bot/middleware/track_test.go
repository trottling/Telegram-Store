package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Track и WaitInFlight трогают только счётчик, поэтому сервисы для теста не
// нужны — Middlewares конструируется напрямую.

// TestTrackDetachesContext — главное свойство: отмена ctx поллинга (её делает
// shutdown, чтобы остановить long-polling) не должна долетать до обработчика,
// иначе начатый update рвётся на середине.
func TestTrackDetachesContext(t *testing.T) {
	m := &Middlewares{}

	ctx, cancel := context.WithCancel(context.Background())

	var handlerErr error
	handler := m.Track(func(ctx context.Context, _ *bot.Bot, _ *models.Update) {
		cancel() // как будто в этот момент пришёл SIGTERM
		handlerErr = ctx.Err()
	})

	handler(ctx, nil, &models.Update{ID: 1})

	if handlerErr != nil {
		t.Errorf("ctx обработчика отменён (%v), а должен быть отвязан от отмены поллинга", handlerErr)
	}
}

// TestWaitInFlightWaitsForHandler — WaitInFlight не возвращается, пока
// обработчик не догреб: именно на это опирается порядок остановки в
// cmd/bot/lifecycle.go, где после ожидания закрывается Redis.
func TestWaitInFlightWaitsForHandler(t *testing.T) {
	m := &Middlewares{}

	release := make(chan struct{})
	started := make(chan struct{})
	handler := m.Track(func(context.Context, *bot.Bot, *models.Update) {
		close(started)
		<-release
	})

	go handler(context.Background(), nil, &models.Update{ID: 1})
	<-started

	// Пока обработчик держит release, ожидание должно упираться в таймаут.
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer shortCancel()
	if err := m.WaitInFlight(shortCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("WaitInFlight вернул %v, ожидался DeadlineExceeded — обработчик ещё в работе", err)
	}

	close(release)

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()
	if err := m.WaitInFlight(waitCtx); err != nil {
		t.Errorf("WaitInFlight вернул %v после завершения обработчика, ожидался nil", err)
	}
}

// TestWaitInFlightReturnsImmediatelyWhenIdle — на простое ожидание не тратим
// таймаут дренажа.
func TestWaitInFlightReturnsImmediatelyWhenIdle(t *testing.T) {
	m := &Middlewares{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := time.Now()
	if err := m.WaitInFlight(ctx); err != nil {
		t.Fatalf("WaitInFlight вернул %v, ожидался nil", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("ожидание заняло %v при отсутствии обработчиков", elapsed)
	}
}

// TestTrackReleasesCounterOnPanic — счётчик обязан освобождаться и при панике,
// иначе одна паника заперла бы shutdown на весь таймаут дренажа. Панику в
// реальной цепочке ловит Recover, стоящий сразу под Track.
func TestTrackReleasesCounterOnPanic(t *testing.T) {
	m := &Middlewares{}

	handler := m.Track(func(context.Context, *bot.Bot, *models.Update) {
		panic("сбой в обработчике")
	})

	func() {
		defer func() { _ = recover() }()
		handler(context.Background(), nil, &models.Update{ID: 1})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.WaitInFlight(ctx); err != nil {
		t.Errorf("WaitInFlight вернул %v — счётчик не освободился после паники", err)
	}
}
