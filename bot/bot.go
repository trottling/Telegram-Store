package bot

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/handlers"
	"github.com/trottling/Telegram-Store/bot/keyboards"
	"github.com/trottling/Telegram-Store/bot/middleware"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	"github.com/trottling/Telegram-Store/internal/config"
	domainfsm "github.com/trottling/Telegram-Store/internal/domain/fsm"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"go.uber.org/zap"
)

// matchAnyLang матчит текст сообщения с подписью кнопки messageID на любом
// поддерживаемом языке — подпись reply-кнопки теперь зависит от языка пользователя
func matchAnyLang(messageID string) bot.MatchFunc {
	return func(update *models.Update) bool {
		if update.Message == nil {
			return false
		}
		for _, lang := range texts.SupportedLanguages {
			if update.Message.Text == texts.T(lang, messageID, nil) {
				return true
			}
		}
		return false
	}
}

type TelegramBot struct {
	bot                  *bot.Bot
	middlewares          *middleware.Middlewares
	config               *config.Config
	webhookConfig        *config.BotWebhookConfig
	log                  *zap.SugaredLogger
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
	stateStore domainfsm.Store,
	telegramConfig *config.TelegramConfig,
	adminPanelConfig *config.AdminPanelConfig,
	webhookConfig *config.BotWebhookConfig,
	log *zap.SugaredLogger,
) (*TelegramBot, error) {

	log.Info("initializing bot")

	kb := keyboards.New(adminPanelConfig)

	middlewares := middleware.New(userService, purchaseService, productService, replenishmentService, stateStore, log)

	opts := []bot.Option{bot.WithMiddlewares(middlewares.Track, middlewares.Recover, middlewares.Metrics, middlewares.Logging, middlewares.AnswerCallback, middlewares.BanCheck, middlewares.FSM)}
	if webhookConfig.URL != "" {
		opts = append(opts, bot.WithWebhookSecretToken(webhookConfig.Secret))
	}

	b, err := bot.New(telegramConfig.Token, opts...)
	if err != nil {
		return nil, err
	}

	// Username бота нужен для реф-ссылок (t.me/<username>?start=<id>) — берём
	// один раз при старте, не на каждый показ ReferralMsg.
	me, err := b.GetMe(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get bot username: %w", err)
	}

	if webhookConfig.URL == "" {
		// Если раньше бот работал вебхуком (или его переключили обратно на
		// polling), Telegram отказывает getUpdates, пока вебхук
		// зарегистрирован ("can't use getUpdates method while webhook is
		// active") — снимаем его на всякий случай, no-op если ничего не было.
		if _, err = b.DeleteWebhook(context.Background(), &bot.DeleteWebhookParams{}); err != nil {
			return nil, fmt.Errorf("delete telegram webhook: %w", err)
		}
	}

	handler := handlers.New(userService, purchaseService, productService, categoryService, settingsService, replenishmentService, stateStore, kb, log, adminPanelConfig, me.Username)

	// MatchTypeCommandStartOnly — команда матчится по entity, не по подстроке
	// текста, так что "/start" и "/start <id>" (deep-link реф-ссылки) оба
	// проходят; паттерн без слэша — команда в entity хранится без него.
	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommandStartOnly, handler.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/admin", bot.MatchTypeExact, handler.AdminHandler)

	b.RegisterHandlerMatchFunc(matchAnyLang(texts.HelpBtn), handler.HelpHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.CatalogBtn), handler.CatalogHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.ProfileBtn), handler.ProfileHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.StartMenuBtn), handler.StartMenuHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.PurchasesBtn), handler.PurchasesHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.RefillBalanceBtn), handler.RefillBalanceHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.ProfileRefreshBtn), handler.ProfileRefreshHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.ReplenishmentsBtn), handler.ReplenishmentsHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.ReferralBtn), handler.ReferralHandler)
	b.RegisterHandlerMatchFunc(matchAnyLang(texts.SettingsBtn), handler.SettingsHandler)

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
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.ReplenishmentDetailCallbackPrefix, bot.MatchTypePrefix, handler.ReplenishmentDetailHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.CheckPaymentCallbackPrefix, bot.MatchTypePrefix, handler.CheckPaymentHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.ReferralCloseCallback, bot.MatchTypeExact, handler.ReferralCloseHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.SettingsLanguageCallback, bot.MatchTypeExact, handler.SettingsLanguageHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, utils.LanguageCallbackPrefix, bot.MatchTypePrefix, handler.LanguageSetHandler)

	return &TelegramBot{
		log:                  log,
		bot:                  b,
		middlewares:          middlewares,
		webhookConfig:        webhookConfig,
		userService:          userService,
		productService:       productService,
		purchaseService:      purchaseService,
		categoryService:      categoryService,
		settingsService:      settingsService,
		replenishmentService: replenishmentService,
	}, nil
}

// Start блокируется до отмены ctx: вебхуком (см. RunWebhookServer в
// cmd/bot), если BotWebhookConfig.URL задан, иначе long-polling'ом.
func (bt *TelegramBot) Start(ctx context.Context) {
	bt.log.Info("starting bot")
	if bt.webhookConfig.URL != "" {
		bt.bot.StartWebhook(ctx)
		return
	}
	bt.bot.Start(ctx)
}

// WebhookHandler — HTTP-хендлер приёма апдейтов от Telegram (проверяет
// X-Telegram-Bot-Api-Secret-Token сам, см. bot.WithWebhookSecretToken выше).
// Используется только в режиме вебхука, см. cmd/bot/webhook_lifecycle.go.
func (bt *TelegramBot) WebhookHandler() http.HandlerFunc {
	return bt.bot.WebhookHandler()
}

// RegisterWebhook сообщает Telegram текущий URL. Идемпотентно — безопасно
// вызывать на каждом старте.
func (bt *TelegramBot) RegisterWebhook(ctx context.Context) error {
	_, err := bt.bot.SetWebhook(ctx, &bot.SetWebhookParams{
		URL:         bt.webhookConfig.URL,
		SecretToken: bt.webhookConfig.Secret,
	})
	return err
}

// WaitInFlight ждёт, пока догребут уже начатые update'ы. Отмена ctx у Start
// останавливает только поллинг: горутины обработки живут отдельно, и без этого
// ожидания их рвало закрытием Redis (см. middleware.Track).
func (bt *TelegramBot) WaitInFlight(ctx context.Context) error {
	return bt.middlewares.WaitInFlight(ctx)
}
