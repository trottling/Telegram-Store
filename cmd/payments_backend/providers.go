package main

import (
	"github.com/sirupsen/logrus"

	"github.com/trottling/Telegram-Store/internal/config"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
	svc "github.com/trottling/Telegram-Store/internal/service"
	"github.com/trottling/Telegram-Store/internal/service/payment"
)

// Сабконфиги *config.Config — см. cmd/admin_backend/providers.go, тот же
// принцип: единственное, что нужно писать руками для fx-графа.
func providePostgresConfig(cfg *config.Config) *config.PostgresConfig { return cfg.Postgres }
func provideRedisConfig(cfg *config.Config) *config.RedisConfig       { return cfg.Redis }
func providePaymentsConfig(cfg *config.Config) *config.PaymentsConfig { return cfg.Payments }
func provideLoggerConfig(cfg *config.Config) *config.LoggerConfig     { return cfg.Logger }

// provideCrystalPayProvider — нужен только вебхуку CrystalPay, чтобы
// перепроверить статус счёта: подпись мерчанта покрывает лишь id, а state в том
// же теле не подписан. Провайдеры двух других мерчантов здесь не заводим —
// Tinkoff подписывает всё тело, YooKassa перезапрашивается своим SDK в
// хендлере. URL берётся из конфига по единообразию: счетов этот бинарник не
// создаёт, так что callback_url в нём не используется.
func provideCrystalPayProvider(settingsService service.SettingsService, paymentsCfg *config.PaymentsConfig) domainpayment.PaymentProvider {
	return payment.NewCrystalPayProvider(settingsService, paymentsCfg.URL)
}

// provideReplenishmentService — providers=nil (не тип из графа, а буквально
// nil): payments_backend счета не создаёт (только принимает вебхуки),
// CreateInvoice отсюда не вызывается. fx.Annotate тут не подходит — это не
// просто приведение типа, а реальный "пустой" аргумент.
func provideReplenishmentService(repo repository.ReplenishmentRepository, userRepo repository.UserRepository, transactor repository.Transactor, cache domaincache.UserCache, log *logrus.Logger) service.ReplenishmentService {
	return svc.NewReplenishmentSrv(repo, userRepo, transactor, nil, cache, log)
}
