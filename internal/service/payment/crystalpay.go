package payment

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"time"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

const crystalPayBaseURL = "https://api.crystalpay.io/v3/"

const (
	crystalPayInvoiceLifetimeMinutes = 60
	// type=purchase — оплатить нужно ровно amount, без авто-пересчёта суммы
	// (в отличие от type=topup, который принимает любую сумму больше/меньше).
	crystalPayInvoiceType = "purchase"
)

// CrystalPayProvider — прямой HTTP-клиент CrystalPay API v3: готового Go SDK
// у CrystalPay нет. Креды читаются из Settings на каждый вызов — админ
// меняет их в панели без перезапуска бота.
type CrystalPayProvider struct {
	settingsService service.SettingsService
	backendURL      string
	httpClient      *http.Client
}

func NewCrystalPayProvider(settingsService service.SettingsService, backendURL string) *CrystalPayProvider {
	return &CrystalPayProvider{
		settingsService: settingsService,
		backendURL:      backendURL,
		httpClient:      &http.Client{Timeout: 15 * time.Second},
	}
}

type crystalPayCreateRequest struct {
	AuthLogin   string  `json:"auth_login"`
	AuthSecret  string  `json:"auth_secret"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Lifetime    int     `json:"lifetime"`
	Description string  `json:"description,omitempty"`
	CallbackURL string  `json:"callback_url,omitempty"`
}

type crystalPayCreateResponse struct {
	Error  bool     `json:"error"`
	Errors []string `json:"errors"`
	ID     string   `json:"id"`
	URL    string   `json:"url"`
}

type crystalPayInfoRequest struct {
	AuthLogin  string `json:"auth_login"`
	AuthSecret string `json:"auth_secret"`
	ID         string `json:"id"`
}

type crystalPayInfoResponse struct {
	Error  bool     `json:"error"`
	Errors []string `json:"errors"`
	State  string   `json:"state"`
}

func (p *CrystalPayProvider) CreateInvoice(ctx context.Context, _ models.TelegramID, amount models.Money, description string) (string, string, error) {
	settings, err := p.settingsService.Get(ctx)
	if err != nil {
		return "", "", err
	}
	cfg := settings.CrystalPay
	if !cfg.Enabled {
		return "", "", domainerrors.ErrMerchantDisabled
	}
	if amount.LessThan(cfg.MinAmount) || (!cfg.MaxAmount.IsZero() && amount.GreaterThan(cfg.MaxAmount)) {
		return "", "", domainerrors.ErrAmountOutOfRange
	}

	req := crystalPayCreateRequest{
		AuthLogin:   cfg.Login,
		AuthSecret:  cfg.Secret,
		Amount:      amount.Float64(),
		Type:        crystalPayInvoiceType,
		Lifetime:    crystalPayInvoiceLifetimeMinutes,
		Description: description,
		CallbackURL: p.backendURL + "/api/webhooks/crystalpay",
	}

	var resp crystalPayCreateResponse
	if err = p.do(ctx, "invoice/create/", req, &resp); err != nil {
		return "", "", err
	}
	if resp.Error {
		return "", "", fmt.Errorf("crystalpay: %v", resp.Errors)
	}
	// Без счёта и ссылки платить нечем: пустой invoice_id ещё и не даст потом
	// сопоставить вебхук с этой строкой.
	if resp.ID == "" || resp.URL == "" {
		return "", "", fmt.Errorf("crystalpay: ответ без id или url счёта")
	}

	return resp.URL, resp.ID, nil
}

func (p *CrystalPayProvider) CheckStatus(ctx context.Context, invoiceID string) (domainpayment.PaymentStatus, error) {
	settings, err := p.settingsService.Get(ctx)
	if err != nil {
		return "", err
	}
	cfg := settings.CrystalPay

	req := crystalPayInfoRequest{AuthLogin: cfg.Login, AuthSecret: cfg.Secret, ID: invoiceID}

	var resp crystalPayInfoResponse
	if err = p.do(ctx, "invoice/info/", req, &resp); err != nil {
		return "", err
	}
	if resp.Error {
		return "", fmt.Errorf("crystalpay: %v", resp.Errors)
	}

	return crystalPayStatus(resp.State), nil
}

// crystalPayStatus — payed/unavailable/failed финальны, остальные — в процессе.
func crystalPayStatus(state string) domainpayment.PaymentStatus {
	switch state {
	case "payed":
		return domainpayment.PaymentStatusPaid
	case "failed", "unavailable":
		return domainpayment.PaymentStatusFailed
	default: // created, notpayed, processing, wrongamount
		return domainpayment.PaymentStatusPending
	}
}

func (p *CrystalPayProvider) do(ctx context.Context, endpoint string, body, dest any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, crystalPayBaseURL+endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Статус проверяем до разбора тела: ответ шлюза или WAF вполне может быть
	// валидным JSON без поля error, и тогда без этой проверки вызов выглядел
	// бы успешным с пустыми полями в ответе.
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("crystalpay: %s ответил %s", endpoint, resp.Status)
	}

	return json.UnmarshalDecode(jsontext.NewDecoder(resp.Body), dest)
}
