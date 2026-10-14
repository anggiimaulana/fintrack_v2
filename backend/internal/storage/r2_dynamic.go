package storage

import (
	"context"
	"sync"
	"time"
)

// R2 setting keys. A DB value overrides the env default.
const (
	KeyR2AccountID       = "r2_account_id"
	KeyR2AccessKeyID     = "r2_access_key_id"
	KeyR2SecretAccessKey = "r2_secret_access_key"
	KeyR2Bucket          = "r2_bucket"
	KeyR2Region          = "r2_region"
	KeyR2Endpoint        = "r2_endpoint"
	KeyR2Prefix          = "r2_prefix"
	KeyR2PublicBaseURL   = "r2_public_base_url"
)

// RuntimeSettings resolves operational config with DB-over-env precedence.
// Implemented by runtimeconfig.Resolver.
type RuntimeSettings interface {
	String(ctx context.Context, settingKey, envKey, def string) string
	Secret(ctx context.Context, settingKey, envKey string) string
}

// DynamicR2Resolver builds and caches an R2Store from runtime configuration.
//
// The store is rebuilt when the resolved credentials change, so a Super Admin
// edit applies without a restart. A short TTL avoids re-reading settings on
// every object operation while still picking up a change quickly.
type DynamicR2Resolver struct {
	settings RuntimeSettings
	ttl      time.Duration

	mu      sync.Mutex
	store   *R2Store
	sig     string
	expires time.Time
}

// NewDynamicR2Resolver builds a resolver. A nil settings source yields a
// resolver that always reports "not configured".
func NewDynamicR2Resolver(settings RuntimeSettings) *DynamicR2Resolver {
	return &DynamicR2Resolver{settings: settings, ttl: 10 * time.Second}
}

// Invalidate drops the cached store so the next call rebuilds it.
func (r *DynamicR2Resolver) Invalidate() {
	r.mu.Lock()
	r.store = nil
	r.sig = ""
	r.expires = time.Time{}
	r.mu.Unlock()
}

// config resolves the effective R2 configuration.
func (r *DynamicR2Resolver) config(ctx context.Context) R2Config {
	return R2Config{
		AccountID:       r.settings.String(ctx, KeyR2AccountID, "R2_ACCOUNT_ID", ""),
		AccessKeyID:     r.settings.String(ctx, KeyR2AccessKeyID, "R2_ACCESS_KEY_ID", ""),
		SecretAccessKey: r.settings.Secret(ctx, KeyR2SecretAccessKey, "R2_SECRET_ACCESS_KEY"),
		Bucket:          r.settings.String(ctx, KeyR2Bucket, "R2_BUCKET", ""),
		Region:          r.settings.String(ctx, KeyR2Region, "R2_REGION", "auto"),
		Endpoint:        r.settings.String(ctx, KeyR2Endpoint, "R2_ENDPOINT", ""),
		Prefix:          r.settings.String(ctx, KeyR2Prefix, "R2_PREFIX", ""),
		PublicBaseURL:   r.settings.String(ctx, KeyR2PublicBaseURL, "R2_PUBLIC_BASE_URL", ""),
	}
}

// signature identifies a config so the cached store can be reused unchanged.
// The secret is included so a credential rotation rebuilds the client.
func signature(c R2Config) string {
	return c.AccountID + "\x00" + c.AccessKeyID + "\x00" + c.SecretAccessKey + "\x00" +
		c.Bucket + "\x00" + c.Region + "\x00" + c.Endpoint + "\x00" + c.Prefix + "\x00" + c.PublicBaseURL
}

// R2 returns the current store, or (nil, nil) when R2 is not fully configured.
func (r *DynamicR2Resolver) R2(ctx context.Context) (*R2Store, error) {
	if r.settings == nil {
		return nil, nil
	}
	cfg := r.config(ctx)
	if !cfg.Enabled() {
		return nil, nil
	}
	sig := signature(cfg)

	r.mu.Lock()
	if r.store != nil && r.sig == sig && time.Now().Before(r.expires) {
		s := r.store
		r.mu.Unlock()
		return s, nil
	}
	r.mu.Unlock()

	store, err := NewR2Store(ctx, cfg)
	if err != nil {
		// Keep serving the previous store (if any) rather than dropping R2 for a
		// transient build error; the caller falls back to local when nil.
		r.mu.Lock()
		prev := r.store
		r.mu.Unlock()
		return prev, nil
	}

	r.mu.Lock()
	r.store = store
	r.sig = sig
	r.expires = time.Now().Add(r.ttl)
	r.mu.Unlock()
	return store, nil
}

var _ R2Resolver = (*DynamicR2Resolver)(nil)
