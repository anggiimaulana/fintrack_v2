package mail

import (
	"context"
	"os"
	"testing"
)

// stubSettings is a SettingReader backed by a map.
type stubSettings map[string]string

func (s stubSettings) GetString(_ context.Context, key string) (string, error) {
	return s[key], nil
}

// TestProvider_DBOverridesEnv verifies a stored setting wins over the env
// default, so an operator can change SMTP without a redeploy.
func TestProvider_DBOverridesEnv(t *testing.T) {
	t.Setenv("SMTP_HOST", "env-host")
	p := NewProvider(stubSettings{KeyMailHost: "db-host"}, nil)

	if got := p.Config(context.Background()).Host; got != "db-host" {
		t.Fatalf("expected DB host to win, got %q", got)
	}
}

// TestProvider_EnvFallback verifies the env default applies when no setting row
// exists — the documented fallback.
func TestProvider_EnvFallback(t *testing.T) {
	t.Setenv("SMTP_HOST", "env-host")
	t.Setenv("SMTP_PORT", "2525")
	p := NewProvider(stubSettings{}, nil)

	cfg := p.Config(context.Background())
	if cfg.Host != "env-host" || cfg.Port != 2525 {
		t.Fatalf("expected env fallback, got host=%q port=%d", cfg.Host, cfg.Port)
	}
}

// TestProvider_NoopWhenUnconfigured verifies a missing config yields the no-op
// sender rather than a half-initialised one.
func TestProvider_NoopWhenUnconfigured(t *testing.T) {
	os.Unsetenv("SMTP_HOST")
	p := NewProvider(stubSettings{}, nil)

	if p.Sender(context.Background()).Configured() {
		t.Fatal("unconfigured provider must report not configured")
	}
	if err := p.Sender(context.Background()).Send(context.Background(), "a@b.co", "s", "b"); err != ErrDisabled {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}

// TestEncryptPassword_RoundTrip verifies a stored password can be decrypted with
// the same key (the DB never holds plaintext).
func TestEncryptPassword_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	enc, err := EncryptPassword("s3cret", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if enc == "s3cret" {
		t.Fatal("ciphertext must differ from plaintext")
	}
	p := NewProvider(stubSettings{KeyMailPassword: enc, KeyMailHost: "h", KeyMailEnabled: "true", KeyMailFrom: "f@x.co"}, key)
	if got := p.Config(context.Background()).Password; got != "s3cret" {
		t.Fatalf("expected decrypted password, got %q", got)
	}
}
