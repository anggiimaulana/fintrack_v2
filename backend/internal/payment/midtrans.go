package payment

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MidtransConfig configures the Midtrans Snap gateway.
type MidtransConfig struct {
	ServerKey    string
	ClientKey    string
	IsProduction bool
	// SnapBaseURL overrides the Snap API base (default: sandbox/production).
	SnapBaseURL string
	// CoreBaseURL overrides the Core API base used for status pulls.
	CoreBaseURL string
	// Timeout for HTTP calls (default 15s).
	Timeout time.Duration
}

// MidtransGateway implements Gateway using Midtrans Snap (hosted checkout).
// Card data never touches this server (PCI-DSS SAQ-A).
type MidtransGateway struct {
	cfg    MidtransConfig
	client *http.Client
}

// NewMidtransGateway builds the gateway with sane defaults.
func NewMidtransGateway(cfg MidtransConfig) *MidtransGateway {
	if cfg.SnapBaseURL == "" {
		if cfg.IsProduction {
			cfg.SnapBaseURL = "https://app.midtrans.com/snap/v1"
		} else {
			cfg.SnapBaseURL = "https://app.sandbox.midtrans.com/snap/v1"
		}
	}
	if cfg.CoreBaseURL == "" {
		if cfg.IsProduction {
			cfg.CoreBaseURL = "https://api.midtrans.com/v2"
		} else {
			cfg.CoreBaseURL = "https://api.sandbox.midtrans.com/v2"
		}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	return &MidtransGateway{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
}

// Name returns the gateway identifier.
func (g *MidtransGateway) Name() string { return "midtrans" }

// basicAuth builds the Midtrans Basic auth header (server_key + ":").
func (g *MidtransGateway) basicAuth() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(g.cfg.ServerKey+":"))
}

// CreateTransaction creates a Snap transaction and returns its token/redirect.
func (g *MidtransGateway) CreateTransaction(ctx context.Context, req CreateTxRequest) (*CreateTxResponse, error) {
	if g.cfg.ServerKey == "" {
		return nil, ErrGatewayNotConfigured
	}

	payload := map[string]any{
		"transaction_details": map[string]any{
			"order_id":     req.OrderID,
			"gross_amount": req.AmountIDR, // integer Rupiah
		},
		"item_details": []map[string]any{{
			"id":       req.OrderID,
			"price":    req.AmountIDR,
			"quantity": 1,
			"name":     truncate(req.Description, 50),
		}},
	}
	if req.CustomerName != "" || req.CustomerEmail != "" {
		payload["customer_details"] = map[string]any{
			"first_name": req.CustomerName,
			"email":      req.CustomerEmail,
		}
	}
	if req.ExpiryMinutes > 0 {
		payload["expiry"] = map[string]any{"unit": "minutes", "duration": req.ExpiryMinutes}
	}
	if req.SaveCard && req.UserID != uuid.Nil {
		payload["credit_card"] = map[string]any{
			"save_card": true,
		}
		payload["user_id"] = req.UserID.String()
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.SnapBaseURL+"/transactions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", g.basicAuth())

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans snap returned %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Token       string `json:"token"`
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}

	return &CreateTxResponse{
		GatewayRef:  req.OrderID,
		SnapToken:   parsed.Token,
		RedirectURL: parsed.RedirectURL,
		Status:      PaymentPending,
		Raw:         map[string]any{"token": parsed.Token, "redirect_url": parsed.RedirectURL},
	}, nil
}

// VerifyWebhook validates the SHA-512 signature and parses the notification.
//
// signature = SHA512(order_id + status_code + gross_amount + server_key)
func (g *MidtransGateway) VerifyWebhook(_ context.Context, payload []byte, _ map[string]string) (*WebhookResult, error) {
	var n struct {
		OrderID               string `json:"order_id"`
		StatusCode            string `json:"status_code"`
		GrossAmount           string `json:"gross_amount"`
		SignatureKey          string `json:"signature_key"`
		TransactionStatus     string `json:"transaction_status"`
		PaymentType           string `json:"payment_type"`
		SavedTokenID          string `json:"saved_token_id"`
		SavedTokenIDExpiredAt string `json:"saved_token_id_expired_at"`
		MaskedCard            string `json:"masked_card"`
		CardType              string `json:"card_type"`
		Bank                  string `json:"bank"`
	}
	if err := json.Unmarshal(payload, &n); err != nil {
		return nil, fmt.Errorf("invalid webhook payload: %w", err)
	}

	expected := midtransSignature(n.OrderID, n.StatusCode, n.GrossAmount, g.cfg.ServerKey)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(n.SignatureKey)) != 1 {
		return nil, ErrInvalidSignature
	}

	amount := parseAmount(n.GrossAmount)
	return &WebhookResult{
		OrderID:               n.OrderID,
		GatewayRef:            n.OrderID,
		Status:                mapMidtransStatus(n.TransactionStatus),
		AmountIDR:             amount,
		PaymentMethod:         n.PaymentType,
		SavedTokenID:          n.SavedTokenID,
		SavedTokenIDExpiredAt: n.SavedTokenIDExpiredAt,
		MaskedCard:            n.MaskedCard,
		CardType:              n.CardType,
		Bank:                  n.Bank,
		Raw:                   map[string]any{"transaction_status": n.TransactionStatus, "payment_type": n.PaymentType},
	}, nil
}

// CheckStatus pulls the authoritative transaction status from Midtrans.
func (g *MidtransGateway) CheckStatus(ctx context.Context, gatewayRef string) (*StatusResult, error) {
	if g.cfg.ServerKey == "" {
		return nil, ErrGatewayNotConfigured
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, g.cfg.CoreBaseURL+"/"+gatewayRef+"/status", nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", g.basicAuth())

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans status returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		OrderID               string `json:"order_id"`
		TransactionStatus     string `json:"transaction_status"`
		GrossAmount           string `json:"gross_amount"`
		SavedTokenID          string `json:"saved_token_id"`
		SavedTokenIDExpiredAt string `json:"saved_token_id_expired_at"`
		MaskedCard            string `json:"masked_card"`
		CardType              string `json:"card_type"`
		Bank                  string `json:"bank"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	return &StatusResult{
		OrderID:               parsed.OrderID,
		Status:                mapMidtransStatus(parsed.TransactionStatus),
		AmountIDR:             parseAmount(parsed.GrossAmount),
		SavedTokenID:          parsed.SavedTokenID,
		SavedTokenIDExpiredAt: parsed.SavedTokenIDExpiredAt,
		MaskedCard:            parsed.MaskedCard,
		CardType:              parsed.CardType,
		Bank:                  parsed.Bank,
		Raw:                   map[string]any{"transaction_status": parsed.TransactionStatus},
	}, nil
}

// CancelTransaction voids/cancels a pending transaction.
func (g *MidtransGateway) CancelTransaction(ctx context.Context, gatewayRef string) error {
	if g.cfg.ServerKey == "" {
		return ErrGatewayNotConfigured
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.CoreBaseURL+"/"+gatewayRef+"/cancel", nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", g.basicAuth())
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("midtrans cancel returned %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// ─── helpers ──────────────────────────────────────────────────

// midtransSignature computes SHA512(order_id + status_code + gross_amount + server_key).
func midtransSignature(orderID, statusCode, grossAmount, serverKey string) string {
	h := sha512.New()
	h.Write([]byte(orderID + statusCode + grossAmount + serverKey))
	return hex.EncodeToString(h.Sum(nil))
}

// mapMidtransStatus maps Midtrans transaction_status to our PaymentStatus.
func mapMidtransStatus(s string) PaymentStatus {
	switch strings.ToLower(s) {
	case "settlement", "capture":
		return PaymentSettled
	case "pending":
		return PaymentPending
	case "deny", "cancel", "failure":
		return PaymentFailed
	case "expire":
		return PaymentExpired
	case "refund", "partial_refund":
		return PaymentRefunded
	default:
		return PaymentPending
	}
}

// parseAmount parses a gross_amount string ("12345.00") into integer Rupiah.
func parseAmount(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i]
	}
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			continue
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ChargeSavedCard charges a previously tokenized card via the Midtrans Core API.
// Card numbers never touch this application: only the token_id is submitted.
func (g *MidtransGateway) ChargeSavedCard(ctx context.Context, req ChargeSavedCardRequest) (*ChargeCardResponse, error) {
	if g.cfg.ServerKey == "" {
		return nil, ErrGatewayNotConfigured
	}
	if req.TokenID == "" {
		return nil, ErrCardTokenNotFound
	}

	payload := map[string]any{
		"payment_type": "credit_card",
		"transaction_details": map[string]any{
			"order_id":     req.OrderID,
			"gross_amount": req.AmountIDR,
		},
		"credit_card": map[string]any{
			"token_id": req.TokenID,
			"user_id":  req.UserID.String(),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.CoreBaseURL+"/charge", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", g.basicAuth())

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans charge returned %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		StatusCode        string `json:"status_code"`
		TransactionStatus string `json:"transaction_status"`
		FraudStatus       string `json:"fraud_status"`
		OrderID           string `json:"order_id"`
		PaymentType       string `json:"payment_type"`
		MaskedCard        string `json:"masked_card"`
		Bank              string `json:"bank"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}

	status := mapMidtransStatus(parsed.TransactionStatus)
	// If marked challenge, do not settle automatically without manual 3DS check
	if parsed.FraudStatus == "challenge" {
		status = PaymentPending
	}

	return &ChargeCardResponse{
		GatewayRef:    parsed.OrderID,
		Status:        status,
		PaymentMethod: parsed.PaymentType,
		MaskedCard:    parsed.MaskedCard,
		Bank:          parsed.Bank,
		Raw: map[string]any{
			"status_code":        parsed.StatusCode,
			"transaction_status": parsed.TransactionStatus,
			"fraud_status":       parsed.FraudStatus,
		},
	}, nil
}

// compile-time assertions.
var (
	_ Gateway     = (*MidtransGateway)(nil)
	_ CardCharger = (*MidtransGateway)(nil)
)
