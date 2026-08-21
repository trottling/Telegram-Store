package main

import (
	"net/http"

	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	adminmetrics "github.com/trottling/Telegram-Store/internal/metrics/admin"
	svc "github.com/trottling/Telegram-Store/internal/service"
	"go.uber.org/zap"
)

// Сабконфиги *config.Config — на fx-графе значение отдаётся ровно одного
// типа, так что для полей одного *Config нужны отдельные "экстракторы". Это
// единственное, что реально нужно писать руками — привязку конкретных
// конструкторов репозиториев/сервисов к доменным интерфейсам делает
// fx.Annotate(..., fx.As(...)) прямо в списке fx.Provide в main.go, отдельные
// провайдер-функции под них не нужны.
func providePostgresConfig(cfg *config.Config) *config.PostgresConfig     { return cfg.Postgres }
func provideRedisConfig(cfg *config.Config) *config.RedisConfig           { return cfg.Redis }
func provideAdminPanelConfig(cfg *config.Config) *config.AdminPanelConfig { return cfg.AdminPanel }
func provideLoggerConfig(cfg *config.Config) *config.LoggerConfig         { return cfg.Logger }

// provideReplenishmentService — providers=nil (не тип из графа, а буквально
// nil): admin_backend счета не создаёт и вебхуки не принимает (см.
// payments_backend), только отдаёт листинг — CreateInvoice отсюда не
// вызывается. fx.Annotate тут не подходит — это не просто приведение типа,
// а реальный "пустой" аргумент.
func provideReplenishmentService(repo repository.ReplenishmentRepository, userRepo repository.UserRepository, transactor repository.Transactor, cache domaincache.UserCache, log *zap.SugaredLogger) service.ReplenishmentService {
	return svc.NewReplenishmentSrv(repo, userRepo, transactor, nil, cache, log)
}

// provideAdminAuthService — тоже не просто приведение к интерфейсу: JWT-секрет
// достаётся из подконфига, а не приходит отдельным типом из графа. Коды
// выдаёт бот (/admin), этот процесс их только обменивает/проверяет.
func provideAdminAuthService(userRepo repository.UserRepository, store adminsession.Store, adminPanelCfg *config.AdminPanelConfig, log *zap.SugaredLogger) service.AdminAuthService {
	return svc.NewAdminAuthSrv(userRepo, store, adminPanelCfg.JWTSecret, log)
}

// provideMetricsServer — свой /metrics-сервер admin_backend (бизнес-метрики
// админских действий + агрегаты Grafana, см. internal/metrics/admin),
// отдельно от собственного порта API.
func provideMetricsServer(adminPanelCfg *config.AdminPanelConfig) *http.Server {
	return adminmetrics.NewServer(adminPanelCfg.MetricsPort)
}
