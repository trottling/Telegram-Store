package service

import (
	"context"

	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
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

// invalidateCategoryAncestorChain сбрасывает кэш списков детей от
// categoryID вверх до корня — видимость категории зависит от остатков во
// всём поддереве, так что смена остатка может затронуть всех предков.
// Best-effort: ошибка чтения пути просто оставляет кэш дожить до TTL.
func invalidateCategoryAncestorChain(ctx context.Context, categoryRepo repository.CategoryRepository, cache domaincache.CategoryCache, categoryID models.CategoryID) {
	path, err := categoryRepo.ListPath(ctx, categoryID)
	if err != nil {
		return
	}
	for _, c := range path {
		_ = cache.InvalidateCategoryChildren(ctx, c.ParentID)
	}
}

// recomputeCategoryStockChain пересчитывает HasStock от categoryID вверх до
// корня, по одному уровню за раз — снизу вверх, как того требует агрегат.
// Останавливается, как только значение на каком-то уровне не изменилось:
// агрегат предка зависит только от этого значения, а не от остальных
// параметров категории, так что выше по цепочке пересчитывать нечего.
// Best-effort, как и инвалидация кэша: ошибка оставляет прежнее значение до
// следующего события или до самолечащего пересчёта в cmd/migrate.
func recomputeCategoryStockChain(ctx context.Context, categoryRepo repository.CategoryRepository, log *zap.SugaredLogger, categoryID models.CategoryID) {
	for {
		changed, err := categoryRepo.RecomputeStock(ctx, categoryID)
		if err != nil {
			log.Warnw("service: category stock recompute failed", "error", err, "category_id", categoryID)
			return
		}
		if !changed {
			return
		}

		category, err := categoryRepo.GetByID(ctx, categoryID)
		if err != nil || category.ParentID == nil {
			return
		}
		categoryID = *category.ParentID
	}
}
