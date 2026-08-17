// Command bot — только Telegram-часть приложения, без admin API и миграций
// (см. cmd/migrate). Отдельный контейнер, схема БД уже должна существовать.
//
// Граф зависимостей собирает go.uber.org/fx. Каждый репозиторий/сервис ниже
// возвращает конкретный тип (*pgdb.UserRepo, *service.UserSrv, ...) — сам
// пакет не знает про fx и не должен об этом знать (fx — забота composition
// root, не доменного/сервисного слоя). fx.Annotate(ctor, fx.As(new(Iface)))
// регистрирует его в графе под доменным интерфейсом, который ждут
// потребители — тот же принцип "наружу из composition root уходит только
// интерфейс", что раньше был в ручной сборке, просто через штатный
// механизм fx, а не через написанные вручную provideX-обёртки. providers.go
// оставляет обычные функции только там, где действительно есть логика
// (сборка map провайдеров платежей, разбор *config.Config на подконфиги,
// вытаскивание JWT-секрета) — то, что fx.Annotate не умеет по определению.
// lifecycle.go — старт/стоп самого бота. fx.New(...).Run() сам слушает
// SIGINT/SIGTERM и вызывает Stop у зарегистрированных хуков — отдельный
// signal.NotifyContext больше не нужен.
package main

import (
	"go.uber.org/fx"

	"github.com/trottling/Telegram-Store/bot"
	rdb "github.com/trottling/Telegram-Store/internal/cache/redis"
	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"github.com/trottling/Telegram-Store/internal/logger"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
	svc "github.com/trottling/Telegram-Store/internal/service"
)

func main() {
	fx.New(
		fx.WithLogger(logger.NewFxLogger),
		fx.Provide(
			config.New,
			logger.New,
			provideTelegramConfig,
			providePostgresConfig,
			provideRedisConfig,
			provideAdminPanelConfig,
			providePaymentsConfig,
			provideLoggerConfig,

			rdb.NewClient,
			// cacheService — и FSM-хранилище, и read-through кэш, один клиент,
			// отданный в граф сразу под все интерфейсы, которые он реализует
			// (единственное место, где fx.Annotate нужен ради нескольких
			// fx.As на одном конструкторе, а не просто ради одного).
			fx.Annotate(
				rdb.NewRedisCache,
				fx.As(new(domaincache.UserCache)),
				fx.As(new(domaincache.ProductCache)),
				fx.As(new(domaincache.CategoryCache)),
				fx.As(new(domaincache.SettingsCache)),
				fx.As(new(domainfsm.Store)),
				fx.As(new(adminsession.Store)),
				fx.As(new(svc.MultiCache)),
			),

			pgdb.NewClient,
			fx.Annotate(pgdb.NewUserRepo, fx.As(new(repository.UserRepository))),
			fx.Annotate(pgdb.NewProductRepo, fx.As(new(repository.ProductRepository))),
			fx.Annotate(pgdb.NewPurchaseRepo, fx.As(new(repository.PurchaseRepository))),
			fx.Annotate(pgdb.NewCategoryRepo, fx.As(new(repository.CategoryRepository))),
			fx.Annotate(pgdb.NewSettingsRepo, fx.As(new(repository.SettingsRepository))),
			fx.Annotate(pgdb.NewReplenishmentRepo, fx.As(new(repository.ReplenishmentRepository))),
			fx.Annotate(pgdb.NewGormTransactor, fx.As(new(repository.Transactor))),

			fx.Annotate(svc.NewUserSrv, fx.As(new(service.UserService))),
			fx.Annotate(svc.NewProductSrv, fx.As(new(service.ProductService))),
			fx.Annotate(svc.NewCategorySrv, fx.As(new(service.CategoryService))),
			fx.Annotate(svc.NewSettingsSrv, fx.As(new(service.SettingsService))),
			fx.Annotate(svc.NewPurchaseSrv, fx.As(new(service.PurchaseService))),
			providePaymentProviders,
			fx.Annotate(svc.NewReplenishmentSrv, fx.As(new(service.ReplenishmentService))),
			provideAdminAuthService,

			bot.New,
		),
		fx.Invoke(runBot),
	).Run()
}
