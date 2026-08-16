// Command backend — только JSON API админ-панели, без бота и миграций
// (см. cmd/migrate). Отдельный контейнер, схема БД уже должна существовать.
package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/trottling/Telegram-Store/backend"
	rdb "github.com/trottling/Telegram-Store/internal/cache/redis"
	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/logger"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
	"github.com/trottling/Telegram-Store/internal/service"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.Logger)
	log.Info("backend: config loaded")

	redisClient, err := rdb.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("backend: failed to connect to redis: %v", err)
	}
	cacheService := rdb.NewRedisCache(redisClient, log)

	db, err := pgdb.NewClient(cfg.Postgres)
	if err != nil {
		log.Fatalf("backend: failed to connect to postgres: %v", err)
	}

	transactor := pgdb.NewGormTransactor(db, log)
	userRepo := pgdb.NewUserRepo(db, log)
	productRepo := pgdb.NewProductRepo(db, log)
	purchaseRepo := pgdb.NewPurchaseRepo(db, log)
	categoryRepo := pgdb.NewCategoryRepo(db, log)
	adminLogRepo := pgdb.NewAdminLogRepo(db, log)
	statsRepo := pgdb.NewStatsRepo(db, log)
	settingsRepo := pgdb.NewSettingsRepo(db, log)

	userService := service.NewUserSrv(userRepo, cacheService, log)
	productService := service.NewProductSrv(productRepo, cacheService, log)
	categoryService := service.NewCategorySrv(categoryRepo, productRepo, cacheService, log)
	// purchaseService тут только для чтения (админ-листинг) — Buy() не вызывается.
	purchaseService := service.NewPurchaseSrv(userRepo, productRepo, purchaseRepo, categoryRepo, transactor, cacheService, log)
	adminService := service.NewAdminSrv(userRepo, productRepo, categoryRepo, purchaseRepo, adminLogRepo, settingsRepo, cacheService, cacheService, log)
	statsService := service.NewStatsSrv(statsRepo, log)
	settingsService := service.NewSettingsSrv(settingsRepo, cacheService, log)

	// Коды выдаёт бот (/admin), этот процесс их только обменивает/проверяет.
	adminAuthService := service.NewAdminAuthSrv(userRepo, cacheService, cfg.AdminPanel.JWTSecret, log)

	webServer := backend.New(userService, productService, categoryService, purchaseService, adminService, statsService, settingsService, adminAuthService, cfg.AdminPanel, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 1)
	go func() {
		errChan <- webServer.Start()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err = webServer.Shutdown(shutdownCtx); err != nil {
			log.Warnf("backend: failed to shut down admin API server: %v", err)
		}
		cancel()
	case err = <-errChan:
		if err != nil {
			log.Errorf("backend: admin API server stopped unexpectedly: %v", err)
		}
	}

	if err = redisClient.Close(); err != nil {
		log.Warnf("backend: failed to close redis client: %v", err)
	}
}
