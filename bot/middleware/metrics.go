package middleware

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	botmetrics "github.com/trottling/Telegram-Store/internal/metrics/bot"
)

// Metrics считает update'ы и их длительность. Стоит сразу после Recover —
// внутри его defer recover(), поэтому паника ниже по цепочке не мешает этому
// defer отработать (Go гарантирует выполнение отложенных функций при
// размотке стека), но сама паника здесь не видна: PanicsTotal инкрементится
// в Recover, у которого есть recovered-значение. Outcome (успех/ошибка)
// сюда не добавляем — bot.HandlerFunc ничего не возвращает, у миддлвари
// физически нет этой информации.
func (m *Middlewares) Metrics(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		kind := classifyKind(update)
		start := time.Now()

		botmetrics.UpdatesTotal.WithLabelValues(kind).Inc()
		defer func() {
			botmetrics.UpdateDurationSeconds.WithLabelValues(kind).Observe(time.Since(start).Seconds())
		}()

		next(ctx, b, update)
	}
}
