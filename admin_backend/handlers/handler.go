// Package handlers — HTTP-хендлеры admin API. Зависит только от
// internal/domain/service, не от repository напрямую. Каждая запись идёт
// через AdminService/UserService — так не теряются аудит-лог и инвалидация кэша.
package handlers

import (
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"go.uber.org/zap"
)

type Handlers struct {
	userService          service.UserService
	productService       service.ProductService
	categoryService      service.CategoryService
	purchaseService      service.PurchaseService
	adminService         service.AdminService
	settingsService      service.SettingsService
	replenishmentService service.ReplenishmentService
	adminAuthService     service.AdminAuthService

	log *zap.SugaredLogger
}

func New(
	userService service.UserService,
	productService service.ProductService,
	categoryService service.CategoryService,
	purchaseService service.PurchaseService,
	adminService service.AdminService,
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	adminAuthService service.AdminAuthService,
	log *zap.SugaredLogger,
) *Handlers {
	return &Handlers{
		userService:          userService,
		productService:       productService,
		categoryService:      categoryService,
		purchaseService:      purchaseService,
		adminService:         adminService,
		settingsService:      settingsService,
		replenishmentService: replenishmentService,
		adminAuthService:     adminAuthService,
		log:                  log,
	}
}
