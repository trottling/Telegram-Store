package texts

import (
	domain "github.com/trottling/Telegram-Store/internal/domain/models"
)

// MerchantName — человекочитаемое имя мерчанта для списка "Мои пополнения".
func MerchantName(lang string, merchant domain.Merchant) string {
	switch merchant {
	case domain.MerchantCrystalPay:
		return T(lang, CrystalPayBtn, nil)
	case domain.MerchantYooKassa:
		return T(lang, YooKassaBtn, nil)
	case domain.MerchantTinkoff:
		return T(lang, TinkoffBtn, nil)
	case domain.MerchantDummy:
		return T(lang, DummyBtn, nil)
	case domain.MerchantReferral:
		return T(lang, ReferralName, nil)
	default:
		return string(merchant)
	}
}

// ReplenishmentStatusName — человекочитаемый статус пополнения.
func ReplenishmentStatusName(lang string, status domain.ReplenishmentStatus) string {
	switch status {
	case domain.ReplenishmentStatusPending:
		return T(lang, ReplenishmentStatusPending, nil)
	case domain.ReplenishmentStatusPaid:
		return T(lang, ReplenishmentStatusPaid, nil)
	case domain.ReplenishmentStatusFailed:
		return T(lang, ReplenishmentStatusFailed, nil)
	case domain.ReplenishmentStatusCancelled:
		return T(lang, ReplenishmentStatusCancelled, nil)
	default:
		return string(status)
	}
}
