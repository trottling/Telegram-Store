package redis

import (
	"fmt"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// товары

func activeProductsKey() string                  { return "products:active" }
func productKey(id models.ProductID) string      { return fmt.Sprintf("product:%s", id) }
func productCountKey(id models.ProductID) string { return fmt.Sprintf("product:%s:count", id) }

// пользователь

func userKey(telegramID models.TelegramID) string { return fmt.Sprintf("user:%s", telegramID) }

// категории

func categoryChildrenKey(parentID *models.CategoryID) string {
	if parentID == nil {
		return "category:children:root"
	}
	return fmt.Sprintf("category:children:%s", *parentID)
}

// настройки бота

func settingsKey() string { return "settings" }

// состояние FSM

func stateKey(telegramID models.TelegramID) string { return fmt.Sprintf("fsm:%s", telegramID) }

// логин-коды и сессии веб-панели

func adminLoginCodeKey(hash string) string { return fmt.Sprintf("admin_login_code:%s", hash) }

// adminLoginCodeOwnerKey — обратный индекс telegramID -> hash активного кода,
// нужен только чтобы погасить предыдущий код при выдаче нового.
func adminLoginCodeOwnerKey(telegramID models.TelegramID) string {
	return fmt.Sprintf("admin_login_code_owner:%s", telegramID)
}

func adminSessionKey(hash string) string { return fmt.Sprintf("admin_session:%s", hash) }

func adminExchangeAttemptsKey(key string) string {
	return fmt.Sprintf("admin_exchange_attempts:%s", key)
}

// кулдаун кнопки «Проверить оплату»

func replenishmentCheckCooldownKey(replenishmentID models.ReplenishmentID) string {
	return fmt.Sprintf("replenishment_check_cooldown:%s", replenishmentID)
}
