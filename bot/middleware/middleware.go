package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sirupsen/logrus"
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
			fields["text"] = update.Message.Text
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
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Message.Chat.ID, true
	}
	return 0, false
}
