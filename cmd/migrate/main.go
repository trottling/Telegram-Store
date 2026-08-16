// Command migrate — одноразовый прогон pgdb.AutoMigrate (схема, легаси-клинапы,
// бутстрап root-admin и дефолтных настроек). Отдельный от cmd/bot и cmd/backend
// бинарник, чтобы два долгоживущих сервиса не гоняли миграцию параллельно —
// docker-compose ждёт его завершения перед стартом остальных.
package main

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/logger"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
)

func main() {
	ctx := context.Background()

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.Logger)
	log.Info("migrate: config loaded")

	db, err := pgdb.NewClient(cfg.Postgres)
	if err != nil {
		log.Fatalf("migrate: failed to connect to postgres: %v", err)
	}

	if err = pgdb.AutoMigrate(ctx, db, log, cfg.Telegram.RootAdminID); err != nil {
		log.Fatalf("migrate: failed to run migrations: %v", err)
	}
	log.Info("migrate: done")
}
