package redis

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()

	server := miniredis.RunT(t)
	log := logrus.New()
	log.SetOutput(io.Discard)

	return NewRedisCache(redis.NewClient(&redis.Options{Addr: server.Addr()}), log)
}

// TestConsumeFSMStateOnlyOnceUnderConcurrency — то, ради чего ConsumeFSMState
// вообще появился: go-telegram/bot обрабатывает каждый update своей горутиной,
// поэтому двойной тап по «Подтвердить» шёл двумя параллельными хендлерами.
// При чтении и очистке двумя вызовами оба видели состояние и оба списывали
// деньги; GETDEL отдаёт его ровно одному.
func TestConsumeFSMStateOnlyOnceUnderConcurrency(t *testing.T) {
	const goroutines = 50

	cache := newTestCache(t)
	ctx := context.Background()

	const telegramID = int64(777)
	want := &domainfsm.State{Step: domainfsm.StepAwaitingBuyConfirmation, ProductID: 5, Quantity: 3, MessageID: 11}
	if err := cache.SetFSMState(ctx, telegramID, want); err != nil {
		t.Fatalf("не удалось записать состояние: %v", err)
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		got      int
		notFound int
	)
	start := make(chan struct{})

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			st, err := cache.ConsumeFSMState(ctx, telegramID)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && st != nil:
				got++
			case errors.Is(err, domainfsm.ErrNotFound):
				notFound++
			default:
				t.Errorf("неожиданная ошибка: %v", err)
			}
		}()
	}

	close(start)
	wg.Wait()

	if got != 1 {
		t.Errorf("состояние получили %d горутин, ожидалась ровно 1", got)
	}
	if notFound != goroutines-1 {
		t.Errorf("ErrNotFound получили %d горутин, ожидалось %d", notFound, goroutines-1)
	}
}

// TestSetLoginCodeInvalidatesPrevious — у админа живым остаётся только
// последний код. Иначе каждый повторный /admin добавлял бы ещё один рабочий
// код и линейно повышал шансы подбора.
func TestSetLoginCodeInvalidatesPrevious(t *testing.T) {
	cache := newTestCache(t)
	ctx := context.Background()

	const telegramID = int64(500)
	if err := cache.SetLoginCode(ctx, "hash-old", telegramID, time.Minute); err != nil {
		t.Fatalf("не удалось записать первый код: %v", err)
	}
	if err := cache.SetLoginCode(ctx, "hash-new", telegramID, time.Minute); err != nil {
		t.Fatalf("не удалось записать второй код: %v", err)
	}

	if _, err := cache.ConsumeLoginCode(ctx, "hash-old"); !errors.Is(err, adminsession.ErrNotFound) {
		t.Errorf("старый код вернул %v, ожидался ErrNotFound — он должен быть погашен", err)
	}

	got, err := cache.ConsumeLoginCode(ctx, "hash-new")
	if err != nil {
		t.Fatalf("новый код не сработал: %v", err)
	}
	if got != telegramID {
		t.Errorf("новый код вернул telegramID %d, ожидался %d", got, telegramID)
	}
}

// TestSetLoginCodeIsPerAdmin — гашение затрагивает только своего админа.
func TestSetLoginCodeIsPerAdmin(t *testing.T) {
	cache := newTestCache(t)
	ctx := context.Background()

	if err := cache.SetLoginCode(ctx, "hash-a", 1, time.Minute); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if err := cache.SetLoginCode(ctx, "hash-b", 2, time.Minute); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if _, err := cache.ConsumeLoginCode(ctx, "hash-a"); err != nil {
		t.Errorf("код первого админа погашен выдачей кода второму: %v", err)
	}
}

// TestIncrExchangeAttemptsWindowDoesNotSlide — окно должно истекать. EXPIRE
// ставится только на первом INCR: если продлевать его на каждой попытке, то у
// того, кто уже упёрся в лимит, ключ будет жить вечно и админ окажется заперт
// навсегда вместо минуты.
func TestIncrExchangeAttemptsWindowDoesNotSlide(t *testing.T) {
	server := miniredis.RunT(t)
	log := logrus.New()
	log.SetOutput(io.Discard)
	cache := NewRedisCache(redis.NewClient(&redis.Options{Addr: server.Addr()}), log)

	ctx := context.Background()
	const window = time.Minute

	for i := int64(1); i <= 3; i++ {
		attempt, err := cache.IncrExchangeAttempts(ctx, "10.0.0.1", window)
		if err != nil {
			t.Fatalf("попытка %d вернула ошибку: %v", i, err)
		}
		if attempt != i {
			t.Errorf("номер попытки = %d, ожидался %d", attempt, i)
		}
		// Между попытками время идёт, но TTL не должен обновляться.
		server.FastForward(20 * time.Second)
	}

	// 60 секунд прошло — счётчик обязан начаться заново.
	server.FastForward(time.Second)
	attempt, err := cache.IncrExchangeAttempts(ctx, "10.0.0.1", window)
	if err != nil {
		t.Fatalf("попытка после истечения окна вернула ошибку: %v", err)
	}
	if attempt != 1 {
		t.Errorf("после истечения окна номер попытки = %d, ожидался 1 (окно не должно скользить)", attempt)
	}
}

// TestIncrExchangeAttemptsIsPerKey — лимит считается по каждому IP отдельно,
// иначе один атакующий закрывал бы вход всем админам.
func TestIncrExchangeAttemptsIsPerKey(t *testing.T) {
	cache := newTestCache(t)
	ctx := context.Background()

	for range 5 {
		if _, err := cache.IncrExchangeAttempts(ctx, "10.0.0.1", time.Minute); err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
	}

	attempt, err := cache.IncrExchangeAttempts(ctx, "10.0.0.2", time.Minute)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if attempt != 1 {
		t.Errorf("для другого IP номер попытки = %d, ожидался 1", attempt)
	}
}

// TestConsumeFSMStateReturnsStateAndDeletes — прочитанное состояние совпадает с
// записанным, а повторный вызов уже ничего не находит.
func TestConsumeFSMStateReturnsStateAndDeletes(t *testing.T) {
	cache := newTestCache(t)
	ctx := context.Background()

	const telegramID = int64(42)
	want := &domainfsm.State{Step: domainfsm.StepAwaitingRefillAmount, Merchant: "crystalpay"}
	if err := cache.SetFSMState(ctx, telegramID, want); err != nil {
		t.Fatalf("не удалось записать состояние: %v", err)
	}

	got, err := cache.ConsumeFSMState(ctx, telegramID)
	if err != nil {
		t.Fatalf("ConsumeFSMState вернул ошибку: %v", err)
	}
	if got.Step != want.Step || got.Merchant != want.Merchant {
		t.Errorf("получено %+v, ожидалось %+v", got, want)
	}

	if _, err = cache.ConsumeFSMState(ctx, telegramID); !errors.Is(err, domainfsm.ErrNotFound) {
		t.Errorf("повторный вызов вернул %v, ожидался ErrNotFound", err)
	}
}
