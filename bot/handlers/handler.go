package handlers

import (
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/internal/config"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"

	"github.com/trottling/Telegram-Store/internal/domain/service"
	"github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

type Handlers struct {
	userService      service.UserService
	purchaseService  service.PurchaseService
	productService   service.ProductService
	categoryService  service.CategoryService
	paymentService   payment.PaymentProvider
	adminAuthService service.AdminAuthService
	stateStore       domainfsm.Store
	kb               *keyboards.Keyboards
	log              *logrus.Logger
	adminPanelConfig *config.AdminPanelConfig
}

func New(
	userService service.UserService,
	purchaseService service.PurchaseService,
	productService service.ProductService,
	categoryService service.CategoryService,
	paymentService payment.PaymentProvider,
	adminAuthService service.AdminAuthService,
	stateStore domainfsm.Store,
	kb *keyboards.Keyboards,
	log *logrus.Logger,
	adminPanelConfig *config.AdminPanelConfig,
) *Handlers {
	return &Handlers{
		userService:      userService,
		purchaseService:  purchaseService,
		productService:   productService,
		categoryService:  categoryService,
		paymentService:   paymentService,
		adminAuthService: adminAuthService,
		stateStore:       stateStore,
		kb:               kb,
		log:              log,
		adminPanelConfig: adminPanelConfig,
	}
}
