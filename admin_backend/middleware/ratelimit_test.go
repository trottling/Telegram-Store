package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// fakeAuthService считает попытки по ключу и режет после limit — та же
// семантика, что у AdminAuthSrv.AllowExchangeAttempt, но без Redis. Здесь
// проверяется не она, а то, какой ключ до неё доходит.
type fakeAuthService struct {
	mu       sync.Mutex
	limit    int
	attempts map[string]int
	keys     []string
}

func (s *fakeAuthService) AllowExchangeAttempt(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts[key]++
	s.keys = append(s.keys, key)
	return s.attempts[key] <= s.limit, nil
}

func (s *fakeAuthService) IssueLoginCode(context.Context, int64) (string, error) {
	panic("не используется")
}
func (s *fakeAuthService) ExchangeLoginCode(context.Context, string) (string, *models.User, error) {
	panic("не используется")
}
func (s *fakeAuthService) ValidateSession(context.Context, string) (*models.User, error) {
	panic("не используется")
}
func (s *fakeAuthService) Logout(context.Context, string) error {
	panic("не используется")
}

// newTestEngine повторяет то, что делает newRouter: доверенные прокси плюс
// лимитер на единственном неаутентифицированном роуте.
func newTestEngine(auth *fakeAuthService, trustedProxies string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logrus.New()
	log.SetOutput(io.Discard)

	r := gin.New()
	if trustedProxies != "" {
		_ = r.SetTrustedProxies(strings.Split(trustedProxies, ","))
	} else {
		_ = r.SetTrustedProxies(nil)
	}
	r.POST("/api/auth/exchange", RateLimitExchange(auth, log), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func post(r *gin.Engine, remoteAddr, forwardedFor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/exchange", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestRateLimitExchangeBlocksAfterLimit — исчерпанный лимит даёт 429, а не
// молчаливый пропуск дальше к обмену кода.
func TestRateLimitExchangeBlocksAfterLimit(t *testing.T) {
	auth := &fakeAuthService{limit: 10, attempts: map[string]int{}}
	r := newTestEngine(auth, "")

	for i := 1; i <= 10; i++ {
		if w := post(r, "10.0.0.1:1234", ""); w.Code != http.StatusOK {
			t.Fatalf("попытка %d: код %d, ожидался 200", i, w.Code)
		}
	}

	w := post(r, "10.0.0.1:1234", "")
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("попытка 11: код %d, ожидался 429", w.Code)
	}
}

// TestRateLimitExchangeIgnoresUntrustedForwardedFor — главное свойство: пока
// прокси не объявлен доверенным, подменённый X-Forwarded-For не должен давать
// новый счётчик. Именно из-за дефолта gin (доверять всем) лимит раньше
// обходился одним заголовком.
func TestRateLimitExchangeIgnoresUntrustedForwardedFor(t *testing.T) {
	auth := &fakeAuthService{limit: 1000, attempts: map[string]int{}}
	r := newTestEngine(auth, "")

	post(r, "203.0.113.7:1234", "1.1.1.1")
	post(r, "203.0.113.7:1234", "2.2.2.2")

	auth.mu.Lock()
	defer auth.mu.Unlock()
	for _, key := range auth.keys {
		if key != "203.0.113.7" {
			t.Fatalf("ключ лимита = %q, ожидался реальный адрес 203.0.113.7: подменённый заголовок не должен влиять", key)
		}
	}
	if got := auth.attempts["203.0.113.7"]; got != 2 {
		t.Errorf("попыток по реальному адресу = %d, ожидалось 2 (оба запроса в один счётчик)", got)
	}
}

// TestRateLimitExchangeIgnoresSpoofedEntryBehindProxy — собственно прод-сценарий
// подбора: атакующий сам присылает X-Forwarded-For, а caddy дописывает его
// реальный адрес справа. Считать нужно по реальному, иначе каждый запрос с новым
// поддельным значением получал бы свой счётчик и лимит ничего не стоил.
func TestRateLimitExchangeIgnoresSpoofedEntryBehindProxy(t *testing.T) {
	auth := &fakeAuthService{limit: 1000, attempts: map[string]int{}}
	r := newTestEngine(auth, "172.28.0.0/16")

	// Слева — подделка клиента, справа — то, что дописал caddy.
	post(r, "172.28.0.5:1234", "1.1.1.1, 198.51.100.99")
	post(r, "172.28.0.5:1234", "2.2.2.2, 198.51.100.99")

	auth.mu.Lock()
	defer auth.mu.Unlock()
	if got := auth.attempts["198.51.100.99"]; got != 2 {
		t.Errorf("попыток по реальному адресу = %d, ожидалось 2: подделка слева не должна давать новый счётчик (все ключи: %v)", got, auth.attempts)
	}
	if len(auth.attempts) != 1 {
		t.Errorf("счётчиков = %d, ожидался 1: поддельные значения не должны создавать свои (%v)", len(auth.attempts), auth.attempts)
	}
}

// TestRateLimitExchangeUsesForwardedForFromTrustedProxy — обратная сторона: за
// caddy настоящий адрес клиента приходит только в заголовке, и по нему-то и
// нужно считать, иначе все запросы схлопнутся в один счётчик прокси.
func TestRateLimitExchangeUsesForwardedForFromTrustedProxy(t *testing.T) {
	auth := &fakeAuthService{limit: 1000, attempts: map[string]int{}}
	r := newTestEngine(auth, "172.28.0.0/16")

	post(r, "172.28.0.5:1234", "198.51.100.10")
	post(r, "172.28.0.5:1234", "198.51.100.11")

	auth.mu.Lock()
	defer auth.mu.Unlock()
	if auth.attempts["198.51.100.10"] != 1 || auth.attempts["198.51.100.11"] != 1 {
		t.Errorf("попытки по IP клиентов = %v, ожидалось по одной на каждый", auth.attempts)
	}
}
