package middleware

import (
	"context"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/bot/utils"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"

	"github.com/trottling/Telegram-Store/internal/domain/service"
)

type Middlewares struct {
	userService          service.UserService
	purchaseService      service.PurchaseService
	productService       service.ProductService
	replenishmentService service.ReplenishmentService
	stateStore           domainfsm.Store
	log                  *logrus.Logger
	// inFlight — update'ы в обработке, см. Track/WaitInFlight.
	inFlight sync.WaitGroup
}

func New(
	userService service.UserService,
	purchaseService service.PurchaseService,
	productService service.ProductService,
	replenishmentService service.ReplenishmentService,
	stateStore domainfsm.Store,
	log *logrus.Logger,
) *Middlewares {
	return &Middlewares{
		userService:          userService,
		purchaseService:      purchaseService,
		productService:       productService,
		replenishmentService: replenishmentService,
		stateStore:           stateStore,
		log:                  log,
	}
}

// Logging логирует каждый update на уровне Debug — первый в цепочке
// middleware, видит трафик независимо от исхода.
func (m *Middlewares) Logging(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, _ := extractChatID(update)
		fields := logrus.Fields{"telegram_id": chatID, "update_id": update.ID}

		switch {
		case update.Message != nil:
			fields["kind"] = "message"
			// Только длина, не сам текст: на Debug сюда попадало всё, что пишут
			// пользователи, включая пересланный им же код входа. Для отладки
			// хватает связки update_id + шаг FSM.
			fields["text_len"] = len(update.Message.Text)
		case update.CallbackQuery != nil:
			fields["kind"] = "callback_query"
			fields["data"] = update.CallbackQuery.Data
		default:
			fields["kind"] = "other"
		}

		m.log.WithFields(fields).Debug("middleware: update received")
		next(ctx, b, update)
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
