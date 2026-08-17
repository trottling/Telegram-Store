// Package adminbackend — HTTP API админ-панели на gin, отдаёт JSON фронтенду.
// Зависит только от internal/domain/service. Вебхуки мерчантов сюда не
// входят — см. отдельный бинарник payments_backend.
package adminbackend

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/admin_backend/handlers"
	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/service"
)

type Server struct {
	httpServer *http.Server
	log        *logrus.Logger
}

func New(
	userService service.UserService,
	productService service.ProductService,
	categoryService service.CategoryService,
	purchaseService service.PurchaseService,
	adminService service.AdminService,
	statsService service.StatsService,
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	adminAuthService service.AdminAuthService,
	cfg *config.AdminPanelConfig,
	log *logrus.Logger,
) *Server {
	h := handlers.New(userService, productService, categoryService, purchaseService, adminService, statsService, settingsService, replenishmentService, adminAuthService, log)
	router := newRouter(h, adminAuthService, cfg.CORSOrigin, log)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router,
		},
		log: log,
	}
}

// Start блокируется, пока сервер не остановят через Shutdown.
func (s *Server) Start() error {
	s.log.WithField("addr", s.httpServer.Addr).Info("admin_backend: starting admin API server")
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("admin_backend: shutting down admin API server")
	return s.httpServer.Shutdown(ctx)
}
