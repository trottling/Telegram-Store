// Package handlers — HTTP-хендлеры payments_backend. Единственная забота —
// приём вебхуков мерчантов, поэтому сервисов всего два (в отличие от
// admin_backend/handlers, которому нужны все девять).
package handlers

import (
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	"github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

type Handlers struct {
	settingsService      service.SettingsService
	replenishmentService service.ReplenishmentService
	// crystalPayProvider нужен только чтобы перепроверить статус счёта:
	// подпись CrystalPay покрывает лишь id, а state в том же теле не подписан.
	// Провайдер именно этого мерчанта, а не map — у двух других статус
	// проверять не нужно (Tinkoff подписывает всё тело, YooKassa
	// перезапрашивается своим SDK прямо в хендлере).
	crystalPayProvider payment.PaymentProvider

	log *logrus.Logger
}

func New(
	settingsService service.SettingsService,
	replenishmentService service.ReplenishmentService,
	crystalPayProvider payment.PaymentProvider,
	log *logrus.Logger,
) *Handlers {
	return &Handlers{
		settingsService:      settingsService,
		replenishmentService: replenishmentService,
		crystalPayProvider:   crystalPayProvider,
		log:                  log,
	}
}
