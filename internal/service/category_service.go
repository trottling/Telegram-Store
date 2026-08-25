package service

import (
	"context"
	"errors"

	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"go.uber.org/zap"
)

type CategorySrv struct {
	categoryRepo repository.CategoryRepository
	productRepo  repository.ProductRepository
	cache        domaincache.CategoryCache
	log          *zap.SugaredLogger
}

func NewCategorySrv(
	categoryRepo repository.CategoryRepository,
	productRepo repository.ProductRepository,
	cache domaincache.CategoryCache,
	log *zap.SugaredLogger,
) *CategorySrv {
	return &CategorySrv{categoryRepo: categoryRepo, productRepo: productRepo, cache: cache, log: log}
}

// ListChildren читает только Redis — видимость каталога считает и публикует
// туда фоновый воркер (см. RefreshCatalogSnapshot), никакого промаха на
// Postgres тут больше нет. Промах (холодный старт до первого тика воркера,
// или воркер ещё не дошёл до только что созданной категории) — это пустой
// список, не ошибка: показать пустой раздел лучше, чем молча ничего не
// ответить на тап.
func (s *CategorySrv) ListChildren(ctx context.Context, parentID *models.CategoryID) ([]models.Category, error) {
	children, err := s.cache.GetCategoryChildren(ctx, parentID)
	if errors.Is(err, domaincache.ErrMiss) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return children, nil
}

func (s *CategorySrv) GetByID(ctx context.Context, id models.CategoryID) (*models.Category, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *CategorySrv) ListPath(ctx context.Context, id models.CategoryID) ([]models.Category, error) {
	return s.categoryRepo.ListPath(ctx, id)
}

func (s *CategorySrv) ListProducts(ctx context.Context, categoryID *models.CategoryID) ([]models.Product, error) {
	return s.productRepo.ListActiveByCategory(ctx, categoryID)
}

// ListAllFlat намеренно мимо кэша — всегда читает из Postgres.
func (s *CategorySrv) ListAllFlat(ctx context.Context) ([]models.Category, error) {
	return s.categoryRepo.ListAllFlat(ctx)
}

// RefreshCatalogSnapshot пересчитывает видимость всего дерева категорий разом
// и публикует в кэш готовые списки детей на каждый узел — вызывается только
// фоновым воркером (см. cmd/bot's catalog worker), не на пути запроса. Раньше
// видимость каждой категории (Category.HasStock) поддерживалась построчно, на
// каждой операции, способной её изменить (покупка, CRUD товара/категории) —
// 5 разных мест, которые нужно было не забыть вызвать, и синхронный обход
// дерева вверх на каждой покупке. Здесь дерево строится в памяти одним
// проходом раз в интервал (Settings.CatalogRefreshIntervalSeconds) — раз это
// больше не путь запроса, дешевизна больше не важна, и городить SQL-агрегат
// незачем.
func (s *CategorySrv) RefreshCatalogSnapshot(ctx context.Context) error {
	all, err := s.categoryRepo.ListAllFlat(ctx)
	if err != nil {
		return err
	}
	stockedIDs, err := s.productRepo.ListStockedCategoryIDs(ctx)
	if err != nil {
		return err
	}

	byID := make(map[models.CategoryID]models.Category, len(all))
	for _, c := range all {
		byID[c.ID] = c
	}

	// Видимость снизу вверх: категория видима, если у неё самой есть остаток
	// (стартовые ID — из stockedIDs) или видим кто-то из её потомков.
	// Достаточно один раз пройти вверх по ParentID от каждой стокнутой
	// категории — уже отмеченного предка второй раз не трогаем.
	visible := make(map[models.CategoryID]bool, len(all))
	for _, id := range stockedIDs {
		for !visible[id] {
			visible[id] = true
			c, ok := byID[id]
			if !ok || c.ParentID == nil {
				break
			}
			id = *c.ParentID
		}
	}

	// childrenByParent группирует ВСЕ категории (видимые и нет) по родителю —
	// так у каждого узла, включая листья без единого потомка, будет явная
	// (пусть пустая) запись в кэше вместо постоянного промаха. all уже
	// отсортирован по (parent_id, name) на уровне SQL (ListAllFlat), поэтому
	// порядок внутри каждой группы сохраняется без пересортировки.
	var rootChildren []models.Category
	childrenByParent := make(map[models.CategoryID][]models.Category, len(all))
	for _, c := range all {
		if _, ok := childrenByParent[c.ID]; !ok {
			childrenByParent[c.ID] = nil // категория-лист: своих детей нет вовсе
		}
		if c.ParentID == nil {
			if visible[c.ID] {
				rootChildren = append(rootChildren, c)
			}
			continue
		}
		if _, ok := childrenByParent[*c.ParentID]; !ok {
			childrenByParent[*c.ParentID] = nil
		}
		if visible[c.ID] {
			childrenByParent[*c.ParentID] = append(childrenByParent[*c.ParentID], c)
		}
	}

	if setErr := s.cache.SetCategoryChildren(ctx, nil, rootChildren); setErr != nil {
		s.log.Warnw("category_service: failed to publish catalog snapshot", "error", setErr, "parent_id", "root")
	}
	for parentID, list := range childrenByParent {
		if setErr := s.cache.SetCategoryChildren(ctx, &parentID, list); setErr != nil {
			s.log.Warnw("category_service: failed to publish catalog snapshot", "error", setErr, "parent_id", parentID)
		}
	}
	return nil
}
