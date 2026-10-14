package storage

import (
	"context"
	"sync"
	"time"
)

// SettingKey is the settings-table key that selects the active driver.
const SettingKey = "storage_driver"

// SettingsReader reads a raw string setting (implemented by the settings repo).
type SettingsReader interface {
	GetString(ctx context.Context, key string) (string, error)
}

// R2Resolver supplies the current R2 store, rebuilt when credentials change.
// Returning (nil, nil) means "R2 not configured" — never an error, so the
// provider can silently stay on local.
type R2Resolver interface {
	R2(ctx context.Context) (*R2Store, error)
}

// Config wires the provider.
type Config struct {
	// DefaultDriver is used when no setting is present ("local" | "r2").
	DefaultDriver string
	// Local is the always-available local driver.
	Local *LocalStore
	// R2 is a static object-storage driver; nil when not configured. Ignored
	// when R2Resolver is set.
	R2 *R2Store
	// R2Resolver, when set, resolves the R2 store at runtime so credentials can
	// be changed from the Super Admin UI without a restart.
	R2Resolver R2Resolver
	// Settings optionally overrides the driver at runtime.
	Settings SettingsReader
	// CacheTTL bounds how long a driver selection is cached (default 5s).
	CacheTTL time.Duration
}

// Provider resolves the active Store at runtime and reads objects with a
// cross-driver fallback, so flipping local <-> r2 takes effect immediately for
// new uploads while previously stored objects remain readable.
type Provider struct {
	local      *LocalStore
	r2         *R2Store
	r2Resolver R2Resolver
	settings   SettingsReader
	def        string
	ttl        time.Duration

	mu      sync.RWMutex
	active  string
	expires time.Time
}

// NewProvider builds a provider. Local is required; R2 may be nil.
func NewProvider(cfg Config) *Provider {
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	def := cfg.DefaultDriver
	if def != DriverR2 {
		def = DriverLocal
	}
	p := &Provider{
		local:      cfg.Local,
		r2:         cfg.R2,
		r2Resolver: cfg.R2Resolver,
		settings:   cfg.Settings,
		def:        def,
		ttl:        ttl,
	}
	return p
}

// r2Store returns the current R2 store: the runtime-resolved one when a resolver
// is configured, else the static one. Resolution errors degrade to nil (local
// only) so a bad credential never takes storage down entirely.
func (p *Provider) r2Store(ctx context.Context) *R2Store {
	if p.r2Resolver != nil {
		s, err := p.r2Resolver.R2(ctx)
		if err != nil {
			return nil
		}
		return s
	}
	return p.r2
}

// Active returns the store for new writes. It honours the `storage_driver`
// setting, falling back to the default when the setting is absent or the
// requested driver is unavailable.
func (p *Provider) Active(ctx context.Context) (Store, error) {
	name := p.activeName(ctx)
	if name == DriverR2 {
		if r2 := p.r2Store(ctx); r2 != nil {
			return r2, nil
		}
	}
	if p.local == nil {
		return nil, ErrDisabled
	}
	return p.local, nil
}

// activeName returns the effective driver name, using a short-lived cache so a
// setting change applies within CacheTTL without a DB hit per upload.
func (p *Provider) activeName(ctx context.Context) string {
	p.mu.RLock()
	if p.active != "" && time.Now().Before(p.expires) {
		name := p.active
		p.mu.RUnlock()
		return name
	}
	p.mu.RUnlock()

	name := p.def
	if p.settings != nil {
		if v, err := p.settings.GetString(ctx, SettingKey); err == nil && v != "" {
			if v == DriverR2 {
				name = DriverR2
			} else {
				name = DriverLocal
			}
		}
	}
	// A request for r2 with no r2 configured stays on local (never hard-fails).
	if name == DriverR2 && p.r2Store(ctx) == nil {
		name = DriverLocal
	}

	p.mu.Lock()
	p.active = name
	p.expires = time.Now().Add(p.ttl)
	p.mu.Unlock()
	return name
}

// Invalidate clears the cached selection so the next call re-reads the setting.
// Call this after the `storage_driver` setting is updated.
func (p *Provider) Invalidate() {
	p.mu.Lock()
	p.active = ""
	p.expires = time.Time{}
	p.mu.Unlock()
}

// Open returns an object by key, trying the active driver first and then the
// other driver. This keeps objects written before a switch readable afterwards.
func (p *Provider) Open(ctx context.Context, key string) (*Object, error) {
	primary, _ := p.Active(ctx)
	r2 := p.r2Store(ctx)
	order := []Store{primary}
	if r2 != nil && primary != nil && primary.Name() != DriverR2 {
		order = append(order, r2)
	}
	if p.local != nil && (primary == nil || primary.Name() != DriverLocal) {
		order = append(order, p.local)
	}

	var lastErr error
	for _, s := range order {
		if s == nil {
			continue
		}
		obj, err := s.Open(ctx, key)
		if err == nil {
			return obj, nil
		}
		lastErr = err
		if err != ErrNotFound {
			// A transport error on one driver should not hide a hit on the other;
			// remember it but keep trying.
			continue
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrNotFound
}

// Delete removes an object from every configured driver (best-effort), so a
// switch never leaves a stale copy behind.
func (p *Provider) Delete(ctx context.Context, key string) error {
	var firstErr error
	for _, s := range []Store{p.local, p.r2Store(ctx)} {
		if s == nil {
			continue
		}
		if err := s.Delete(ctx, key); err != nil && err != ErrNotFound && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Status describes the provider for the admin UI: the active driver, whether
// each driver is configured/healthy, and the public URL base (if any).
type Status struct {
	Active       string `json:"active"`
	Default      string `json:"default"`
	LocalReady   bool   `json:"local_ready"`
	R2Configured bool   `json:"r2_configured"`
	R2Healthy    bool   `json:"r2_healthy"`
	R2Bucket     string `json:"r2_bucket,omitempty"`
	R2Endpoint   string `json:"r2_endpoint,omitempty"`
	R2PublicURL  string `json:"r2_public_url,omitempty"`
	LocalRoot    string `json:"local_root,omitempty"`
}

// Status reports the current storage configuration/health.
func (p *Provider) Status(ctx context.Context) Status {
	st := Status{
		Active:  p.activeName(ctx),
		Default: p.def,
	}
	if p.local != nil {
		st.LocalReady = p.local.Healthy(ctx) == nil
		st.LocalRoot = p.local.Root()
	}
	if r2 := p.r2Store(ctx); r2 != nil {
		st.R2Configured = true
		st.R2Healthy = r2.Healthy(ctx) == nil
		st.R2Bucket = r2.Bucket()
		st.R2Endpoint = r2.Endpoint()
		st.R2PublicURL = r2.PublicBaseURL()
	}
	return st
}

// DriverNames lists the drivers this provider can serve.
func (p *Provider) DriverNames() []string {
	out := []string{DriverLocal}
	if p.r2Store(context.Background()) != nil {
		out = append(out, DriverR2)
	}
	return out
}
