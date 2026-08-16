// Package dto — тела запросов и пагинация admin API. Ответы в основном
// переиспользуют типы из internal/domain/models напрямую.
package dto

import domainmodels "github.com/trottling/Telegram-Store/internal/domain/models"

// Paginated — обёртка, в которой отвечают все list-эндпоинты.
type Paginated[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
}

type CreateCategoryRequest struct {
	ParentID    *int64 `json:"parent_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateCategoryRequest struct {
	ParentID    *int64 `json:"parent_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateProductRequest struct {
	CategoryID  *int64  `json:"category_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type UpdateProductRequest struct {
	CategoryID  *int64  `json:"category_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	IsActive    bool    `json:"is_active"`
}

type AddItemsRequest struct {
	Contents []string `json:"contents"`
}

type UpdateBalanceRequest struct {
	Amount float64 `json:"amount"` // может быть отрицательной — списание
}

type UpdateSettingsRequest struct {
	SupportUsername string                          `json:"support_username"`
	CrystalPay      domainmodels.CrystalPaySettings `json:"crystalpay"`
	YooKassa        domainmodels.YooKassaSettings   `json:"yookassa"`
	Tinkoff         domainmodels.TinkoffSettings    `json:"tinkoff"`
	Referral        domainmodels.ReferralSettings   `json:"referral"`
}

// ExchangeCodeRequest — тело POST /api/auth/exchange, одноразовый код от бота.
type ExchangeCodeRequest struct {
	Code string `json:"code"`
}

// TokenResponse — сессионный токен, выдаётся в обмен на код.
type TokenResponse struct {
	Token string `json:"token"`
}

// ReferralCountResponse — GET /api/users/:telegram_id/referrals.
type ReferralCountResponse struct {
	Count int64 `json:"count"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
