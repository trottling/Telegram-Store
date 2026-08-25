// Package paymentsbackend — HTTP API, принимающий только вебхуки платёжных
// мерчантов. Зависит только от internal/domain/service, как и admin_backend.
package paymentsbackend

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"github.com/trottling/Telegram-Store/internal/domain/service/payment"
	"github.com/trottling/Telegram-Store/payments_backend/handlers"
	"go.uber.org/zap"
)

type Server struct {
	httpServer *http.Server
	log        *zap.SugaredLogger
}

func New(
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	crystalPayProvider payment.Provider,
	cfg *config.PaymentsConfig,
	log *zap.SugaredLogger,
) *Server {
	h := handlers.New(settingsService, replenishmentService, crystalPayProvider, log)
	router := newRouter(h)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router,
		},
		log: log,
	}
}

func (s *Server) Start() error {
	s.log.Infow("payments_backend: starting webhooks server", "addr", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("payments_backend: shutting down webhooks server")
	return s.httpServer.Shutdown(ctx)
}
