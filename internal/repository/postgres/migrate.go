package postgres

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/trottling/TG-Store/internal/domain/models"
	"gorm.io/gorm"
)

// AutoMigrate создаёт/обновляет схему для всех доменных сущностей. Вызывается один раз из cmd/migrate.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Product{},
		&models.ProductItem{},
		&models.Purchase{},
		&models.AdminLog{},
	)
}

// BackfillUserRoles переносит старые is_banned/is_admin в новую колонку
// role и дропает их. Идемпотентно (no-op, если колонок уже нет). Порядок
// трёх апдейтов важен: banned приоритетнее admin. Всё в одной транзакции
// вместе с DROP COLUMN.
func BackfillUserRoles(ctx context.Context, db *gorm.DB, log *logrus.Logger) error {
	if !db.Migrator().HasColumn(&models.User{}, "is_banned") {
		return nil
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if _, err := gorm.G[models.User](tx).Where("is_banned = ?", true).Update(ctx, "role", models.RoleBanned); err != nil {
			return fmt.Errorf("backfill banned users: %w", err)
		}
		if _, err := gorm.G[models.User](tx).
			Where("is_banned = ? AND is_admin = ?", false, true).
			Update(ctx, "role", models.RoleAdmin); err != nil {
			return fmt.Errorf("backfill admin users: %w", err)
		}
		if _, err := gorm.G[models.User](tx).
			Where("is_banned = ? AND is_admin = ?", false, false).
			Update(ctx, "role", models.RoleUser); err != nil {
			return fmt.Errorf("backfill plain users: %w", err)
		}

		if err := tx.Migrator().DropColumn(&models.User{}, "is_banned"); err != nil {
			return fmt.Errorf("drop is_banned column: %w", err)
		}
		if err := tx.Migrator().DropColumn(&models.User{}, "is_admin"); err != nil {
			return fmt.Errorf("drop is_admin column: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Info("postgres: backfilled legacy is_banned/is_admin into role, dropped old columns")
	return nil
}

// DropLegacyAdminTokenColumn дропает users.admin_token_hash — остаток
// старой схемы авторизации, ныне не используется. Идемпотентно.
func DropLegacyAdminTokenColumn(db *gorm.DB, log *logrus.Logger) error {
	if !db.Migrator().HasColumn(&models.User{}, "admin_token_hash") {
		return nil
	}

	if err := db.Migrator().DropColumn(&models.User{}, "admin_token_hash"); err != nil {
		return fmt.Errorf("drop admin_token_hash column: %w", err)
	}

	log.Info("postgres: dropped legacy admin_token_hash column")
	return nil
}
