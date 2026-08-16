package handlers

import (
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/internal/config"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"

	"github.com/trottling/Telegram-Store/internal/domain/service"
)

type Handlers struct {
	userService          service.UserService
	purchaseService      service.PurchaseService
	productService       service.ProductService
	categoryService      service.CategoryService
	settingsService      service.SettingsService
	replenishmentService service.ReplenishmentService
	adminAuthService     service.AdminAuthService
	stateStore           domainfsm.Store
	kb                   *keyboards.Keyboards
	log                  *logrus.Logger
	adminPanelConfig     *config.AdminPanelConfig
	// botUsername — для реф-ссылок (t.me/<botUsername>?start=<id>), см. bot.New.
	botUsername string
}

func New(
	userService service.UserService,
	purchaseService service.PurchaseService,
	productService service.ProductService,
	categoryService service.CategoryService,
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	adminAuthService service.AdminAuthService,
	stateStore domainfsm.Store,
	kb *keyboards.Keyboards,
	log *logrus.Logger,
	adminPanelConfig *config.AdminPanelConfig,
	botUsername string,
) *Handlers {
	return &Handlers{
		userService:          userService,
		purchaseService:      purchaseService,
		productService:       productService,
		categoryService:      categoryService,
		settingsService:      settingsService,
		replenishmentService: replenishmentService,
		adminAuthService:     adminAuthService,
		stateStore:           stateStore,
		kb:                   kb,
		log:                  log,
		adminPanelConfig:     adminPanelConfig,
		botUsername:          botUsername,
	}
}
