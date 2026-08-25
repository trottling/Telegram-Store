package main

import (
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/trottling/Telegram-Store/internal/domain/service"
)

// catalogRefreshTimeout — потолок на один пересчёт снепшота, чтобы зависший
// запрос к Postgres не держал цикл вечно вместо обычного Warn и следующей
// попытки через интервал.
const catalogRefreshTimeout = 30 * time.Second

// defaultCatalogRefreshInterval — если Settings временно недоступны (сбой
// Redis/Postgres), не крутимся в busy-loop и не виснем — ждём разумный
// интервал по умолчанию и пробуем снова.
const defaultCatalogRefreshInterval = 30 * time.Second

// RunCatalogRefreshWorker — фоновый пересчёт видимости каталога, тем же
// паттерном fx.Lifecycle, что и runBot/RunMetricsServer. Единственный
// потребитель CategorySrv.RefreshCatalogSnapshot — раньше видимость
// (Category.HasStock) пересчитывалась на каждой покупке/CRUD-операции
// синхронно, вверх по дереву, в самом ответе пользователю; здесь то же самое
// делается раз в Settings.CatalogRefreshIntervalSeconds, вне пути запроса.
// ListChildren в промежутке отдаёт то, что воркер положил в кэш в прошлый
// раз — задержка видимости в пределах интервала осознанно принята в обмен на
// отсутствие обхода дерева на каждую запись.
func RunCatalogRefreshWorker(lc fx.Lifecycle, categoryService service.CategoryService, settingsService service.SettingsService, log *zap.SugaredLogger) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go runCatalogRefreshLoop(ctx, categoryService, settingsService, log)
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
}

func runCatalogRefreshLoop(ctx context.Context, categoryService service.CategoryService, settingsService service.SettingsService, log *zap.SugaredLogger) {
	for {
		refreshCtx, refreshCancel := context.WithTimeout(ctx, catalogRefreshTimeout)
		err := categoryService.RefreshCatalogSnapshot(refreshCtx)
		refreshCancel()
		if err != nil {
			log.Warnw("bot: catalog snapshot refresh failed, keeping previous snapshot until next tick", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(nextCatalogRefreshInterval(ctx, settingsService, log)):
		}
	}
}

// nextCatalogRefreshInterval читает интервал из Settings заново на каждом
// цикле — правка в панели подхватывается без рестарта бота, тем же принципом,
// что и у платёжных провайдеров (см. internal/service/payment).
func nextCatalogRefreshInterval(ctx context.Context, settingsService service.SettingsService, log *zap.SugaredLogger) time.Duration {
	settings, err := settingsService.Get(ctx)
	if err != nil {
		log.Warnw("bot: failed to read catalog refresh interval, using default", "error", err, "default_seconds", int(defaultCatalogRefreshInterval.Seconds()))
		return defaultCatalogRefreshInterval
	}
	return time.Duration(settings.CatalogRefreshIntervalSeconds) * time.Second
}
