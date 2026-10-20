package currency

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fintrack/backend/internal/runtimeconfig"
)

type memoryWorkerRepo struct {
	currencies []Meta
	rates      map[string][]Rate
}

func (m *memoryWorkerRepo) List(_ context.Context, includeInactive bool) ([]Meta, error) {
	out := []Meta{}
	for _, c := range m.currencies {
		if includeInactive || c.IsActive {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *memoryWorkerRepo) Get(_ context.Context, code string) (*Meta, error) {
	for _, c := range m.currencies {
		if c.Code == code {
			return &c, nil
		}
	}
	return nil, nil
}

func (m *memoryWorkerRepo) RateAsOf(_ context.Context, code string, at time.Time) (*Rate, error) {
	rates, ok := m.rates[code]
	if !ok || len(rates) == 0 {
		return nil, nil
	}
	for i := len(rates) - 1; i >= 0; i-- {
		if !rates[i].EffectiveAt.After(at) {
			r := rates[i]
			return &r, nil
		}
	}
	return nil, nil
}

func (m *memoryWorkerRepo) InsertRate(_ context.Context, r Rate) error {
	m.rates[r.CurrencyCode] = append(m.rates[r.CurrencyCode], r)
	return nil
}

func (m *memoryWorkerRepo) ListRates(_ context.Context, code string, limit int) ([]Rate, error) {
	rates := m.rates[code]
	if len(rates) > limit {
		rates = rates[:limit]
	}
	return rates, nil
}

func TestRateSyncWorker_Disabled(t *testing.T) {
	settings := &mockSettingsReader{
		data: map[string]string{
			"currency_auto_sync_enabled": "false",
		},
	}
	resolver := runtimeconfig.New(settings, nil)
	factory := NewProviderFactory(resolver, nil)
	repo := &memoryWorkerRepo{}

	worker := NewRateSyncWorker(repo, factory, resolver)
	n, err := worker.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 updates when disabled, got %d", n)
	}
}

func TestRateSyncWorker_SyncSuccessAndDeduplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"result":"success","base_code":"IDR","rates":{"JPY":0.0095}}`)
	}))
	defer server.Close()

	settings := &mockSettingsReader{
		data: map[string]string{
			"currency_auto_sync_enabled":   "true",
			"currency_rate_provider":       "open_er_api",
			"currency_provider_base_url":   server.URL,
			"currency_sync_interval_hours": "6",
			"currency_dedup_hours":         "24",
		},
	}
	resolver := runtimeconfig.New(settings, nil)
	factory := NewProviderFactory(resolver, server.Client())

	repo := &memoryWorkerRepo{
		currencies: []Meta{
			{Code: "IDR", IsActive: true, IsBase: true, RateSource: "manual"},
			{Code: "USD", IsActive: true, IsBase: false, RateSource: "manual"},    // should be skipped (manual)
			{Code: "JPY", IsActive: true, IsBase: false, RateSource: "provider"},  // should be synced
		},
		rates: map[string][]Rate{},
	}

	worker := NewRateSyncWorker(repo, factory, resolver)

	// First pass: should insert JPY rate
	updated, err := worker.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on first pass: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 currency updated, got %d", updated)
	}
	if len(repo.rates["JPY"]) != 1 {
		t.Fatalf("expected 1 JPY rate, got %d", len(repo.rates["JPY"]))
	}
	if len(repo.rates["USD"]) != 0 {
		t.Errorf("expected USD (manual) to be skipped, got %d", len(repo.rates["USD"]))
	}

	// Second pass immediately with same rate: should be deduplicated
	updated2, err := worker.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on second pass: %v", err)
	}
	if updated2 != 0 {
		t.Errorf("expected 0 updates on second pass (deduplicated), got %d", updated2)
	}
	if len(repo.rates["JPY"]) != 1 {
		t.Errorf("expected JPY rates to remain at 1 after dedup, got %d", len(repo.rates["JPY"]))
	}
}

func TestRateSyncWorker_ProviderError_Graceful(t *testing.T) {
	// Server returns HTTP 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream fail", http.StatusInternalServerError)
	}))
	defer server.Close()

	settings := &mockSettingsReader{
		data: map[string]string{
			"currency_auto_sync_enabled": "true",
			"currency_rate_provider":     "open_er_api",
			"currency_provider_base_url": server.URL,
		},
	}
	resolver := runtimeconfig.New(settings, nil)
	factory := NewProviderFactory(resolver, server.Client())

	existingRate := Rate{
		CurrencyCode: "JPY",
		RateToBase:   105.0,
		EffectiveAt:  time.Now().Add(-2 * time.Hour),
		Source:       "manual",
	}

	repo := &memoryWorkerRepo{
		currencies: []Meta{
			{Code: "JPY", IsActive: true, IsBase: false, RateSource: "provider"},
		},
		rates: map[string][]Rate{
			"JPY": {existingRate},
		},
	}

	worker := NewRateSyncWorker(repo, factory, resolver)
	updated, err := worker.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("worker should not crash on provider error: %v", err)
	}
	if updated != 0 {
		t.Errorf("expected 0 updates on provider error, got %d", updated)
	}
	// Existing rate is untouched
	if len(repo.rates["JPY"]) != 1 {
		t.Errorf("expected existing rate to remain intact, got %d", len(repo.rates["JPY"]))
	}
}
