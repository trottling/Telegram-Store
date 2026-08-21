// Package adminsession хранит одноразовые коды входа и сессии админ-панели.
// Всё в Redis, ничего в Postgres — отдельный от domain/cache и domain/fsm контекст.
package adminsession

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound — код/сессия не найдены, истекли или уже использованы.
var ErrNotFound = errors.New("adminsession: not found")

// Store реализуется internal/cache/redis.Cache.
type Store interface {
	SetSession(ctx context.Context, sessionHash string, telegramID int64, ttl time.Duration) error
	GetSession(ctx context.Context, sessionHash string) (telegramID int64, err error)
	DeleteSession(ctx context.Context, sessionHash string) error

	// IncrExchangeAttempts считает попытки обмена кода в окне window и
	// возвращает номер текущей. Код всего 6-значный, поэтому без счётчика его
	// можно подбирать сколько угодно: неудачная попытка ничего не расходует.
	// key — IP клиента.
	IncrExchangeAttempts(ctx context.Context, key string, window time.Duration) (attempt int64, err error)
}
