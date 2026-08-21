package main

import (
	"github.com/trottling/Telegram-Store/internal/config"
	domainmodels "github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
	"github.com/trottling/Telegram-Store/internal/service/payment"
	"go.uber.org/zap"
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
func providePaymentsConfig(cfg *config.Config) *config.PaymentsConfig     { return cfg.Payments }
func provideMetricsConfig(cfg *config.Config) *config.MetricsConfig       { return cfg.Metrics }
func provideLoggerConfig(cfg *config.Config) *config.LoggerConfig         { return cfg.Logger }

// providePaymentProviders — по одному провайдеру на мерчант, включая
// DummyProvider (тестовый, без реальной оплаты — см. internal/service/payment/dummy.go);
// MerchantReferral сюда не входит, начисления с рефералов создаются
// напрямую, без CreateInvoice (см. domain/models.Replenishment). Реальная
// сборка map — не просто приведение типа, поэтому fx.Annotate тут не
// подходит, нужна обычная функция. Колбэки CrystalPay/Tinkoff/Dummy указывают
// на payments_backend (paymentsCfg.URL), не на admin_backend — вебхуки принимает он.
func providePaymentProviders(settingsService service.SettingsService, paymentsCfg *config.PaymentsConfig, log *zap.SugaredLogger) map[domainmodels.Merchant]domainpayment.PaymentProvider {
	// Забытый PAYMENTS_BACKEND_URL иначе никак не заметить: счёт создастся,
	// ссылка на оплату будет рабочей, деньги спишутся — а подтверждение не
	// придёт, потому что мерчант стучится из интернета на localhost. Отличить
	// прод от разработки в конфиге нельзя (переменная одна), поэтому шумим тут.
	if paymentsCfg.IsLoopbackURL() {
		log.Warnw("bot: payments callback URL is a loopback address, merchant webhooks will never arrive", "payments_backend_url", paymentsCfg.URL)
	}

	return map[domainmodels.Merchant]domainpayment.PaymentProvider{
		domainmodels.MerchantCrystalPay: payment.NewCrystalPayProvider(settingsService, paymentsCfg.URL),
		domainmodels.MerchantYooKassa:   payment.NewYooKassaProvider(settingsService),
		domainmodels.MerchantTinkoff:    payment.NewTinkoffProvider(settingsService, paymentsCfg.URL),
		domainmodels.MerchantDummy:      payment.NewDummyProvider(settingsService, paymentsCfg.URL, log),
	}
}
