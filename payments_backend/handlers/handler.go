// Package handlers — HTTP-хендлеры payments_backend. Единственная забота —
// приём вебхуков мерчантов, поэтому сервисов всего два (в отличие от
// admin_backend/handlers, которому нужны все девять).
package handlers

import (
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/domain/service"
)

type Handlers struct {
	settingsService      service.SettingsService
	replenishmentService service.ReplenishmentService

	log *logrus.Logger
}

func New(
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	log *logrus.Logger,
) *Handlers {
	return &Handlers{
		settingsService:      settingsService,
		replenishmentService: replenishmentService,
		log:                  log,
	}
}
