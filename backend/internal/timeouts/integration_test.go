package timeouts

import (
	"context"
	"testing"
	"time"

	"github.com/fintrack/backend/internal/runtimeconfig"
)

// stubSettings mirrors the settings repository: it returns the raw JSON text of
// a stored value (what GetString yields for a JSON number is the bare digits).
type stubSettings map[string]string

func (s stubSettings) GetString(_ context.Context, key string) (string, error) {
	return s[key], nil
}

// TestLoad_ThroughRuntimeResolver proves the real DB-over-env chain works: a
// stored settings row (raw JSON number, exactly as GetString returns it) is read
// and turned into the right duration, with env supplying the fallback.
func TestLoad_ThroughRuntimeResolver(t *testing.T) {
	// Simulate the settings table holding the seeded values. GetString on a JSON
	// number returns the bare digits (json.Unmarshal into a string fails, so the
	// repo returns the raw bytes).
	store := stubSettings{
		KeyChat:   "60000",
		KeyVision: "120000",
		KeyVoice:  "120000",
	}
	res := runtimeconfig.New(store, nil)

	cfg := Load(context.Background(), res)

	if cfg.Chat != 60*time.Second {
		t.Fatalf("expected chat 60s from DB, got %v", cfg.Chat)
	}
	if cfg.Vision != 120*time.Second {
		t.Fatalf("expected vision 120s from DB, got %v", cfg.Vision)
	}
	if cfg.Voice != 120*time.Second {
		t.Fatalf("expected voice 120s from DB, got %v", cfg.Voice)
	}
	// Client budget is auto-derived (slowest + margin) when not set.
	if cfg.Client != 120*time.Second+ms(clientMarginMS) {
		t.Fatalf("expected auto client budget, got %v", cfg.Client)
	}
}

// TestLoad_EnvFallback proves an env var supplies the value when no settings row
// exists, so a deployment can configure the budget without touching the DB.
func TestLoad_EnvFallback(t *testing.T) {
	t.Setenv(EnvVision, "95000")
	res := runtimeconfig.New(stubSettings{}, nil)

	cfg := Load(context.Background(), res)
	if cfg.Vision != 95*time.Second {
		t.Fatalf("expected vision 95s from env, got %v", cfg.Vision)
	}
}

// TestLoad_DBBeatsEnv locks the precedence: a settings row wins over the env var.
func TestLoad_DBBeatsEnv(t *testing.T) {
	t.Setenv(EnvVision, "95000")
	res := runtimeconfig.New(stubSettings{KeyVision: "150000"}, nil)

	cfg := Load(context.Background(), res)
	if cfg.Vision != 150*time.Second {
		t.Fatalf("expected DB to win (150s), got %v", cfg.Vision)
	}
}
