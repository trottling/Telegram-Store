package main

import (
	"github.com/sirupsen/logrus"

	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domainmodels "github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
	svc "github.com/trottling/Telegram-Store/internal/service"
	"github.com/trottling/Telegram-Store/internal/service/payment"
)

// Сабконфиги *config.Config — на fx-графе значение отдаётся ровно одного
// типа, так что для полей одного *Config нужны отдельные "экстракторы". Это
// единственное, что реально нужно писать руками — привязку конкретных
// конструкторов репозиториев/сервисов к доменным интерфейсам делает
// fx.Annotate(..., fx.As(...)) прямо в списке fx.Provide в main.go, отдельные
// провайдер-функции под них не нужны.
func provideTelegramConfig(cfg *config.Config) *config.TelegramConfig     { return cfg.Telegram }
func providePostgresConfig(cfg *config.Config) *config.PostgresConfig     { return cfg.Postgres }
func provideRedisConfig(cfg *config.Config) *config.RedisConfig           { return cfg.Redis }
func provideAdminPanelConfig(cfg *config.Config) *config.AdminPanelConfig { return cfg.AdminPanel }
func provideLoggerConfig(cfg *config.Config) *config.LoggerConfig         { return cfg.Logger }

// providePaymentProviders — по одному провайдеру на реальный мерчант;
// MerchantReferral сюда не входит, начисления с рефералов создаются
// напрямую, без CreateInvoice (см. domain/models.Replenishment). Реальная
// сборка map — не просто приведение типа, поэтому fx.Annotate тут не
// подходит, нужна обычная функция.
func providePaymentProviders(settingsService service.SettingsService, adminPanelCfg *config.AdminPanelConfig) map[domainmodels.Merchant]domainpayment.PaymentProvider {
	return map[domainmodels.Merchant]domainpayment.PaymentProvider{
		domainmodels.MerchantCrystalPay: payment.NewCrystalPayProvider(settingsService, adminPanelCfg.URL),
		domainmodels.MerchantYooKassa:   payment.NewYooKassaProvider(settingsService),
		domainmodels.MerchantTinkoff:    payment.NewTinkoffProvider(settingsService, adminPanelCfg.URL),
	}
}

// provideAdminAuthService — тоже не просто приведение к интерфейсу: JWT-секрет
// достаётся из подконфига, а не приходит отдельным типом из графа. Бот сам
// AdminService не вызывает, только выдаёт код для /admin.
func provideAdminAuthService(userRepo repository.UserRepository, store adminsession.Store, adminPanelCfg *config.AdminPanelConfig, log *logrus.Logger) service.AdminAuthService {
	return svc.NewAdminAuthSrv(userRepo, store, adminPanelCfg.JWTSecret, log)
}
