package bootstrap

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

func TestConfigFromEnv_Defaults(t *testing.T) {
	get := func(string) string { return "" }
	cfg := ConfigFromEnv(get)

	if cfg.EndpointName != "9router-primary" {
		t.Errorf("endpoint name = %q, want 9router-primary", cfg.EndpointName)
	}
	if cfg.Provider != "9router" {
		t.Errorf("provider = %q, want 9router", cfg.Provider)
	}
	if cfg.BaseURL != "" {
		t.Errorf("base url should be empty when unset, got %q", cfg.BaseURL)
	}
	if cfg.EnableFlag {
		t.Error("flag should default OFF")
	}
	if got := cfg.Models[domain.UseCaseChat]; got != "gemini/gemini-3.5-flash-lite" {
		t.Errorf("chat model = %q, want gemini/gemini-3.5-flash-lite", got)
	}
	if got := cfg.Models[domain.UseCaseEmbedding]; got != "gemini/gemini-embedding-001" {
		t.Errorf("embedding model = %q", got)
	}
}

func TestConfigFromEnv_Overrides(t *testing.T) {
	env := map[string]string{
		"AI_GATEWAY_BASE_URL":        "http://127.0.0.1:20128/v1",
		"AI_GATEWAY_API_KEY":         "sk-test",
		"AI_GATEWAY_ENDPOINT_NAME":   "primary",
		"AI_GATEWAY_PROVIDER":        "9router",
		"AI_GATEWAY_ENABLED":         "true",
		"AI_GATEWAY_MODEL_CHAT":      "custom-model",
		"AI_GATEWAY_MODEL_EMBEDDING": "custom-embed",
	}
	cfg := ConfigFromEnv(func(k string) string { return env[k] })

	if cfg.BaseURL != "http://127.0.0.1:20128/v1" {
		t.Errorf("base url = %q", cfg.BaseURL)
	}
	if cfg.APIKey != "sk-test" {
		t.Errorf("api key = %q", cfg.APIKey)
	}
	if cfg.EndpointName != "primary" {
		t.Errorf("endpoint name = %q", cfg.EndpointName)
	}
	if !cfg.EnableFlag {
		t.Error("flag should be ON for AI_GATEWAY_ENABLED=true")
	}
	if cfg.Models[domain.UseCaseChat] != "custom-model" {
		t.Errorf("chat model = %q", cfg.Models[domain.UseCaseChat])
	}
	if cfg.Models[domain.UseCaseEmbedding] != "custom-embed" {
		t.Errorf("embedding model = %q", cfg.Models[domain.UseCaseEmbedding])
	}
}

// TestEnsureSeeded_SkipsWithoutBaseURL verifies the no-op path: with no env
// default there is nothing to seed and no database is touched.
func TestEnsureSeeded_SkipsWithoutBaseURL(t *testing.T) {
	res, err := EnsureSeeded(context.Background(), nil, make([]byte, 32), Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Skipped {
		t.Error("expected Skipped=true when BaseURL is empty")
	}
}

// TestEnsureSeeded_DoesNotOverwriteExisting verifies the core contract: once the
// database has data, the env defaults never overwrite it. Requires a live
// database; skips when TEST_DATABASE_URL is unset.
func TestEnsureSeeded_DoesNotOverwriteExisting(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping bootstrap integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	ctx := context.Background()

	var endpointsBefore, routesBefore int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ai_endpoints`).Scan(&endpointsBefore); err != nil {
		t.Fatalf("count endpoints: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ai_routes`).Scan(&routesBefore); err != nil {
		t.Fatalf("count routes: %v", err)
	}
	if endpointsBefore == 0 {
		t.Skip("no endpoints present; this test asserts the DB-wins path")
	}

	// A distinctive env default that must NOT be applied because rows exist.
	cfg := Config{
		EndpointName: "bootstrap-should-not-insert",
		BaseURL:      "http://127.0.0.1:9999/v1",
		APIKey:       "sk-should-not-be-written",
		Provider:     "test",
		Modalities:   []string{"chat"},
		Models:       map[domain.UseCase]string{domain.UseCaseChat: "should-not-apply"},
		EnableFlag:   true,
	}
	if _, err := EnsureSeeded(ctx, pool, make([]byte, 32), cfg); err != nil {
		t.Fatalf("EnsureSeeded: %v", err)
	}

	var endpointsAfter, routesAfter int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ai_endpoints`).Scan(&endpointsAfter)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ai_routes`).Scan(&routesAfter)

	if endpointsAfter != endpointsBefore {
		t.Errorf("endpoints changed: %d -> %d (env overwrote DB)", endpointsBefore, endpointsAfter)
	}
	if routesAfter != routesBefore {
		t.Errorf("routes changed: %d -> %d (env overwrote DB)", routesBefore, routesAfter)
	}

	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ai_endpoints WHERE name = 'bootstrap-should-not-insert'`).Scan(&n)
	if n != 0 {
		t.Error("bootstrap inserted a new endpoint despite existing rows")
	}
}
