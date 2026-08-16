package service

import (
	"context"

	domaincache "github.com/trottling/TG-Store/internal/domain/cache"
	"github.com/trottling/TG-Store/internal/domain/repository"
)

// multiCache — композиция кэшей для AdminSrv и PurchaseSrv, которым нужно
// инвалидировать сразу несколько сущностей за одно действие. Живёт здесь,
// а не в internal/domain/cache — это удобство сервисного слоя, не доменный порт.
type multiCache interface {
	domaincache.UserCache
	domaincache.ProductCache
	domaincache.CategoryCache
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
