// Package adminsession хранит сессии админ-панели и счётчик попыток обмена
// initData. Всё в Redis, ничего в Postgres — отдельный от domain/cache и
// domain/fsm контекст.
package adminsession

import (
	"context"
	"errors"
	"time"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ErrNotFound — сессия не найдена, истекла или уже отозвана.
var ErrNotFound = errors.New("adminsession: not found")

// Store реализуется internal/cache/redis.Cache.
type Store interface {
	SetSession(ctx context.Context, sessionHash string, telegramID models.TelegramID, ttl time.Duration) error
	GetSession(ctx context.Context, sessionHash string) (telegramID models.TelegramID, err error)
	DeleteSession(ctx context.Context, sessionHash string) error

	// IncrExchangeAttempts считает попытки обмена initData в окне window и
	// возвращает номер текущей. initData подписывает Telegram, подобрать её
	// нельзя — это не защита от перебора, а обычная защита от злоупотребления
	// самим эндпоинтом. key — IP клиента.
	IncrExchangeAttempts(ctx context.Context, key string, window time.Duration) (attempt int64, err error)
}
