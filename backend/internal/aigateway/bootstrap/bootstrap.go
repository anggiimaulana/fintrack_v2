// Package bootstrap seeds the centralized AI gateway configuration from
// environment defaults the FIRST time the platform starts against an empty
// database.
//
// Design contract ("env defaults, DB overrides"):
//
//   - Environment variables only ever FILL AN EMPTY TABLE. The moment the Super
//     Admin has configured anything through the admin UI (or a prior bootstrap
//     run), the database is authoritative and the env values are ignored for
//     that table — they are never used to overwrite an operator decision.
//   - This keeps a fresh checkout self-configuring (a single env var gets a
//     working gateway) while guaranteeing that "everything is dynamic from the
//     Super Admin side": once data exists, only the admin can change it.
//
// The seeding is deliberately split per table so an operator who has defined
// endpoints but no routes still gets the default routes, and vice versa.
package bootstrap

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/crypto"
)

// Config holds the environment-provided defaults. Empty fields are skipped.
type Config struct {
	// EndpointName is the unique name of the seeded endpoint.
	EndpointName string
	// BaseURL is the OpenAI-compatible API root, e.g. http://127.0.0.1:20128/v1.
	BaseURL string
	// APIKey is the upstream key. It is encrypted at rest before insertion.
	// Empty means a keyless endpoint (e.g. a local proxy that needs no auth).
	APIKey string
	// Provider is a free-form label shown in the admin UI (e.g. "9router").
	Provider string
	// Modalities advertised by the endpoint.
	Modalities []string
	// Models maps a gateway use case to the default upstream model.
	Models map[domain.UseCase]string
	// TimeoutsMS maps a gateway use case to the per-route timeout seeded on a
	// fresh install. Slow modalities (vision/voice) need a generous ceiling so a
	// healthy-but-slow provider is not aborted mid-answer. Missing keys fall
	// back to the column default.
	TimeoutsMS map[domain.UseCase]int
	// EnableFlag seeds the feature_ai_gateway_centralized flag when it is absent.
	EnableFlag bool
}

// ConfigFromEnv reads the defaults from the environment, applying the
// documented fallbacks.
func ConfigFromEnv(getenv func(string) string) Config {
	baseURL := strings.TrimSpace(getenv("AI_GATEWAY_BASE_URL"))
	name := strings.TrimSpace(getenv("AI_GATEWAY_ENDPOINT_NAME"))
	if name == "" {
		name = "9router-primary"
	}
	provider := strings.TrimSpace(getenv("AI_GATEWAY_PROVIDER"))
	if provider == "" {
		provider = "9router"
	}
	return Config{
		EndpointName: name,
		BaseURL:      baseURL,
		APIKey:       strings.TrimSpace(getenv("AI_GATEWAY_API_KEY")),
		Provider:     provider,
		Modalities:   []string{"chat", "vision", "voice", "embedding", "reasoning"},
		Models: map[domain.UseCase]string{
			domain.UseCaseChat:      envOr(getenv, "AI_GATEWAY_MODEL_CHAT", "gemini/gemini-3.5-flash-lite"),
			domain.UseCaseReasoning: envOr(getenv, "AI_GATEWAY_MODEL_REASONING", "deepseek-r1"),
			domain.UseCaseVision:    envOr(getenv, "AI_GATEWAY_MODEL_VISION", "gemini/gemini-3.5-flash"),
			domain.UseCaseVoice:     envOr(getenv, "AI_GATEWAY_MODEL_VOICE", "whisper-1"),
			domain.UseCaseEmbedding: envOr(getenv, "AI_GATEWAY_MODEL_EMBEDDING", "gemini/gemini-embedding-001"),
		},
		TimeoutsMS: map[domain.UseCase]int{
			// Chat streams token-by-token, so a minute is plenty; vision (receipt
			// OCR) and voice transcription need room for the model to read the
			// media and answer, hence the 2-minute ceiling.
			domain.UseCaseChat:      envIntOr(getenv, "AI_GATEWAY_TIMEOUT_CHAT_MS", 60000),
			domain.UseCaseVision:    envIntOr(getenv, "AI_GATEWAY_TIMEOUT_VISION_MS", 120000),
			domain.UseCaseVoice:     envIntOr(getenv, "AI_GATEWAY_TIMEOUT_VOICE_MS", 120000),
			domain.UseCaseReasoning: envIntOr(getenv, "AI_GATEWAY_TIMEOUT_REASONING_MS", 120000),
		},
		EnableFlag: isTruthy(getenv("AI_GATEWAY_ENABLED")),
	}
}

func envOr(getenv func(string) string, key, def string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return def
}

// envIntOr reads a positive integer from the environment, falling back to def.
func envIntOr(getenv func(string) string, key string, def int) int {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// Result reports what the bootstrap did, for logging.
type Result struct {
	EndpointSeeded bool
	RoutesSeeded   int
	FlagSeeded     bool
	Skipped        bool // true when the DB already had data (env ignored)
}

// EnsureSeeded seeds the gateway tables from cfg when they are empty. It is
// idempotent and safe to call on every start.
//
// A table that already contains rows is left untouched: the operator's
// configuration wins over the environment, always.
func EnsureSeeded(ctx context.Context, pool *pgxpool.Pool, encryptionKey []byte, cfg Config) (Result, error) {
	var res Result

	if cfg.BaseURL == "" {
		// No env default to seed from; nothing to do.
		res.Skipped = true
		return res, nil
	}

	// ── Endpoints ────────────────────────────────────────────────
	var endpointCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ai_endpoints`).Scan(&endpointCount); err != nil {
		return res, fmt.Errorf("count ai_endpoints: %w", err)
	}

	var endpointID string
	if endpointCount == 0 {
		enc, err := encryptOptional(cfg.APIKey, encryptionKey)
		if err != nil {
			return res, fmt.Errorf("encrypt endpoint key: %w", err)
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO ai_endpoints (name, base_url, api_key_encrypted, provider, priority, status, health, modalities, weight)
			VALUES ($1, $2, NULLIF($3, ''), $4, 1, 'active', 'unknown', $5, 100)
			RETURNING id`,
			cfg.EndpointName, cfg.BaseURL, enc, cfg.Provider, cfg.Modalities,
		).Scan(&endpointID); err != nil {
			return res, fmt.Errorf("seed ai_endpoint: %w", err)
		}
		res.EndpointSeeded = true
	} else {
		res.Skipped = true
		// Reuse the lowest-priority active endpoint for route seeding.
		_ = pool.QueryRow(ctx,
			`SELECT id FROM ai_endpoints WHERE status = 'active' ORDER BY priority ASC, name ASC LIMIT 1`,
		).Scan(&endpointID)
	}

	// ── Routes ───────────────────────────────────────────────────
	var routeCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ai_routes`).Scan(&routeCount); err != nil {
		return res, fmt.Errorf("count ai_routes: %w", err)
	}
	if routeCount == 0 && endpointID != "" {
		for useCase, model := range cfg.Models {
			if model == "" {
				continue
			}
			// Use the configured per-use-case timeout when present; otherwise
			// fall back to the column default (30000ms) by passing 0 and letting
			// COALESCE pick the column default. Vision/voice get a generous
			// ceiling so a slow-but-healthy OCR/transcription is not aborted.
			timeoutMS := cfg.TimeoutsMS[useCase]
			if _, err := pool.Exec(ctx, `
				INSERT INTO ai_routes (use_case, endpoint_id, model, combo, priority, is_active, timeout_ms)
				VALUES ($1, $2, $3, 'default', 1, true, COALESCE(NULLIF($4, 0), 30000))`,
				string(useCase), endpointID, model, timeoutMS,
			); err != nil {
				return res, fmt.Errorf("seed ai_route %s: %w", useCase, err)
			}
			res.RoutesSeeded++
		}
	}

	// ── Feature flag ─────────────────────────────────────────────
	// Only seed when absent; an existing row (even disabled) is the operator's
	// decision and is never overwritten.
	if cfg.EnableFlag {
		tag, err := pool.Exec(ctx, `
			INSERT INTO feature_flags (key, name, description, is_enabled, rollout)
			VALUES ('feature_ai_gateway_centralized', 'AI Gateway Centralized',
			        'Route all AI traffic through the centralized endpoint pool.', true, 100)
			ON CONFLICT (key) DO NOTHING`)
		if err != nil {
			return res, fmt.Errorf("seed feature flag: %w", err)
		}
		res.FlagSeeded = tag.RowsAffected() > 0
	}

	return res, nil
}

// encryptOptional encrypts a non-empty key with AES-256-GCM, returning "" for
// an empty input (keyless endpoint).
func encryptOptional(key string, masterKey []byte) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", nil
	}
	if len(masterKey) != 32 {
		return "", fmt.Errorf("master key must be 32 bytes")
	}
	return crypto.Encrypt([]byte(key), masterKey)
}
