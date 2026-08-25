package redis

import "time"

const (
	userTTL           = 10 * time.Minute
	activeProductsTTL = time.Minute
	productTTL        = time.Minute
	productCountTTL   = 30 * time.Second
	// categoryChildrenTTL — не про свежесть (значение всегда пишет фоновый
	// воркер, см. CategorySrv.RefreshCatalogSnapshot, а не читатели по
	// промаху), а предохранитель на случай, если воркер вообще перестал
	// бегать: без TTL мёртвый воркер оставил бы каталог с последним верным
	// снепшотом навсегда, вместо явно опустевшего списка. С запасом даже
	// над максимально разрешённым интервалом (3600с, см. AdminSrv.UpdateSettings).
	categoryChildrenTTL = time.Hour
	stateTTL            = 5 * time.Minute
	// settingsTTL держим коротким не ради свежести данных, а как предохранитель:
	// в Settings лежат креды платёжных мерчантов и их флаги Enabled, а
	// инвалидация после правки в панели — best-effort. Если она не сработает,
	// это окно, в течение которого бот продолжит ходить к отключённому мерчанту
	// или со старыми кредами.
	settingsTTL = 5 * time.Minute
	// replenishmentCheckCooldownTTL — минимальный интервал между походами к
	// CheckStatus мерчанта по одному счёту (кнопка «Проверить оплату» в
	// боте). Не про точность, а про то, чтобы частые тапы не превращались в
	// частые запросы к API мерчанта.
	replenishmentCheckCooldownTTL = 15 * time.Second
)
