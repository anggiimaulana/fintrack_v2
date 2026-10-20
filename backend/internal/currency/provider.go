package currency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fintrack/backend/internal/runtimeconfig"
)

var (
	ErrProviderUnavailable = errors.New("rate provider unavailable")
	ErrCurrencyUnsupported = errors.New("currency not supported by provider")
)

// RateProvider abstracts external exchange rate services.
type RateProvider interface {
	Name() string
	// FetchRate returns the amount of toCode per 1 unit of fromCode (e.g. from JPY to IDR).
	FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error)
}

// ProviderFactory dynamically resolves the configured RateProvider based on DB settings.
type ProviderFactory struct {
	resolver *runtimeconfig.Resolver
	client   *http.Client
}

// NewProviderFactory builds a ProviderFactory with the runtime configuration resolver.
func NewProviderFactory(resolver *runtimeconfig.Resolver, client *http.Client) *ProviderFactory {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &ProviderFactory{
		resolver: resolver,
		client:   client,
	}
}

// ResolveProvider returns the active RateProvider matching dynamic admin settings.
func (f *ProviderFactory) ResolveProvider(ctx context.Context) (RateProvider, error) {
	driver := "open_er_api"
	customURL := ""
	apiKey := ""
	timeoutSec := 10

	if f.resolver != nil {
		driver = f.resolver.String(ctx, "currency_rate_provider", "CURRENCY_RATE_PROVIDER", "open_er_api")
		customURL = f.resolver.String(ctx, "currency_provider_base_url", "CURRENCY_PROVIDER_BASE_URL", "")
		apiKey = f.resolver.Secret(ctx, "currency_provider_api_key", "CURRENCY_PROVIDER_API_KEY")
		timeoutSec = f.resolver.Int(ctx, "currency_sync_timeout_seconds", "CURRENCY_SYNC_TIMEOUT_SECONDS", 10)
	}

	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: f.client.Transport,
	}

	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "floatrates":
		baseURL := customURL
		if baseURL == "" {
			baseURL = "https://www.floatrates.com/daily"
		}
		return &FloatRatesProvider{BaseURL: baseURL, Client: httpClient}, nil
	case "frankfurter":
		baseURL := customURL
		if baseURL == "" {
			baseURL = "https://api.frankfurter.dev/v1"
		}
		return &FrankfurterProvider{BaseURL: baseURL, Client: httpClient}, nil
	case "custom":
		return &CustomProvider{BaseURL: customURL, APIKey: apiKey, Client: httpClient}, nil
	case "open_er_api":
		fallthrough
	default:
		baseURL := customURL
		if baseURL == "" {
			baseURL = "https://open.er-api.com/v6/latest"
		}
		return &OpenERProvider{BaseURL: baseURL, Client: httpClient}, nil
	}
}

// ─── Open Exchange Rates API (open.er-api.com) ──────────────────

type OpenERProvider struct {
	BaseURL string
	Client  *http.Client
}

func (p *OpenERProvider) Name() string { return "open_er_api" }

func (p *OpenERProvider) FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error) {
	from := strings.ToUpper(strings.TrimSpace(fromCode))
	to := strings.ToUpper(strings.TrimSpace(toCode))
	if from == to {
		return 1.0, nil
	}

	url := fmt.Sprintf("%s/%s", strings.TrimRight(p.BaseURL, "/"), to)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	req.Header.Set("User-Agent", "FinTrack-RateSync/2.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: HTTP %d", ErrProviderUnavailable, resp.StatusCode)
	}

	var data struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("%w: invalid json response: %v", ErrProviderUnavailable, err)
	}
	if data.Result != "success" || data.Rates == nil {
		return 0, fmt.Errorf("%w: response status not success", ErrProviderUnavailable)
	}

	// data.Rates contains amount of each currency per 1 unit of `toCode` (e.g. IDR).
	// If toCode is IDR and fromCode is JPY: 1 IDR = rates[JPY] JPY.
	// Therefore: 1 JPY = 1 / rates[JPY] IDR.
	rateFrom, ok := data.Rates[from]
	if !ok || rateFrom <= 0 {
		return 0, fmt.Errorf("%w: %s", ErrCurrencyUnsupported, from)
	}

	return 1.0 / rateFrom, nil
}

// ─── FloatRates Provider (floatrates.com) ───────────────────────

type FloatRatesProvider struct {
	BaseURL string
	Client  *http.Client
}

func (p *FloatRatesProvider) Name() string { return "floatrates" }

func (p *FloatRatesProvider) FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error) {
	from := strings.ToLower(strings.TrimSpace(fromCode))
	to := strings.ToLower(strings.TrimSpace(toCode))
	if from == to {
		return 1.0, nil
	}

	url := fmt.Sprintf("%s/%s.json", strings.TrimRight(p.BaseURL, "/"), to)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	req.Header.Set("User-Agent", "FinTrack-RateSync/2.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: HTTP %d", ErrProviderUnavailable, resp.StatusCode)
	}

	type floatItem struct {
		Code        string  `json:"code"`
		Rate        float64 `json:"rate"`
		InverseRate float64 `json:"inverseRate"`
	}
	var data map[string]floatItem
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("%w: invalid json response: %v", ErrProviderUnavailable, err)
	}

	item, ok := data[from]
	if !ok || item.InverseRate <= 0 {
		return 0, fmt.Errorf("%w: %s", ErrCurrencyUnsupported, fromCode)
	}

	return item.InverseRate, nil
}

// ─── Frankfurter Provider (frankfurter.dev) ─────────────────────

type FrankfurterProvider struct {
	BaseURL string
	Client  *http.Client
}

func (p *FrankfurterProvider) Name() string { return "frankfurter" }

func (p *FrankfurterProvider) FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error) {
	from := strings.ToUpper(strings.TrimSpace(fromCode))
	to := strings.ToUpper(strings.TrimSpace(toCode))
	if from == to {
		return 1.0, nil
	}

	url := fmt.Sprintf("%s/latest?from=%s&to=%s", strings.TrimRight(p.BaseURL, "/"), from, to)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	req.Header.Set("User-Agent", "FinTrack-RateSync/2.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: HTTP %d", ErrProviderUnavailable, resp.StatusCode)
	}

	var data struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("%w: invalid json response: %v", ErrProviderUnavailable, err)
	}

	rate, ok := data.Rates[to]
	if !ok || rate <= 0 {
		return 0, fmt.Errorf("%w: %s", ErrCurrencyUnsupported, from)
	}

	return rate, nil
}

// ─── Custom Provider ────────────────────────────────────────────

type CustomProvider struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func (p *CustomProvider) Name() string { return "custom" }

func (p *CustomProvider) FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error) {
	if p.BaseURL == "" {
		return 0, fmt.Errorf("%w: custom provider base URL not configured", ErrProviderUnavailable)
	}

	from := strings.ToUpper(strings.TrimSpace(fromCode))
	to := strings.ToUpper(strings.TrimSpace(toCode))
	if from == to {
		return 1.0, nil
	}

	url := fmt.Sprintf("%s?from=%s&to=%s", strings.TrimRight(p.BaseURL, "/"), from, to)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
		req.Header.Set("X-API-Key", p.APIKey)
	}
	req.Header.Set("User-Agent", "FinTrack-RateSync/2.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: HTTP %d", ErrProviderUnavailable, resp.StatusCode)
	}

	var data struct {
		Rate  float64            `json:"rate"`
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("%w: invalid json response: %v", ErrProviderUnavailable, err)
	}

	if data.Rate > 0 {
		return data.Rate, nil
	}
	if rate, ok := data.Rates[to]; ok && rate > 0 {
		return rate, nil
	}

	return 0, fmt.Errorf("%w: %s to %s", ErrCurrencyUnsupported, from, to)
}
