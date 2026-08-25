package service

import (
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"go.uber.org/zap"
)

// MultiCache — композиция кэшей для AdminSrv и PurchaseSrv, которым нужно
// инвалидировать сразу несколько сущностей за одно действие. Живёт здесь,
// а не в internal/domain/cache — это удобство сервисного слоя, не доменный
// порт. Экспортирован (не multiCache), чтобы cmd/*/main.go мог сослаться на
// тип при связке через fx.Annotate(..., fx.As(new(service.MultiCache))).
type MultiCache interface {
	domaincache.UserCache
	domaincache.ProductCache
	domaincache.CategoryCache
}

// logInvalidation — инвалидация кэша везде best-effort, ронять из-за неё уже
// выполненную запись в БД нельзя. Но там, где расхождение стоит денег или прав
// (баланс, роль, настройки мерчантов), сбой обязан быть виден: иначе Redis
// молча отдаёт устаревшее значение до истечения TTL, и жалоба вроде «бан не
// применился» выглядит необъяснимой. Каталог сюда не заводим — он устаревает
// на минуту, и цена этого мала.
func logInvalidation(log *zap.SugaredLogger, err error, entity string, id any) {
	if err == nil {
		return
	}
	log.Warnw("service: cache invalidation failed, stale value until TTL", "error", err, "entity", entity, "id", id)
}
