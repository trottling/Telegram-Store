// Package admintoken — одноразовые коды входа и JWT-сессии веб-панели.
// Чистая криптография, без доменных зависимостей.
package admintoken

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// loginCodeDigits — длина одноразового кода из /admin.
const loginCodeDigits = 6

// AdminClaims — payload сессионного JWT. IsAdmin намеренно не хранится в
// токене — ValidateSession всегда перепроверяет права в Postgres.
type AdminClaims struct {
	TelegramID int64 `json:"telegram_id"`
	jwt.RegisteredClaims
}

// GenerateCode — пара (код, его хеш для Store), crypto/rand.
func GenerateCode(key []byte) (plaintext string, hash string, err error) {
	m := big.NewInt(1)
	for range loginCodeDigits {
		m.Mul(m, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, m)
	if err != nil {
		return "", "", err
	}
	plaintext = fmt.Sprintf("%0*d", loginCodeDigits, n.Int64())
	return plaintext, Hash(plaintext, key), nil
}

// GenerateSessionJWT подписывает токен на ttl (HS256). Подпись даёт
// целостность и срок действия, но отзыв — через domain/adminsession.Store
// (см. AdminAuthSrv.ValidateSession), сам JWT нельзя аннулировать.
func GenerateSessionJWT(telegramID int64, ttl time.Duration, secret []byte) (string, error) {
	now := time.Now()
	claims := AdminClaims{
		TelegramID: telegramID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseSessionJWT проверяет подпись и срок, не трогает Redis/Postgres.
func ParseSessionJWT(tokenString string, secret []byte) (*AdminClaims, error) {
	claims := &AdminClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		// Только HMAC — защита от подмены алгоритма (например, на "none").
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("admintoken: unexpected signing method %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("admintoken: invalid session token")
	}
	return claims, nil
}

// Hash — общий способ хешировать код/токен перед поиском в Store.
//
// HMAC, а не голый sha256: для сессионного токена разницы нет (энтропии в нём
// достаточно), но код входа шестизначный, и радужная таблица на миллион
// значений строится мгновенно. Без ключа доступ на чтение к Redis давал бы
// действующий код входа; с ключом хеш бесполезен без ADMIN_JWT_SECRET.
func Hash(plaintext string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(plaintext))
	return hex.EncodeToString(mac.Sum(nil))
}
