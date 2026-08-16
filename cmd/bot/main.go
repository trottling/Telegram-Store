// Command bot — только Telegram-часть приложения, без admin API и миграций
// (см. cmd/migrate). Отдельный контейнер, схема БД уже должна существовать.
package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/trottling/Telegram-Store/bot"
	rdb "github.com/trottling/Telegram-Store/internal/cache/redis"
	"github.com/trottling/Telegram-Store/internal/config"
	domainmodels "github.com/trottling/Telegram-Store/internal/domain/models"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
	"github.com/trottling/Telegram-Store/internal/logger"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
	"github.com/trottling/Telegram-Store/internal/service"
	"github.com/trottling/Telegram-Store/internal/service/payment"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.Logger)
	log.Info("bot: config loaded")

	redisClient, err := rdb.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("bot: failed to connect to redis: %v", err)
	}

	// cacheService — и FSM-хранилище, и read-through кэш, один клиент.
	cacheService := rdb.NewRedisCache(redisClient, log)

	db, err := pgdb.NewClient(cfg.Postgres)
	if err != nil {
		log.Fatalf("bot: failed to connect to postgres: %v", err)
	}

	// Только то, что нужно боту — adminLogRepo/statsRepo тут не нужны.
	transactor := pgdb.NewGormTransactor(db, log)
	userRepo := pgdb.NewUserRepo(db, log)
	productRepo := pgdb.NewProductRepo(db, log)
	purchaseRepo := pgdb.NewPurchaseRepo(db, log)
	categoryRepo := pgdb.NewCategoryRepo(db, log)
	settingsRepo := pgdb.NewSettingsRepo(db, log)
	replenishmentRepo := pgdb.NewReplenishmentRepo(db, log)

	userService := service.NewUserSrv(userRepo, cacheService, log)
	productService := service.NewProductSrv(productRepo, cacheService, log)
	purchaseService := service.NewPurchaseSrv(userRepo, productRepo, purchaseRepo, categoryRepo, transactor, cacheService, log)
	categoryService := service.NewCategorySrv(categoryRepo, productRepo, cacheService, log)
	settingsService := service.NewSettingsSrv(settingsRepo, cacheService, log)

	// Один провайдер на мерчанта — MerchantReferral сюда не входит, начисления
	// с рефералов через CreateInvoice не идут (см. domain/models.Replenishment).
	providers := map[domainmodels.Merchant]domainpayment.PaymentProvider{
		domainmodels.MerchantCrystalPay: payment.NewCrystalPayProvider(settingsService, cfg.AdminPanel.URL),
		domainmodels.MerchantYooKassa:   payment.NewYooKassaProvider(settingsService),
		domainmodels.MerchantTinkoff:    payment.NewTinkoffProvider(settingsService, cfg.AdminPanel.URL),
	}
	replenishmentService := service.NewReplenishmentSrv(replenishmentRepo, userRepo, providers, cacheService, log)

	// Бот сам AdminService не вызывает — только выдаёт код для /admin.
	adminAuthService := service.NewAdminAuthSrv(userRepo, cacheService, cfg.AdminPanel.JWTSecret, log)

	b, err := bot.New(userService, productService, purchaseService, categoryService, settingsService, replenishmentService, adminAuthService, cacheService, cfg.Telegram, cfg.AdminPanel, log)
	if err != nil {
		log.Fatalf("bot: failed to init bot: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	b.Start(ctx)

	if err = redisClient.Close(); err != nil {
		log.Warnf("bot: failed to close redis client: %v", err)
	}
}
