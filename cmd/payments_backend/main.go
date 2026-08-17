// Command payments_backend — только приём вебхуков платёжных мерчантов,
// без админки и без бота. Отдельный контейнер, схема БД уже должна
// существовать. Узкий граф зависимостей: ни UserService/AdminAuthService,
// ни Product/Category/AdminLog/Stats репозиториев/сервисов — вебхукам они
// не нужны (см. payments_backend/handlers.Handlers — всего два сервиса).
//
// Граф зависимостей собирает go.uber.org/fx — тот же принцип, что в
// cmd/admin_backend и cmd/bot: конструкторы возвращают конкретные типы,
// fx.Annotate(ctor, fx.As(new(Iface))) регистрирует их в графе под нужным
// доменным интерфейсом. lifecycle.go — старт/стоп HTTP-сервера.
// fx.New(...).Run() сам слушает SIGINT/SIGTERM.
package main

import (
	"go.uber.org/fx"

	"github.com/trottling/Telegram-Store/internal/config"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"github.com/trottling/Telegram-Store/internal/logger"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
	svc "github.com/trottling/Telegram-Store/internal/service"

	rdb "github.com/trottling/Telegram-Store/internal/cache/redis"
	paymentsbackend "github.com/trottling/Telegram-Store/payments_backend"
)

func main() {
	fx.New(
		fx.WithLogger(logger.NewFxLogger),
		fx.Provide(
			config.New,
			logger.New,
			providePostgresConfig,
			provideRedisConfig,
			providePaymentsConfig,
			provideLoggerConfig,

			rdb.NewClient,
			// cacheService — тот же клиент, что у admin_backend/bot, но здесь
			// регистрируется только под теми интерфейсами, что реально нужны:
			// UserCache (ReplenishmentSrv) и SettingsCache (SettingsSrv). Ни
			// ProductCache/CategoryCache, ни adminsession.Store, ни MultiCache —
			// payments_backend ничего админского не знает.
			fx.Annotate(
				rdb.NewRedisCache,
				fx.As(new(domaincache.UserCache)),
				fx.As(new(domaincache.SettingsCache)),
			),

			pgdb.NewClient,
			fx.Annotate(pgdb.NewUserRepo, fx.As(new(repository.UserRepository))),
			fx.Annotate(pgdb.NewSettingsRepo, fx.As(new(repository.SettingsRepository))),
			fx.Annotate(pgdb.NewReplenishmentRepo, fx.As(new(repository.ReplenishmentRepository))),
			// Транзактор нужен ReplenishmentSrv.Confirm: смена статуса счёта и
			// начисление баланса должны коммититься вместе, иначе ретрай
			// вебхука уже ничего не исправит.
			fx.Annotate(pgdb.NewGormTransactor, fx.As(new(repository.Transactor))),

			fx.Annotate(svc.NewSettingsSrv, fx.As(new(service.SettingsService))),
			provideReplenishmentService,
			provideCrystalPayProvider,

			paymentsbackend.New,
		),
		fx.Invoke(runServer),
	).Run()
}
