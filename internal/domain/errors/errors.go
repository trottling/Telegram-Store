package errors

import "errors"

var (
	// пользователи
	ErrUserNotFound = errors.New("user not found")

	// история покупок
	ErrPurchaseNotFound = errors.New("purchase not found")

	// дерево каталога
	ErrCategoryNotFound = errors.New("category not found")

	// настройки бота
	ErrSettingsNotFound     = errors.New("settings not found")
	ErrInvalidSettingsInput = errors.New("invalid settings input")

	// пополнения баланса
	ErrReplenishmentNotFound = errors.New("replenishment not found")
	ErrInvalidMerchant       = errors.New("unknown or unavailable merchant")
	ErrMerchantDisabled      = errors.New("merchant is disabled")
	ErrAmountOutOfRange      = errors.New("amount is out of the merchant's allowed range")

	// сценарий покупки
	ErrProductNotFound   = errors.New("product not found")
	ErrProductInactive   = errors.New("product is not active")
	ErrProductOutOfStock = errors.New("product out of stock")
	ErrNotEnoughBalance  = errors.New("not enough balance")
	ErrInvalidQuantity   = errors.New("quantity must be positive")
	ErrTooManyProducts   = errors.New("too many products requested at once")

	// админ-действия
	ErrAlreadyAdmin        = errors.New("user is already an admin")
	ErrInvalidAmount       = errors.New("amount must not be zero")
	ErrInvalidProductInput = errors.New("invalid product name or price")
	ErrNoItemsProvided     = errors.New("no items provided")

	// веб-панель
	ErrNotAdmin                = errors.New("user is not an admin")
	ErrCannotRevokeRootAdmin   = errors.New("cannot revoke root admin privileges")
	ErrCannotRevokeSelf        = errors.New("cannot revoke your own admin rights")
	ErrOnlyRootAdminCanPromote = errors.New("only the root admin can grant admin rights")
	ErrCannotBanRootAdmin      = errors.New("cannot ban the root admin")
	ErrCannotBanSelf           = errors.New("cannot ban your own account")
	ErrInvalidToken            = errors.New("invalid or expired session")
	ErrInvalidInitData         = errors.New("invalid or expired init data")
	ErrTooManyAttempts         = errors.New("too many attempts")
	ErrCategoryNotEmpty        = errors.New("category still has child categories or products")
	ErrProductHasPurchases     = errors.New("product has purchase history and cannot be deleted")
)
