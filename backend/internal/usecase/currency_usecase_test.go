package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fintrack/backend/internal/currency"
	"github.com/fintrack/backend/internal/runtimeconfig"
)

type memoryCurrencyRepo struct {
	currencies map[string]currency.Meta
	rates      map[string][]currency.Rate
}

func newMemoryCurrencyRepo() *memoryCurrencyRepo {
	return &memoryCurrencyRepo{
		currencies: map[string]currency.Meta{
			"IDR": {Code: "IDR", Name: "Rupiah Indonesia", Symbol: "Rp", Locale: "id-ID", IsActive: true, IsBase: true},
		},
		rates: map[string][]currency.Rate{},
	}
}

func (m *memoryCurrencyRepo) List(_ context.Context, includeInactive bool) ([]currency.Meta, error) {
	out := []currency.Meta{}
	for _, c := range m.currencies {
		if includeInactive || c.IsActive {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *memoryCurrencyRepo) Get(_ context.Context, code string) (*currency.Meta, error) {
	c, ok := m.currencies[code]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

func (m *memoryCurrencyRepo) RateAsOf(_ context.Context, code string, at time.Time) (*currency.Rate, error) {
	rates, ok := m.rates[code]
	if !ok || len(rates) == 0 {
		return nil, nil
	}
	// Return newest whose EffectiveAt <= at
	for i := len(rates) - 1; i >= 0; i-- {
		if !rates[i].EffectiveAt.After(at) {
			r := rates[i]
			return &r, nil
		}
	}
	return nil, nil
}

func (m *memoryCurrencyRepo) InsertRate(_ context.Context, r currency.Rate) error {
	m.rates[r.CurrencyCode] = append(m.rates[r.CurrencyCode], r)
	return nil
}

func (m *memoryCurrencyRepo) ListRates(_ context.Context, code string, limit int) ([]currency.Rate, error) {
	rates := m.rates[code]
	if len(rates) > limit {
		rates = rates[:limit]
	}
	return rates, nil
}

func (m *memoryCurrencyRepo) CountRates(_ context.Context, code string) (int, error) {
	return len(m.rates[code]), nil
}

func (m *memoryCurrencyRepo) UpsertCurrency(_ context.Context, meta currency.Meta) error {
	m.currencies[meta.Code] = meta
	return nil
}

type staticSettings struct {
	data map[string]string
}

func (s *staticSettings) GetString(_ context.Context, key string) (string, error) {
	return s.data[key], nil
}

func TestUpsertCurrency_Validation(t *testing.T) {
	repo := newMemoryCurrencyRepo()
	svc := currency.NewService(repo)
	uc := NewCurrencyUsecase(svc, repo)

	// Invalid code
	_, err := uc.UpsertCurrency(context.Background(), CurrencyInput{Code: "JP", Name: "Yen"})
	if !errors.Is(err, ErrCurrencyInvalid) {
		t.Errorf("expected ErrCurrencyInvalid for 2-letter code, got %v", err)
	}

	// Base currency immutable
	_, err = uc.UpsertCurrency(context.Background(), CurrencyInput{Code: "IDR", Name: "Rupiah Baru"})
	if !errors.Is(err, ErrCurrencyInvalid) {
		t.Errorf("expected ErrCurrencyInvalid for modifying base, got %v", err)
	}
}

func TestUpsertCurrency_ManualWithoutRate_Rejected(t *testing.T) {
	repo := newMemoryCurrencyRepo()
	svc := currency.NewService(repo)
	uc := NewCurrencyUsecase(svc, repo)

	isActive := true
	manualSource := "manual"
	_, err := uc.UpsertCurrency(context.Background(), CurrencyInput{
		Code:       "JPY",
		Name:       "Yen Jepang",
		IsActive:   &isActive,
		RateSource: &manualSource,
	})
	if !errors.Is(err, ErrCurrencyActivation) {
		t.Fatalf("expected ErrCurrencyActivation for manual currency without rate, got %v", err)
	}
}

func TestUpsertCurrency_ProviderAutoFetch_Success(t *testing.T) {
	// Mock external rate provider server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"result":"success","base_code":"IDR","rates":{"JPY":0.0095}}`)
	}))
	defer server.Close()

	settings := &staticSettings{
		data: map[string]string{
			"currency_rate_provider":     "open_er_api",
			"currency_provider_base_url": server.URL,
		},
	}
	resolver := runtimeconfig.New(settings, nil)
	factory := currency.NewProviderFactory(resolver, server.Client())

	repo := newMemoryCurrencyRepo()
	svc := currency.NewService(repo)
	uc := NewCurrencyUsecase(svc, repo, factory)

	isActive := true
	providerSource := "provider"
	saved, err := uc.UpsertCurrency(context.Background(), CurrencyInput{
		Code:       "JPY",
		Name:       "Yen Jepang",
		IsActive:   &isActive,
		RateSource: &providerSource,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if saved == nil || !saved.IsActive {
		t.Fatalf("expected active currency, got %+v", saved)
	}

	// Verify rate was auto-inserted
	rates, err := uc.Rates(context.Background(), "JPY", 10)
	if err != nil || len(rates) != 1 {
		t.Fatalf("expected 1 rate row, got %d, err=%v", len(rates), err)
	}

	expectedRate := 1.0 / 0.0095
	if rates[0].RateToBase < expectedRate-0.01 || rates[0].RateToBase > expectedRate+0.01 {
		t.Errorf("expected ~%f, got %f", expectedRate, rates[0].RateToBase)
	}
	if rates[0].Source != "provider" {
		t.Errorf("expected source 'provider', got %s", rates[0].Source)
	}
}

func TestUpsertCurrency_ProviderAutoFetch_Failure(t *testing.T) {
	// Mock server returning HTTP 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	defer server.Close()

	settings := &staticSettings{
		data: map[string]string{
			"currency_rate_provider":     "open_er_api",
			"currency_provider_base_url": server.URL,
		},
	}
	resolver := runtimeconfig.New(settings, nil)
	factory := currency.NewProviderFactory(resolver, server.Client())

	repo := newMemoryCurrencyRepo()
	svc := currency.NewService(repo)
	uc := NewCurrencyUsecase(svc, repo, factory)

	isActive := true
	providerSource := "provider"
	_, err := uc.UpsertCurrency(context.Background(), CurrencyInput{
		Code:       "JPY",
		Name:       "Yen Jepang",
		IsActive:   &isActive,
		RateSource: &providerSource,
	})
	if err == nil {
		t.Fatal("expected error when provider is down, got nil")
	}
	if !errors.Is(err, ErrCurrencyInvalid) {
		t.Errorf("expected ErrCurrencyInvalid wrapper, got %v", err)
	}
}
