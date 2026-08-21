// Command admin_backend — только JSON API админ-панели, без бота, вебхуков
// платежей (см. cmd/payments_backend) и миграций (см. cmd/migrate).
// Отдельный контейнер, схема БД уже должна существовать.
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
// (разбор *config.Config на подконфиги, вытаскивание JWT-секрета, буквальный
// nil вместо providers у ReplenishmentSrv) — то, что fx.Annotate не умеет по
// определению. lifecycle.go — старт/стоп самого HTTP-сервера. fx.New(...).Run()
// сам слушает SIGINT/SIGTERM и вызывает Stop у зарегистрированных хуков —
// отдельный signal.NotifyContext больше не нужен.
package main

import (
	"fmt"
	"os"

	"go.uber.org/fx"

	adminbackend "github.com/trottling/Telegram-Store/admin_backend"
	rdb "github.com/trottling/Telegram-Store/internal/cache/redis"
	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"github.com/trottling/Telegram-Store/internal/logger"
	adminmetrics "github.com/trottling/Telegram-Store/internal/metrics/admin"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
	svc "github.com/trottling/Telegram-Store/internal/service"
)

func main() {
	app := fx.New(
		fx.WithLogger(logger.NewFxLogger),
		fx.Provide(
			config.New,
			logger.New,
			providePostgresConfig,
			provideRedisConfig,
			provideAdminPanelConfig,
			provideLoggerConfig,

			rdb.NewClient,
			// cacheService — один клиент, отданный в граф сразу под все
			// интерфейсы, которые он реализует (fsm.Store сюда не входит —
			// у admin_backend нет FSM-сценариев).
			fx.Annotate(
				rdb.NewRedisCache,
				fx.As(new(domaincache.UserCache)),
				fx.As(new(domaincache.ProductCache)),
				fx.As(new(domaincache.CategoryCache)),
				fx.As(new(domaincache.SettingsCache)),
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
			fx.Annotate(pgdb.NewAdminLogRepo, fx.As(new(repository.AdminLogRepository))),
			fx.Annotate(pgdb.NewAnalyticsRepo, fx.As(new(repository.AnalyticsRepository))),
			fx.Annotate(pgdb.NewGormTransactor, fx.As(new(repository.Transactor))),

			fx.Annotate(svc.NewUserSrv, fx.As(new(service.UserService))),
			fx.Annotate(svc.NewProductSrv, fx.As(new(service.ProductService))),
			fx.Annotate(svc.NewCategorySrv, fx.As(new(service.CategoryService))),
			fx.Annotate(svc.NewSettingsSrv, fx.As(new(service.SettingsService))),
			// purchaseService тут только для чтения (админ-листинг) — Buy() не вызывается.
			fx.Annotate(svc.NewPurchaseSrv, fx.As(new(service.PurchaseService))),
			fx.Annotate(svc.NewAdminSrv, fx.As(new(service.AdminService))),
			fx.Annotate(svc.NewAnalyticsSrv, fx.As(new(service.AnalyticsService))),
			fx.Annotate(svc.NewAdminAuthSrv, fx.As(new(service.AdminAuthService))),
			provideReplenishmentService,

			provideMetricsServer,
			adminmetrics.NewAnalyticsCollector,

			adminbackend.New,
		),
		fx.Invoke(runServer, RunMetricsServer),
	)

	// Ошибку сборки графа и упавшего Invoke (например, недоступный Postgres)
	// fx.New отдаёт через app.Err(), а Run() о ней только молча выходит с
	// кодом 1. Пишем прямо в stderr: LOG_LEVEL не должен скрывать фатальный
	// сбой старта — та же причина, что и в cmd/migrate.
	if err := app.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "admin_backend: failed: %v\n", err)
		os.Exit(1)
	}

	app.Run()
}
