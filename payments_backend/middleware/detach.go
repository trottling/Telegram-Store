// Package middleware — обработчики, общие для всех трёх вебхуков.
package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// webhookTimeout — потолок на всю обработку вебхука после отцепления ctx.
const webhookTimeout = 30 * time.Second

// Detach отвязывает ctx обработчика от разрыва соединения мерчантом.
//
// gin отменяет ctx запроса, как только клиент отключился, а платёжные
// мерчанты рвут соединение по своему таймауту (обычно единицы секунд). Без
// отцепления такой обрыв мог прийти между коммитом транзакции в Confirm и
// последующим сбросом кэша баланса — пользователь остался бы со старым
// балансом до истечения TTL, хотя деньги уже зачислены.
//
// Значения ctx сохраняются, отменяемость — нет; вместо неё собственный
// дедлайн, чтобы зависший вебхук не жил вечно.
func Detach() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), webhookTimeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
