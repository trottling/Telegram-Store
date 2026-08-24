package cache

import "context"

// ReplenishmentCheckCooldown — короткий TTL-маркер «этот счёт уже проверяли
// недавно», не read-through кэш сущности. Защищает ReplenishmentService.
// CheckInvoice от спама по кнопке «Проверить оплату» в боте: без него частые
// тапы бьют напрямую в CheckStatus мерчанта и рискуют упереться в его же
// rate limit.
type ReplenishmentCheckCooldown interface {
	// TryAcquire — true, если кулдауна ещё не было и он выставлен прямо
	// сейчас; false — счёт проверяли недавно, поход к мерчанту нужно
	// придержать.
	TryAcquire(ctx context.Context, replenishmentID int64) (bool, error)
}
