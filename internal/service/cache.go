package service

import (
	"context"

	"github.com/sirupsen/logrus"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
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
func logInvalidation(log *logrus.Logger, err error, entity string, id any) {
	if err == nil {
		return
	}
	log.WithError(err).WithFields(logrus.Fields{"entity": entity, "id": id}).
		Warn("service: cache invalidation failed, stale value until TTL")
}

// invalidateCategoryAncestorChain сбрасывает кэш списков детей от
// categoryID вверх до корня — видимость категории зависит от остатков во
// всём поддереве, так что смена остатка может затронуть всех предков.
// Best-effort: ошибка чтения пути просто оставляет кэш дожить до TTL.
func invalidateCategoryAncestorChain(ctx context.Context, categoryRepo repository.CategoryRepository, cache domaincache.CategoryCache, categoryID int64) {
	path, err := categoryRepo.ListPath(ctx, categoryID)
	if err != nil {
		return
	}
	for _, c := range path {
		_ = cache.InvalidateCategoryChildren(ctx, c.ParentID)
	}
}
