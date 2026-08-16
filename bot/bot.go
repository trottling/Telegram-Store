package bot

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/bot/handlers"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/bot/middleware"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	"github.com/trottling/Telegram-Store/internal/config"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
	"github.com/trottling/Telegram-Store/internal/domain/service"
)

type TelegramBot struct {
	bot                  *bot.Bot
	config               *config.Config
	log                  *logrus.Logger
	userService          service.UserService
	productService       service.ProductService
	purchaseService      service.PurchaseService
	categoryService      service.CategoryService
	settingsService      service.SettingsService
	replenishmentService service.ReplenishmentService
}

func New(
	userService service.UserService,
	productService service.ProductService,
	purchaseService service.PurchaseService,
	categoryService service.CategoryService,
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	adminAuthService service.AdminAuthService,
	stateStore domainfsm.Store,
	telegramConfig *config.TelegramConfig,
	adminPanelConfig *config.AdminPanelConfig,
	log *logrus.Logger,
) (*TelegramBot, error) {

	log.Info("initializing bot")

	kb := keyboards.New(adminPanelConfig)

	middlewares := middleware.New(userService, purchaseService, productService, replenishmentService, stateStore, log)

	handler := handlers.New(userService, purchaseService, productService, categoryService, settingsService, replenishmentService, adminAuthService, stateStore, kb, log, adminPanelConfig)

	b, err := bot.New(telegramConfig.Token, bot.WithMiddlewares(middlewares.Logging, middlewares.AnswerCallback, middlewares.BanCheck, middlewares.FSM))
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handler.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/admin", bot.MatchTypeExact, handler.AdminHandler)

	b.RegisterHandler(bot.HandlerTypeMessageText, texts.HelpBtn, bot.MatchTypeExact, handler.HelpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, texts.CatalogBtn, bot.MatchTypeExact, handler.CatalogHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, texts.ProfileBtn, bot.MatchTypeExact, handler.ProfileHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, texts.StartMenuBtn, bot.MatchTypeExact, handler.StartMenuHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, texts.PurchasesBtn, bot.MatchTypeExact, handler.PurchasesHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, texts.RefillBalanceBtn, bot.MatchTypeExact, handler.RefillBalanceHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, texts.ProfileRefreshBtn, bot.MatchTypeExact, handler.ProfileRefreshHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, texts.ReplenishmentsBtn, bot.MatchTypeExact, handler.ReplenishmentsHandler)

	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.ProductCallbackPrefix, bot.MatchTypePrefix, handler.ProductHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.BuyCallbackPrefix, bot.MatchTypePrefix, handler.BuyHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.BuyQtyCallbackPrefix, bot.MatchTypePrefix, handler.BuyQtyHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.BuyCancelCallback, bot.MatchTypeExact, handler.BuyCancelHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.BuyConfirmCallback, bot.MatchTypeExact, handler.BuyConfirmHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.PurchaseCallbackPrefix, bot.MatchTypePrefix, handler.PurchaseDetailHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.PurchasesPageCallbackPrefix, bot.MatchTypePrefix, handler.PurchasesPageHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.CategoryCallbackPrefix, bot.MatchTypePrefix, handler.CategoryHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.CatalogRootCallback, bot.MatchTypeExact, handler.CatalogRootHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.MainMenuCallback, bot.MatchTypeExact, handler.MainMenuHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.StartProfileCallback, bot.MatchTypeExact, handler.ProfileCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.RefillMerchantCallbackPrefix, bot.MatchTypePrefix, handler.RefillMerchantHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.ReplenishmentsPageCallbackPrefix, bot.MatchTypePrefix, handler.ReplenishmentsPageHandler)

	return &TelegramBot{
		log:                  log,
		bot:                  b,
		userService:          userService,
		productService:       productService,
		purchaseService:      purchaseService,
		categoryService:      categoryService,
		settingsService:      settingsService,
		replenishmentService: replenishmentService,
	}, nil
}

// Start запускает long-polling и блокируется до отмены ctx.
func (bt *TelegramBot) Start(ctx context.Context) {
	bt.log.Info("starting bot")
	bt.bot.Start(ctx)
}
