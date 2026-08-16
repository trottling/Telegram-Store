package utils

import (
	"github.com/trottling/Telegram-Store/bot/texts"
	domain "github.com/trottling/Telegram-Store/internal/domain/models"
)

// MerchantName — человекочитаемое имя мерчанта для списка "Мои пополнения".
func MerchantName(merchant domain.Merchant) string {
	switch merchant {
	case domain.MerchantCrystalPay:
		return texts.CrystalPayBtn
	case domain.MerchantYooKassa:
		return texts.YooKassaBtn
	case domain.MerchantTinkoff:
		return texts.TinkoffBtn
	case domain.MerchantReferral:
		return texts.ReferralName
	default:
		return string(merchant)
	}
}

// ReplenishmentStatusName — человекочитаемый статус пополнения.
func ReplenishmentStatusName(status domain.ReplenishmentStatus) string {
	switch status {
	case domain.ReplenishmentStatusPending:
		return texts.ReplenishmentStatusPending
	case domain.ReplenishmentStatusPaid:
		return texts.ReplenishmentStatusPaid
	case domain.ReplenishmentStatusFailed:
		return texts.ReplenishmentStatusFailed
	case domain.ReplenishmentStatusCancelled:
		return texts.ReplenishmentStatusCancelled
	default:
		return string(status)
	}
}
