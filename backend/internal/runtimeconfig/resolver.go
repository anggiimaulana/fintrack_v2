// Package runtimeconfig resolves operational configuration at runtime with a
// single precedence rule: an explicit database setting wins, otherwise the
// process environment supplies the default.
//
// This is the same contract already used by storage_driver, mail_* and the AI
// gateway bootstrap, extracted so every other credential/knob follows it too.
// Values are cached for a short TTL so a per-request read does not hammer the
// settings table, while a Super Admin change still applies within seconds.
//
// What must NOT live here: bootstrap-critical config (DATABASE_URL, JWT_SECRET,
// ENCRYPTION_KEY, REDIS_URL, PORT). Those are needed to reach the database in
// the first place, so they stay environment-only.
package runtimeconfig

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fintrack/backend/internal/infrastructure/crypto"
)

// SettingsReader reads a raw setting value ("" when absent). Implemented by the
// settings repository.
type SettingsReader interface {
	GetString(ctx context.Context, key string) (string, error)
}

// Resolver reads configuration with DB-over-env precedence.
type Resolver struct {
	settings SettingsReader
	// encKey decrypts stored secrets (32 bytes, AES-256-GCM). May be nil.
	encKey []byte
	ttl    time.Duration

	mu    sync.RWMutex
	cache map[string]cached
}

type cached struct {
	value   string
	expires time.Time
}

// New builds a resolver. settings may be nil (env-only). encKey may be nil
// (secrets then come from env only).
func New(settings SettingsReader, encKey []byte) *Resolver {
	return &Resolver{
		settings: settings,
		encKey:   encKey,
		ttl:      5 * time.Second,
		cache:    map[string]cached{},
	}
}

// Invalidate clears the cache so the next read sees a fresh setting.
func (r *Resolver) Invalidate() {
	r.mu.Lock()
	r.cache = map[string]cached{}
	r.mu.Unlock()
}

// raw resolves one key: DB setting, else env, else def. Results are cached.
func (r *Resolver) raw(ctx context.Context, settingKey, envKey, def string) string {
	cacheKey := settingKey + "\x00" + envKey + "\x00" + def
	r.mu.RLock()
	if c, ok := r.cache[cacheKey]; ok && time.Now().Before(c.expires) {
		r.mu.RUnlock()
		return c.value
	}
	r.mu.RUnlock()

	value := def
	if r.settings != nil {
		if v, err := r.settings.GetString(ctx, settingKey); err == nil && strings.TrimSpace(v) != "" {
			value = v
		}
	}
	if value == def {
		// No DB override: use the environment default when present.
		if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
			value = v
		}
	}

	r.mu.Lock()
	r.cache[cacheKey] = cached{value: value, expires: time.Now().Add(r.ttl)}
	r.mu.Unlock()
	return value
}

// String resolves a plain string value.
func (r *Resolver) String(ctx context.Context, settingKey, envKey, def string) string {
	return r.raw(ctx, settingKey, envKey, def)
}

// Bool resolves a boolean value.
func (r *Resolver) Bool(ctx context.Context, settingKey, envKey string, def bool) bool {
	v := r.raw(ctx, settingKey, envKey, "")
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// Int resolves an integer value.
func (r *Resolver) Int(ctx context.Context, settingKey, envKey string, def int) int {
	v := r.raw(ctx, settingKey, envKey, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// Secret resolves a secret: the DB value is decrypted (AES-GCM hex), falling
// back to the environment when absent or undecryptable. Never logged.
func (r *Resolver) Secret(ctx context.Context, settingKey, envKey string) string {
	if r.settings != nil {
		if enc, err := r.settings.GetString(ctx, settingKey); err == nil && strings.TrimSpace(enc) != "" {
			if len(r.encKey) == 32 {
				if plain, derr := crypto.Decrypt(enc, r.encKey); derr == nil {
					return string(plain)
				}
			}
		}
	}
	return strings.TrimSpace(os.Getenv(envKey))
}

// HasSecret reports whether a value is configured for the key, from DB or env,
// without revealing it. Used by the admin UI to show "sudah diatur".
func (r *Resolver) HasSecret(ctx context.Context, settingKey, envKey string) bool {
	if r.settings != nil {
		if enc, err := r.settings.GetString(ctx, settingKey); err == nil && strings.TrimSpace(enc) != "" {
			return true
		}
	}
	return strings.TrimSpace(os.Getenv(envKey)) != ""
}

// MaskSecret returns a partially masked view of the effective secret so an
// operator can recognise which credential is active without exposing it.
func (r *Resolver) MaskSecret(ctx context.Context, settingKey, envKey string) string {
	return MaskSecret(r.Secret(ctx, settingKey, envKey))
}

// SourceOf reports where the effective value came from: "db" when an override
// is stored, else "env", else "none". Used by the UI to explain precedence.
func (r *Resolver) SourceOf(ctx context.Context, settingKey, envKey string) string {
	if r.settings != nil {
		if v, err := r.settings.GetString(ctx, settingKey); err == nil && strings.TrimSpace(v) != "" {
			return "db"
		}
	}
	if strings.TrimSpace(os.Getenv(envKey)) != "" {
		return "env"
	}
	return "none"
}

// EncryptSecret seals a plaintext secret for storage. Returns an error when no
// encryption key is configured, so callers can refuse rather than store plaintext.
func EncryptSecret(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", crypto.ErrInvalidKeyLength
	}
	return crypto.Encrypt([]byte(plaintext), key)
}

// MaskSecret renders a secret for display: keeps a short prefix/suffix so an
// operator can tell which credential is active without exposing it in full.
func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "••••••••"
	}
	return value[:4] + "••••" + value[len(value)-4:]
}
