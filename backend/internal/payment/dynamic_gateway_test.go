package payment

import (
	"context"
	"testing"
)

// stubRuntimeSettings is a RuntimeSettings backed by a map.
type stubRuntimeSettings map[string]string

func (s stubRuntimeSettings) String(_ context.Context, settingKey, envKey, def string) string {
	if v, ok := s[settingKey]; ok && v != "" {
		return v
	}
	return def
}

func (s stubRuntimeSettings) Bool(_ context.Context, settingKey, envKey string, def bool) bool {
	if v, ok := s[settingKey]; ok {
		return v == "true"
	}
	return def
}

func (s stubRuntimeSettings) Secret(_ context.Context, settingKey, envKey string) string {
	return s[settingKey]
}

// TestDynamicMidtrans_EnabledRequiresFlagAndKey verifies Midtrans is only
// considered available when both the enable flag and a server key are present —
// so a half-configured gateway never breaks checkout.
func TestDynamicMidtrans_EnabledRequiresFlagAndKey(t *testing.T) {
	cases := []struct {
		name     string
		settings stubRuntimeSettings
		want     bool
	}{
		{"both present", stubRuntimeSettings{"midtrans_enabled": "true", "midtrans_server_key": "sk"}, true},
		{"flag only", stubRuntimeSettings{"midtrans_enabled": "true"}, false},
		{"key only", stubRuntimeSettings{"midtrans_server_key": "sk"}, false},
		{"neither", stubRuntimeSettings{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewDynamicMidtransGateway(tc.settings)
			if got := g.Enabled(context.Background()); got != tc.want {
				t.Fatalf("Enabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRegistry_SkipsUnavailableGateway verifies the resolver falls back to
// manual when the configured default (midtrans) is not usable.
func TestRegistry_SkipsUnavailableGateway(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewDynamicMidtransGateway(stubRuntimeSettings{})) // unavailable
	reg.Register(NewDynamicManualGateway(stubRuntimeSettings{"manual_bank_name": "BCA"}))

	settings := stubSettingsRepo{values: map[string]string{"payment_gateway_default": "midtrans"}}
	g, err := reg.ResolveGateway(context.Background(), settings, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Name() != "manual" {
		t.Fatalf("expected fallback to manual, got %q", g.Name())
	}
}

// TestRegistry_UsesAvailableGateway verifies the configured gateway is used when
// it reports itself available.
func TestRegistry_UsesAvailableGateway(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewDynamicMidtransGateway(stubRuntimeSettings{"midtrans_enabled": "true", "midtrans_server_key": "sk"}))
	reg.Register(NewDynamicManualGateway(stubRuntimeSettings{}))

	settings := stubSettingsRepo{values: map[string]string{"payment_gateway_default": "midtrans"}}
	g, err := reg.ResolveGateway(context.Background(), settings, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Name() != "midtrans" {
		t.Fatalf("expected midtrans, got %q", g.Name())
	}
}

// stubSettingsRepo implements SettingsRepository for the registry tests.
type stubSettingsRepo struct{ values map[string]string }

func (s stubSettingsRepo) GetString(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}
func (s stubSettingsRepo) GetInt(context.Context, string) (int64, error) { return 0, nil }
func (s stubSettingsRepo) Set(context.Context, string, any) error        { return nil }

// TestDynamicManual_ResolvesBankDetails verifies the manual gateway reads the
// bank details at call time (so an operator edit applies immediately).
func TestDynamicManual_ResolvesBankDetails(t *testing.T) {
	g := NewDynamicManualGateway(stubRuntimeSettings{
		"manual_bank_name":      "BCA",
		"manual_account_number": "1234567890",
		"manual_account_holder": "PT FinTrack",
	})
	res, err := g.CreateTransaction(context.Background(), CreateTxRequest{AmountIDR: 100000, OrderID: "ord-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	instr, ok := res.Raw["instructions"].(map[string]any)
	if !ok {
		t.Fatalf("expected instructions map, got %T", res.Raw["instructions"])
	}
	if instr["bank_name"] != "BCA" || instr["account_number"] != "1234567890" || instr["account_holder"] != "PT FinTrack" {
		t.Fatalf("unexpected instructions: %+v", instr)
	}
}
