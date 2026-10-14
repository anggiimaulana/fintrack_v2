package payment

import (
	"context"

	"github.com/fintrack/backend/internal/runtimeconfig"
)

// Settings keys for the Midtrans gateway. A DB value overrides the env default.
const (
	KeyMidtransEnabled      = "midtrans_enabled"
	KeyMidtransServerKey    = "midtrans_server_key"
	KeyMidtransClientKey    = "midtrans_client_key"
	KeyMidtransIsProduction = "midtrans_is_production"

	KeyManualBankName      = "manual_bank_name"
	KeyManualAccountNumber = "manual_account_number"
	KeyManualAccountHolder = "manual_account_holder"
)

// RuntimeSettings resolves operational config with DB-over-env precedence.
type RuntimeSettings interface {
	String(ctx context.Context, settingKey, envKey, def string) string
	Bool(ctx context.Context, settingKey, envKey string, def bool) bool
	Secret(ctx context.Context, settingKey, envKey string) string
}

var _ RuntimeSettings = (*runtimeconfig.Resolver)(nil)

// DynamicMidtransGateway resolves its configuration on every call, so a Super
// Admin change to the Midtrans credentials takes effect without a redeploy. It
// delegates to a freshly built MidtransGateway, which is cheap (no I/O in the
// constructor) and keeps the request-scoped config consistent.
type DynamicMidtransGateway struct {
	settings RuntimeSettings
}

func NewDynamicMidtransGateway(settings RuntimeSettings) *DynamicMidtransGateway {
	return &DynamicMidtransGateway{settings: settings}
}

func (g *DynamicMidtransGateway) Name() string { return "midtrans" }

// config resolves the effective Midtrans configuration for this call.
func (g *DynamicMidtransGateway) config(ctx context.Context) MidtransConfig {
	return MidtransConfig{
		ServerKey:    g.settings.Secret(ctx, KeyMidtransServerKey, "MIDTRANS_SERVER_KEY"),
		ClientKey:    g.settings.String(ctx, KeyMidtransClientKey, "MIDTRANS_CLIENT_KEY", ""),
		IsProduction: g.settings.Bool(ctx, KeyMidtransIsProduction, "MIDTRANS_IS_PRODUCTION", false),
	}
}

// Enabled reports whether Midtrans is switched on and has a server key. The
// registry uses this to fall back to manual when Midtrans is not usable.
func (g *DynamicMidtransGateway) Enabled(ctx context.Context) bool {
	cfg := g.config(ctx)
	return g.settings.Bool(ctx, KeyMidtransEnabled, "MIDTRANS_ENABLED", false) && cfg.ServerKey != ""
}

func (g *DynamicMidtransGateway) CreateTransaction(ctx context.Context, req CreateTxRequest) (*CreateTxResponse, error) {
	return NewMidtransGateway(g.config(ctx)).CreateTransaction(ctx, req)
}

func (g *DynamicMidtransGateway) VerifyWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	return NewMidtransGateway(g.config(ctx)).VerifyWebhook(ctx, payload, headers)
}

func (g *DynamicMidtransGateway) CheckStatus(ctx context.Context, gatewayRef string) (*StatusResult, error) {
	return NewMidtransGateway(g.config(ctx)).CheckStatus(ctx, gatewayRef)
}

func (g *DynamicMidtransGateway) CancelTransaction(ctx context.Context, gatewayRef string) error {
	return NewMidtransGateway(g.config(ctx)).CancelTransaction(ctx, gatewayRef)
}

// DynamicManualGateway resolves the bank-transfer instructions on every call so
// an operator can correct the account details without a restart.
type DynamicManualGateway struct {
	settings RuntimeSettings
}

func NewDynamicManualGateway(settings RuntimeSettings) *DynamicManualGateway {
	return &DynamicManualGateway{settings: settings}
}

func (g *DynamicManualGateway) Name() string { return "manual" }

func (g *DynamicManualGateway) CreateTransaction(ctx context.Context, req CreateTxRequest) (*CreateTxResponse, error) {
	return NewManualGateway(
		g.settings.String(ctx, KeyManualBankName, "MANUAL_BANK_NAME", ""),
		g.settings.String(ctx, KeyManualAccountNumber, "MANUAL_ACCOUNT_NUMBER", ""),
		g.settings.String(ctx, KeyManualAccountHolder, "MANUAL_ACCOUNT_HOLDER", ""),
	).CreateTransaction(ctx, req)
}

func (g *DynamicManualGateway) VerifyWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error) {
	return NewManualGateway("", "", "").VerifyWebhook(ctx, payload, headers)
}

func (g *DynamicManualGateway) CheckStatus(ctx context.Context, gatewayRef string) (*StatusResult, error) {
	return NewManualGateway("", "", "").CheckStatus(ctx, gatewayRef)
}

func (g *DynamicManualGateway) CancelTransaction(ctx context.Context, gatewayRef string) error {
	return NewManualGateway("", "", "").CancelTransaction(ctx, gatewayRef)
}
