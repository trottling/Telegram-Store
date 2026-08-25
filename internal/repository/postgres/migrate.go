package postgres

import (
	"context"
	"fmt"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// partialIndexes — то, что не выражается тегами GORM.
//
// Непроданных единиц у товара всегда мало, а проданные копятся и не удаляются.
// Обычный индекс по product_id со временем заставляет пробегать все проданные
// строки, чтобы отфильтровать их по is_sold; частичный содержит только
// непроданные и остаётся маленьким независимо от оборота. Задействован в самых
// горячих местах: ReserveItem (денежный путь под локами), проверка наличия в
// листинге товаров, рекурсивный CTE видимости категорий, счётчик на карточке.
//
// IF NOT EXISTS — cmd/migrate прогоняется на каждый деплой.
var partialIndexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_product_items_unsold
	 ON product_items (product_id) WHERE is_sold = false`,
}

func createPartialIndexes(db *gorm.DB) error {
	for _, stmt := range partialIndexes {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

// recomputeAllCategoryStock пересчитывает Category.HasStock для всех
// категорий разом — тот же агрегат, что RecomputeStock держит по одной
// категории на запись, но сразу по всему дереву снизу вверх одним запросом.
// Нужен как бэкфилл при первом появлении колонки; безопасно гонять на каждый
// деплой (таблица категорий маленькая) — заодно самолечит любое случайное
// расхождение, если какой-то путь в коде забыл вызвать RecomputeStock.
func recomputeAllCategoryStock(db *gorm.DB) error {
	const query = `
		WITH RECURSIVE subtree AS (
			SELECT id, id AS branch_id FROM categories
			UNION ALL
			SELECT c.id, s.branch_id FROM categories c JOIN subtree s ON c.parent_id = s.id
		),
		stocked AS (
			SELECT DISTINCT s.branch_id
			FROM subtree s
			JOIN products p ON p.category_id = s.id
			WHERE p.is_active = true
			  AND EXISTS (SELECT 1 FROM product_items pi WHERE pi.product_id = p.id AND pi.is_sold = false)
		)
		UPDATE categories SET has_stock = (id IN (SELECT branch_id FROM stocked))`
	return db.Exec(query).Error
}

// refreshCollationVersions — постоянный шум в логах, а не признак реальной
// проблемы: postgres:*-alpine собран на musl, а он вообще не версионирует
// collation. Postgres видит расхождение с версией, записанной при создании
// базы, и предупреждает на каждое новое подключение ("no actual collation
// version, but a version was recorded") — штатный ALTER DATABASE ... REFRESH
// COLLATION VERSION тут даже падает ("invalid collation version change"),
// сравнивать нечего. Обнулить datcollversion — единственный способ снять
// предупреждение; поведение сортировки строк это не меняет ни на бит.
//
// Ошибку не считаем фатальной для всей миграции и просто предупреждаем:
// на managed Postgres (не наш bundled образ) роль из POSTGRES_USER может не
// быть суперпользователем — обновлять системный каталог pg_database тогда
// нечем, но это не должно останавливать реальную схему.
func refreshCollationVersions(db *gorm.DB, log *zap.SugaredLogger) {
	if err := db.Exec(`UPDATE pg_database SET datcollversion = NULL WHERE datcollversion IS NOT NULL`).Error; err != nil {
		log.Warnw("postgres: failed to clear stale collation versions, the warning will keep appearing in logs", "error", err)
	}
}

// AutoMigrate — единственная точка входа для cmd/migrate: схема (DDL), бутстрап root-admin + дефолтных настроек.
// cmd/migrate/main.go - тонкая обвязка
func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.SugaredLogger, rootAdminID int64) error {
	if err := db.AutoMigrate(
		&userRecord{},
		&models.Category{},
		&models.Product{},
		&models.ProductItem{},
		&models.Purchase{},
		&models.AdminLog{},
		&models.Settings{},
		&models.Replenishment{},
	); err != nil {
		return err
	}
	log.Info("postgres: schema migrated")

	if err := createPartialIndexes(db); err != nil {
		return fmt.Errorf("create partial indexes: %w", err)
	}
	log.Info("postgres: partial indexes ensured")

	if err := recomputeAllCategoryStock(db); err != nil {
		return fmt.Errorf("recompute category stock: %w", err)
	}
	log.Info("postgres: category stock recomputed")

	refreshCollationVersions(db, log)

	userRepo := NewUserRepo(db, log)
	if err := userRepo.EnsureRootAdminExists(ctx, rootAdminID); err != nil {
		return fmt.Errorf("ensure root admin exists: %w", err)
	}
	log.Info("postgres: root admin ensured")

	settingsRepo := NewSettingsRepo(db, log)
	if err := settingsRepo.EnsureExists(ctx, &models.Settings{SupportUsername: "@#####"}); err != nil {
		return fmt.Errorf("ensure settings exist: %w", err)
	}
	log.Info("postgres: settings ensured")

	return nil
}
