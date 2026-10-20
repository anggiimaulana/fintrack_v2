package currency

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/fintrack/backend/internal/runtimeconfig"
)

type mockSettingsReader struct {
	data map[string]string
}

func (m *mockSettingsReader) GetString(_ context.Context, key string) (string, error) {
	return m.data[key], nil
}

func TestOpenERProvider_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/IDR" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"result":"success","base_code":"IDR","rates":{"IDR":1,"JPY":0.0095,"USD":0.000062}}`)
	}))
	defer server.Close()

	p := &OpenERProvider{BaseURL: server.URL, Client: server.Client()}
	rate, err := p.FetchRate(context.Background(), "JPY", "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 / 0.0095 = 105.263...
	expected := 1.0 / 0.0095
	if rate < expected-0.001 || rate > expected+0.001 {
		t.Errorf("expected ~%f, got %f", expected, rate)
	}

	// Same currency
	sameRate, err := p.FetchRate(context.Background(), "IDR", "IDR")
	if err != nil || sameRate != 1.0 {
		t.Errorf("expected 1.0 for same currency, got %f, err=%v", sameRate, err)
	}
}

func TestOpenERProvider_UnsupportedCurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"result":"success","base_code":"IDR","rates":{"IDR":1}}`)
	}))
	defer server.Close()

	p := &OpenERProvider{BaseURL: server.URL, Client: server.Client()}
	_, err := p.FetchRate(context.Background(), "XYZ", "IDR")
	if !errors.Is(err, ErrCurrencyUnsupported) {
		t.Errorf("expected ErrCurrencyUnsupported, got %v", err)
	}
}

func TestOpenERProvider_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	p := &OpenERProvider{BaseURL: server.URL, Client: server.Client()}
	_, err := p.FetchRate(context.Background(), "JPY", "IDR")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}

func TestFloatRatesProvider_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/idr.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// Real FloatRates API returns numbers as quoted strings
		fmt.Fprintln(w, `{"jpy":{"code":"JPY","rate":"0.0095","inverseRate":"105.26"}}`)
	}))
	defer server.Close()

	p := &FloatRatesProvider{BaseURL: server.URL, Client: server.Client()}
	rate, err := p.FetchRate(context.Background(), "JPY", "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 105.26 {
		t.Errorf("expected 105.26, got %f", rate)
	}
}

func TestFrankfurterProvider_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"amount":1.0,"base":"JPY","rates":{"IDR":105.5}}`)
	}))
	defer server.Close()

	p := &FrankfurterProvider{BaseURL: server.URL, Client: server.Client()}
	rate, err := p.FetchRate(context.Background(), "JPY", "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 105.5 {
		t.Errorf("expected 105.5, got %f", rate)
	}
}

func TestCustomProvider_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret-123" {
			t.Errorf("expected bearer token, got: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"rate":106.2}`)
	}))
	defer server.Close()

	p := &CustomProvider{BaseURL: server.URL, APIKey: "secret-123", Client: server.Client()}
	rate, err := p.FetchRate(context.Background(), "JPY", "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 106.2 {
		t.Errorf("expected 106.2, got %f", rate)
	}
}

func TestCustomProvider_MissingBaseURL(t *testing.T) {
	p := &CustomProvider{BaseURL: "", Client: http.DefaultClient}
	_, err := p.FetchRate(context.Background(), "JPY", "IDR")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}

func TestProviderFactory_DynamicResolution(t *testing.T) {
	settings := &mockSettingsReader{
		data: map[string]string{
			"currency_rate_provider": "floatrates",
			"currency_sync_timeout_seconds": "5",
		},
	}
	resolver := runtimeconfig.New(settings, nil)
	factory := NewProviderFactory(resolver, nil)

	p, err := factory.ResolveProvider(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "floatrates" {
		t.Errorf("expected floatrates provider, got %s", p.Name())
	}

	// Dynamic switch in settings
	settings.data["currency_rate_provider"] = "frankfurter"
	resolver.Invalidate()

	p2, err := factory.ResolveProvider(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p2.Name() != "frankfurter" {
		t.Errorf("expected frankfurter provider, got %s", p2.Name())
	}
}
