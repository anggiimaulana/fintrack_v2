package runtimeconfig

import (
	"context"
	"testing"
)

// stubSettings is a SettingReader backed by a map.
type stubSettings map[string]string

func (s stubSettings) GetString(_ context.Context, key string) (string, error) {
	return s[key], nil
}

func TestResolver_DBOverridesEnv(t *testing.T) {
	t.Setenv("MY_ENV", "env-value")
	r := New(stubSettings{"my_key": "db-value"}, nil)

	if got := r.String(context.Background(), "my_key", "MY_ENV", ""); got != "db-value" {
		t.Fatalf("expected DB value to win, got %q", got)
	}
}

func TestResolver_EnvFallbackWhenSettingAbsent(t *testing.T) {
	t.Setenv("MY_ENV", "env-value")
	r := New(stubSettings{}, nil)

	if got := r.String(context.Background(), "my_key", "MY_ENV", ""); got != "env-value" {
		t.Fatalf("expected env fallback, got %q", got)
	}
}

func TestResolver_DefaultWhenBothAbsent(t *testing.T) {
	r := New(stubSettings{}, nil)
	if got := r.String(context.Background(), "my_key", "MY_ENV_UNSET", "def"); got != "def" {
		t.Fatalf("expected default, got %q", got)
	}
	if got := r.Bool(context.Background(), "k", "MY_ENV_UNSET", true); got != true {
		t.Fatal("expected bool default")
	}
	if got := r.Int(context.Background(), "k", "MY_ENV_UNSET", 42); got != 42 {
		t.Fatal("expected int default")
	}
}

// TestResolver_SecretRoundTrip verifies a stored secret is decrypted, and that
// HasSecret reports presence without revealing the value.
func TestResolver_SecretRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	enc, err := EncryptSecret("super-secret", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	r := New(stubSettings{"my_secret": enc}, key)

	if got := r.Secret(context.Background(), "my_secret", "MY_SECRET_ENV"); got != "super-secret" {
		t.Fatalf("expected decrypted secret, got %q", got)
	}
	if !r.HasSecret(context.Background(), "my_secret", "MY_SECRET_ENV") {
		t.Fatal("HasSecret should report true when a value is stored")
	}
}

// TestResolver_SecretFallsBackToEnv verifies an absent DB secret uses env.
func TestResolver_SecretFallsBackToEnv(t *testing.T) {
	t.Setenv("MY_SECRET_ENV", "from-env")
	r := New(stubSettings{}, nil)
	if got := r.Secret(context.Background(), "my_secret", "MY_SECRET_ENV"); got != "from-env" {
		t.Fatalf("expected env secret, got %q", got)
	}
}

func TestMaskSecret(t *testing.T) {
	if got := MaskSecret(""); got != "" {
		t.Fatalf("empty should stay empty, got %q", got)
	}
	if got := MaskSecret("short"); got != "••••••••" {
		t.Fatalf("short secrets must be fully masked, got %q", got)
	}
	// Keeps a short prefix/suffix so an operator can identify the credential.
	got := MaskSecret("sk_live_abcdef123456")
	if got == "sk_live_abcdef123456" {
		t.Fatal("masked value must not equal the original")
	}
	if len(got) == 0 {
		t.Fatal("expected a masked value")
	}
}

// TestResolver_InvalidateClearsCache verifies a settings change is picked up
// after invalidation instead of serving the cached value.
func TestResolver_InvalidateClearsCache(t *testing.T) {
	settings := stubSettings{"k": "first"}
	r := New(settings, nil)
	if got := r.String(context.Background(), "k", "K_UNSET", ""); got != "first" {
		t.Fatalf("expected first, got %q", got)
	}
	settings["k"] = "second"
	// Cached: still the first value.
	if got := r.String(context.Background(), "k", "K_UNSET", ""); got != "first" {
		t.Fatalf("expected cached first, got %q", got)
	}
	r.Invalidate()
	if got := r.String(context.Background(), "k", "K_UNSET", ""); got != "second" {
		t.Fatalf("expected second after invalidate, got %q", got)
	}
}
