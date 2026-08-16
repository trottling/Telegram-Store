package redis

import "fmt"

// товары

func activeProductsKey() string       { return "products:active" }
func productKey(id int64) string      { return fmt.Sprintf("product:%d", id) }
func productCountKey(id int64) string { return fmt.Sprintf("product:%d:count", id) }

// пользователь

func userKey(telegramID int64) string { return fmt.Sprintf("user:%d", telegramID) }

// категории

func categoryChildrenKey(parentID *int64) string {
	if parentID == nil {
		return "category:children:root"
	}
	return fmt.Sprintf("category:children:%d", *parentID)
}

// настройки бота

func settingsKey() string { return "settings" }

// состояние FSM

func stateKey(telegramID int64) string { return fmt.Sprintf("fsm:%d", telegramID) }

// логин-коды и сессии веб-панели

func adminLoginCodeKey(hash string) string { return fmt.Sprintf("admin_login_code:%s", hash) }
func adminSessionKey(hash string) string   { return fmt.Sprintf("admin_session:%s", hash) }
