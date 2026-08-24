package middleware

import (
	"context"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/utils"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
	"go.uber.org/zap"

	"github.com/trottling/Telegram-Store/internal/domain/service"
)

type Middlewares struct {
	userService          service.UserService
	purchaseService      service.PurchaseService
	productService       service.ProductService
	replenishmentService service.ReplenishmentService
	stateStore           domainfsm.Store
	log                  *zap.SugaredLogger
	// inFlight — update'ы в обработке, см. Track/WaitInFlight.
	inFlight sync.WaitGroup
	// sem — семафор на maxConcurrentUpdates одновременно обрабатываемых
	// update'ов, см. Track.
	sem chan struct{}
}

func New(
	userService service.UserService,
	purchaseService service.PurchaseService,
	productService service.ProductService,
	replenishmentService service.ReplenishmentService,
	stateStore domainfsm.Store,
	log *zap.SugaredLogger,
) *Middlewares {
	return &Middlewares{
		userService:          userService,
		purchaseService:      purchaseService,
		productService:       productService,
		replenishmentService: replenishmentService,
		stateStore:           stateStore,
		log:                  log,
		sem:                  make(chan struct{}, maxConcurrentUpdates),
	}
}

// Logging логирует каждый update на уровне Debug — первый в цепочке
// middleware, видит трафик независимо от исхода.
func (m *Middlewares) Logging(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, _ := extractChatID(update)
		kind := classifyKind(update)
		fields := []any{"telegram_id", chatID, "update_id", update.ID, "kind", kind}

		switch {
		case update.Message != nil:
			// Только длина, не сам текст: на Debug сюда попадало всё, что пишут
			// пользователи, включая пересланный им же код входа. Для отладки
			// хватает связки update_id + шаг FSM.
			fields = append(fields, "text_len", len(update.Message.Text))
		case update.CallbackQuery != nil:
			fields = append(fields, "data", update.CallbackQuery.Data)
		}

		m.log.Debugw("middleware: update received", fields...)
		next(ctx, b, update)
	}
}

// classifyKind — message/callback_query/other; переиспользуется Logging и
// Metrics, чтобы не заводить два разных источника истины об одной и той же
// классификации.
func classifyKind(update *models.Update) string {
	switch {
	case update.Message != nil:
		return "message"
	case update.CallbackQuery != nil:
		return "callback_query"
	default:
		return "other"
	}
}

// extractChatID достаёт chat ID из update; ok — false, если его нет ни в одном виде.
func extractChatID(update *models.Update) (chatID int64, ok bool) {
	if update.Message != nil {
		return update.Message.Chat.ID, true
	}
	return utils.CallbackChatID(update)
}

// extractLanguageCode достаёт From.LanguageCode из update — Message.From
// nilable, CallbackQuery.From — значение, не указатель.
func extractLanguageCode(update *models.Update) string {
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.LanguageCode
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.LanguageCode
	}
	return ""
}
