package main

import (
	"github.com/sirupsen/logrus"

	"github.com/trottling/Telegram-Store/internal/config"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	svc "github.com/trottling/Telegram-Store/internal/service"
)

// Сабконфиги *config.Config — см. cmd/admin_backend/providers.go, тот же
// принцип: единственное, что нужно писать руками для fx-графа.
func providePostgresConfig(cfg *config.Config) *config.PostgresConfig { return cfg.Postgres }
func provideRedisConfig(cfg *config.Config) *config.RedisConfig       { return cfg.Redis }
func providePaymentsConfig(cfg *config.Config) *config.PaymentsConfig { return cfg.Payments }
func provideLoggerConfig(cfg *config.Config) *config.LoggerConfig     { return cfg.Logger }

// provideReplenishmentService — providers=nil (не тип из графа, а буквально
// nil): payments_backend счета не создаёт (только принимает вебхуки),
// CreateInvoice отсюда не вызывается. fx.Annotate тут не подходит — это не
// просто приведение типа, а реальный "пустой" аргумент.
func provideReplenishmentService(repo repository.ReplenishmentRepository, userRepo repository.UserRepository, cache domaincache.UserCache, log *logrus.Logger) service.ReplenishmentService {
	return svc.NewReplenishmentSrv(repo, userRepo, nil, cache, log)
}
