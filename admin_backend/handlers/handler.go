// Package handlers — HTTP-хендлеры admin API. Зависит только от
// internal/domain/service, не от repository напрямую. Каждая запись идёт
// через AdminService/UserService — так не теряются аудит-лог и инвалидация кэша.
package handlers

import (
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/domain/service"
)

type Handlers struct {
	userService          service.UserService
	productService       service.ProductService
	categoryService      service.CategoryService
	purchaseService      service.PurchaseService
	adminService         service.AdminService
	statsService         service.StatsService
	settingsService      service.SettingsService
	replenishmentService service.ReplenishmentService
	adminAuthService     service.AdminAuthService

	// cookieDomain — Domain сессионной cookie, см. Exchange/Logout.
	cookieDomain string
	log          *logrus.Logger
}

func New(
	userService service.UserService,
	productService service.ProductService,
	categoryService service.CategoryService,
	purchaseService service.PurchaseService,
	adminService service.AdminService,
	statsService service.StatsService,
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	adminAuthService service.AdminAuthService,
	cookieDomain string,
	log *logrus.Logger,
) *Handlers {
	return &Handlers{
		userService:          userService,
		productService:       productService,
		categoryService:      categoryService,
		purchaseService:      purchaseService,
		adminService:         adminService,
		statsService:         statsService,
		settingsService:      settingsService,
		replenishmentService: replenishmentService,
		adminAuthService:     adminAuthService,
		cookieDomain:         cookieDomain,
		log:                  log,
	}
}
