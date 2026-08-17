// Package paymentsbackend — HTTP API, принимающий только вебхуки платёжных
// мерчантов. Зависит только от internal/domain/service, как и admin_backend.
package paymentsbackend

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"github.com/trottling/Telegram-Store/payments_backend/handlers"
)

type Server struct {
	httpServer *http.Server
	log        *logrus.Logger
}

func New(
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	cfg *config.PaymentsConfig,
	log *logrus.Logger,
) *Server {
	h := handlers.New(settingsService, replenishmentService, log)
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
	s.log.WithField("addr", s.httpServer.Addr).Info("payments_backend: starting webhooks server")
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
