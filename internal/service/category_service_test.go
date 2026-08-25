package service

import (
	"context"
	"testing"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
)

// fakeCategoryRepoTree — только ListAllFlat отдаёт реальные данные,
// остальное не используется RefreshCatalogSnapshot и паникует, если позвано.
type fakeCategoryRepoTree struct {
	all []models.Category
}

func (r *fakeCategoryRepoTree) Create(context.Context, *models.Category) error {
	panic("не используется")
}
func (r *fakeCategoryRepoTree) GetByID(context.Context, models.CategoryID) (*models.Category, error) {
	panic("не используется")
}
func (r *fakeCategoryRepoTree) Update(context.Context, *models.Category) error {
	panic("не используется")
}
func (r *fakeCategoryRepoTree) Delete(context.Context, models.CategoryID) error {
	panic("не используется")
}
func (r *fakeCategoryRepoTree) ListPath(context.Context, models.CategoryID) ([]models.Category, error) {
	panic("не используется")
}
func (r *fakeCategoryRepoTree) ListAllFlat(context.Context) ([]models.Category, error) {
	return r.all, nil
}
func (r *fakeCategoryRepoTree) CountChildren(context.Context, models.CategoryID) (int64, error) {
	panic("не используется")
}

type fakeProductRepoStocked struct {
	stockedCategoryIDs []models.CategoryID
}

func (r *fakeProductRepoStocked) Create(context.Context, *models.Product) error {
	panic("не используется")
}
func (r *fakeProductRepoStocked) GetByID(context.Context, models.ProductID) (*models.Product, error) {
	panic("не используется")
}
func (r *fakeProductRepoStocked) Update(context.Context, *models.Product) error {
	panic("не используется")
}
func (r *fakeProductRepoStocked) Delete(context.Context, models.ProductID) error {
	panic("не используется")
}
func (r *fakeProductRepoStocked) ListActive(context.Context) ([]models.Product, error) {
	panic("не используется")
}
func (r *fakeProductRepoStocked) ListActiveByCategory(context.Context, *models.CategoryID) ([]models.Product, error) {
	panic("не используется")
}
func (r *fakeProductRepoStocked) ListStockedCategoryIDs(context.Context) ([]models.CategoryID, error) {
	return r.stockedCategoryIDs, nil
}
func (r *fakeProductRepoStocked) AddItems(context.Context, models.ProductID, []string) error {
	panic("не используется")
}
func (r *fakeProductRepoStocked) ReserveItems(context.Context, models.ProductID, int) ([]models.ProductItem, error) {
	panic("не используется")
}
func (r *fakeProductRepoStocked) CountAvailableItems(context.Context, models.ProductID) (int, error) {
	panic("не используется")
}
func (r *fakeProductRepoStocked) ListAll(context.Context, int, int, *models.CategoryID) ([]models.ProductAdminSummary, error) {
	panic("не используется")
}
func (r *fakeProductRepoStocked) CountAll(context.Context, *models.CategoryID) (int64, error) {
	panic("не используется")
}
func (r *fakeProductRepoStocked) CountByCategoryID(context.Context, models.CategoryID) (int64, error) {
	panic("не используется")
}

// fakeCategoryCache записывает каждый SetCategoryChildren — тест проверяет,
// что запись случилась для каждого узла дерева, включая невидимые и листья,
// а не только там, где нашлось что показать.
type fakeCategoryCache struct {
	written map[string][]models.Category // "root" или CategoryID.String()
}

func newFakeCategoryCache() *fakeCategoryCache {
	return &fakeCategoryCache{written: map[string][]models.Category{}}
}

func (c *fakeCategoryCache) GetCategoryChildren(context.Context, *models.CategoryID) ([]models.Category, error) {
	panic("не используется")
}
func (c *fakeCategoryCache) SetCategoryChildren(_ context.Context, parentID *models.CategoryID, categories []models.Category) error {
	key := "root"
	if parentID != nil {
		key = parentID.String()
	}
	c.written[key] = categories
	return nil
}

// namesOf — сравнивать список категорий по именам удобнее, чем по UUID.
func namesOf(categories []models.Category) []string {
	names := make([]string, len(categories))
	for i, c := range categories {
		names[i] = c.Name
	}
	return names
}

func sameNames(t *testing.T, got []models.Category, want []string) {
	t.Helper()
	gotNames := namesOf(got)
	if len(gotNames) != len(want) {
		t.Fatalf("got %v, want %v", gotNames, want)
	}
	for i := range want {
		if gotNames[i] != want[i] {
			t.Fatalf("got %v, want %v", gotNames, want)
		}
	}
}

// TestRefreshCatalogSnapshot_VisibilityPropagatesUpward — регрессия на само
// назначение снепшота: категория без собственного остатка обязана остаться
// видимой, если остаток есть у любого потомка вниз по дереву, и обязана
// пропасть из списка, если ни у неё, ни у потомков остатка нет — включая
// случай в несколько уровней глубиной, где раньше это делал SQL-агрегат
// (Category.HasStock), а теперь тот же обход идёт в памяти.
func TestRefreshCatalogSnapshot_VisibilityPropagatesUpward(t *testing.T) {
	// Дерево:
	//   A (без остатка)
	//     └ B (без остатка)
	//         └ C (остаток есть, лист)
	//   D (без остатка, лист, корень)
	a := models.Category{ID: models.NewCategoryID(), Name: "A"}
	b := models.Category{ID: models.NewCategoryID(), ParentID: &a.ID, Name: "B"}
	c := models.Category{ID: models.NewCategoryID(), ParentID: &b.ID, Name: "C"}
	d := models.Category{ID: models.NewCategoryID(), Name: "D"}

	categoryRepo := &fakeCategoryRepoTree{all: []models.Category{a, b, c, d}}
	productRepo := &fakeProductRepoStocked{stockedCategoryIDs: []models.CategoryID{c.ID}}
	cache := newFakeCategoryCache()
	srv := NewCategorySrv(categoryRepo, productRepo, cache, zap.NewNop().Sugar())

	if err := srv.RefreshCatalogSnapshot(context.Background()); err != nil {
		t.Fatalf("RefreshCatalogSnapshot вернул ошибку: %v", err)
	}

	// D не должен просочиться в корень — ни у него, ни у его несуществующих
	// потомков остатка нет.
	sameNames(t, cache.written["root"], []string{"A"})
	sameNames(t, cache.written[a.ID.String()], []string{"B"})
	sameNames(t, cache.written[b.ID.String()], []string{"C"})

	// Листья — явная пустая запись, а не отсутствие ключа: иначе
	// CategorySrv.ListChildren видел бы промах кэша на каждый заход в C или D.
	if got, ok := cache.written[c.ID.String()]; !ok || len(got) != 0 {
		t.Errorf("C: ожидалась явная пустая запись, получено %v (есть ключ: %v)", got, ok)
	}
	if got, ok := cache.written[d.ID.String()]; !ok || len(got) != 0 {
		t.Errorf("D: ожидалась явная пустая запись, получено %v (есть ключ: %v)", got, ok)
	}
}

// TestRefreshCatalogSnapshot_SiblingWithoutStockStaysHidden — соседняя ветка
// без остатка не должна попасть в список вместе с видимой веткой того же
// родителя.
func TestRefreshCatalogSnapshot_SiblingWithoutStockStaysHidden(t *testing.T) {
	parent := models.Category{ID: models.NewCategoryID(), Name: "Parent"}
	stocked := models.Category{ID: models.NewCategoryID(), ParentID: &parent.ID, Name: "Stocked"}
	empty := models.Category{ID: models.NewCategoryID(), ParentID: &parent.ID, Name: "Empty"}

	categoryRepo := &fakeCategoryRepoTree{all: []models.Category{parent, stocked, empty}}
	productRepo := &fakeProductRepoStocked{stockedCategoryIDs: []models.CategoryID{stocked.ID}}
	cache := newFakeCategoryCache()
	srv := NewCategorySrv(categoryRepo, productRepo, cache, zap.NewNop().Sugar())

	if err := srv.RefreshCatalogSnapshot(context.Background()); err != nil {
		t.Fatalf("RefreshCatalogSnapshot вернул ошибку: %v", err)
	}

	sameNames(t, cache.written[parent.ID.String()], []string{"Stocked"})
}
