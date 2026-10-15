// Package configcatalog turns the quota/entitlement configuration into DATA.
//
// Historically three things were hardcoded in Go:
//
//  1. the settings allowlist + validators (admin/settings.go),
//  2. the limit_key → usage-resolver mapping (middleware/usage_resolvers.go),
//  3. the feature/limit key catalog (entitlement/catalog.go).
//
// This package reads them from three tables (setting_definitions,
// metric_definitions, feature_definitions) so an operator can add or change
// configuration from the Super Admin UI WITHOUT a redeploy. The code keeps the
// CAPABILITY (resolvers); the database holds the POLICY.
//
// Freshness is event-driven, not TTL-based: the cache is version-stamped and is
// only reloaded when the version changes (bumped on every config write and
// propagated across instances by Redis). A generous TTL acts purely as a last
// resort for changes made outside the API (e.g. a manual SQL edit).
package configcatalog

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CacheVersionScope is the version scope bumped whenever any config catalog row
// changes. A single scope keeps invalidation O(1) and cross-cutting.
const CacheVersionScope = "cfg"

// VersionStore is the narrow slice of the cache engine this package needs.
// BumpVersion increments a scope's version; GetVersion reads it. Both are
// implemented by infraRedis.CacheEngine (with an in-memory fallback when Redis
// is unavailable), so this package works in single-instance mode too.
type VersionStore interface {
	GetVersion(ctx context.Context, scope string) int64
	BumpVersion(ctx context.Context, scope string) int64
}

// SettingDefinition describes one writable setting.
type SettingDefinition struct {
	Key          string          `json:"key"`
	Kind         string          `json:"kind"`
	MinValue     *int64          `json:"min_value,omitempty"`
	MaxValue     *int64          `json:"max_value,omitempty"`
	EnumValues   []string        `json:"enum_values,omitempty"`
	MaxLen       int             `json:"max_len"`
	DefaultValue json.RawMessage `json:"default_value,omitempty"`
	GroupKey     string          `json:"group_key"`
	Label        string          `json:"label"`
	Description  string          `json:"description"`
	IsWritable   bool            `json:"is_writable"`
	IsSecret     bool            `json:"is_secret"`
	SortOrder    int             `json:"sort_order"`
	IsActive     bool            `json:"is_active"`
}

// MetricDefinition describes one enforceable numeric limit.
type MetricDefinition struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	Unit          string `json:"unit"`
	Description   string `json:"description"`
	IsEnforced    bool   `json:"is_enforced"`
	ResolverKey   string `json:"resolver_key,omitempty"`
	RatioAdjusted bool   `json:"ratio_adjusted"`
	PerCycle      bool   `json:"per_cycle"`
	GroupKey      string `json:"group_key"`
	SortOrder     int    `json:"sort_order"`
	IsActive      bool   `json:"is_active"`
}

// FeatureDefinition describes one boolean feature switch.
type FeatureDefinition struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	GroupKey    string `json:"group_key"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

// Catalog holds the three definition sets plus the version they were read at.
type Catalog struct {
	Settings map[string]SettingDefinition
	Metrics  map[string]MetricDefinition
	Features map[string]FeatureDefinition
	// Ordered slices preserve sort_order for UI rendering.
	SettingsList []SettingDefinition
	MetricsList  []MetricDefinition
	FeaturesList []FeatureDefinition
	Version      int64
}

// CatalogStore loads and caches the catalog. The cache is version-stamped: a
// read only hits the database when the shared version has moved, so a config
// change is visible on the very next read without polling.
type CatalogStore struct {
	pool    *pgxpool.Pool
	version VersionStore
	// fallbackTTL bounds staleness when the version store is unavailable or the
	// change happened outside the API (manual SQL). It is deliberately long so
	// it never becomes the primary freshness mechanism.
	fallbackTTL time.Duration

	mu      sync.RWMutex
	catalog *Catalog
	loaded  time.Time
}

// NewCatalogStore builds the store. version may be nil (falls back to TTL-only).
func NewCatalogStore(pool *pgxpool.Pool, version VersionStore) *CatalogStore {
	return &CatalogStore{pool: pool, version: version, fallbackTTL: 5 * time.Minute}
}

// WithFallbackTTL overrides the last-resort TTL (tests).
func (s *CatalogStore) WithFallbackTTL(d time.Duration) *CatalogStore {
	s.fallbackTTL = d
	return s
}

// Get returns the current catalog, reloading when the version changed or the
// fallback TTL lapsed. On a load error it serves the last good catalog so a
// transient DB blip never breaks request handling.
func (s *CatalogStore) Get(ctx context.Context) *Catalog {
	ver := int64(0)
	if s.version != nil {
		ver = s.version.GetVersion(ctx, CacheVersionScope)
	}

	s.mu.RLock()
	c := s.catalog
	fresh := c != nil && c.Version == ver && time.Since(s.loaded) < s.fallbackTTL
	s.mu.RUnlock()
	if fresh {
		return c
	}

	loaded, err := s.load(ctx, ver)
	if err != nil {
		if c != nil {
			return c // serve last good catalog
		}
		return emptyCatalog(ver) // never nil: callers index into the maps
	}
	s.mu.Lock()
	s.catalog = loaded
	s.loaded = time.Now()
	s.mu.Unlock()
	return loaded
}

// Invalidate bumps the shared version so every instance reloads on its next
// read. Call it AFTER the config write commits.
func (s *CatalogStore) Invalidate(ctx context.Context) {
	if s.version != nil {
		s.version.BumpVersion(ctx, CacheVersionScope)
	}
}

// UpsertMetric inserts or updates a metric definition, then invalidates the
// cache so the change is visible on the next read (and to every instance).
// Adding a metric needs no deploy: the code provides the resolver capability,
// the operator chooses it here.
func (s *CatalogStore) UpsertMetric(ctx context.Context, m MetricDefinition) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO metric_definitions
		    (key, label, unit, description, is_enforced, resolver_key, ratio_adjusted,
		     per_cycle, group_key, sort_order, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,true)
		ON CONFLICT (key) DO UPDATE SET
		    label = EXCLUDED.label, unit = EXCLUDED.unit, description = EXCLUDED.description,
		    is_enforced = EXCLUDED.is_enforced, resolver_key = EXCLUDED.resolver_key,
		    ratio_adjusted = EXCLUDED.ratio_adjusted, per_cycle = EXCLUDED.per_cycle,
		    group_key = EXCLUDED.group_key, sort_order = EXCLUDED.sort_order,
		    is_active = true, updated_at = now()`,
		m.Key, m.Label, m.Unit, m.Description, m.IsEnforced, nullableStr(m.ResolverKey),
		m.RatioAdjusted, m.PerCycle, m.GroupKey, m.SortOrder)
	if err != nil {
		return err
	}
	s.Invalidate(ctx)
	return nil
}

// DeactivateMetric soft-deletes a metric (is_active = false) rather than
// deleting it: plan_limits may still reference the key, and a hard delete would
// orphan that data. Invalidate so the change propagates.
func (s *CatalogStore) DeactivateMetric(ctx context.Context, key string) error {
	if _, err := s.pool.Exec(ctx,
		`UPDATE metric_definitions SET is_active = false, updated_at = now() WHERE key = $1`, key); err != nil {
		return err
	}
	s.Invalidate(ctx)
	return nil
}

// UpsertFeature inserts or updates a feature definition and invalidates.
func (s *CatalogStore) UpsertFeature(ctx context.Context, f FeatureDefinition) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO feature_definitions (key, label, description, group_key, sort_order, is_active)
		VALUES ($1,$2,$3,$4,$5,true)
		ON CONFLICT (key) DO UPDATE SET
		    label = EXCLUDED.label, description = EXCLUDED.description,
		    group_key = EXCLUDED.group_key, sort_order = EXCLUDED.sort_order,
		    is_active = true, updated_at = now()`,
		f.Key, f.Label, f.Description, f.GroupKey, f.SortOrder)
	if err != nil {
		return err
	}
	s.Invalidate(ctx)
	return nil
}

// DeactivateFeature soft-deletes a feature definition.
func (s *CatalogStore) DeactivateFeature(ctx context.Context, key string) error {
	if _, err := s.pool.Exec(ctx,
		`UPDATE feature_definitions SET is_active = false, updated_at = now() WHERE key = $1`, key); err != nil {
		return err
	}
	s.Invalidate(ctx)
	return nil
}

// nullableStr returns nil for an empty string so a nullable column stays NULL
// instead of storing "".
func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Setting returns a setting definition by key.
func (s *CatalogStore) Setting(ctx context.Context, key string) (SettingDefinition, bool) {
	d, ok := s.Get(ctx).Settings[key]
	return d, ok
}

// Metric returns a metric definition by key.
func (s *CatalogStore) Metric(ctx context.Context, key string) (MetricDefinition, bool) {
	d, ok := s.Get(ctx).Metrics[key]
	return d, ok
}

// MetricDefinitions returns the ordered metric list (for the admin UI).
func (s *CatalogStore) MetricDefinitions(ctx context.Context) []MetricDefinition {
	return s.Get(ctx).MetricsList
}

// FeatureDefinitions returns the ordered feature list (for the admin UI).
func (s *CatalogStore) FeatureDefinitions(ctx context.Context) []FeatureDefinition {
	return s.Get(ctx).FeaturesList
}

// SettingDefinition returns a setting definition by key.
func (s *CatalogStore) SettingDefinition(ctx context.Context, key string) (SettingDefinition, bool) {
	d, ok := s.Get(ctx).Settings[key]
	return d, ok
}

// SettingDefinitions returns the ordered setting list (for the admin UI).
func (s *CatalogStore) SettingDefinitions(ctx context.Context) []SettingDefinition {
	return s.Get(ctx).SettingsList
}

// LiveInt reads the OPERATOR'S CURRENT value for an integer setting, falling
// back to the definition's default and then to `fallback`.
//
// This is distinct from IntSetting (which reads only the definition default):
// an operator changing a value in the settings UI writes to the `settings`
// table, so any code path that must honour that change has to read the live
// value. Reading only the definition default would silently ignore the change.
func (s *CatalogStore) LiveInt(ctx context.Context, key string, fallback int64) int64 {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&raw)
	if err == nil && len(raw) > 0 {
		var n int64
		if json.Unmarshal(raw, &n) == nil {
			return n
		}
	}
	// No live value: use the definition's default, then the caller's fallback.
	return s.IntSetting(ctx, key, fallback)
}

// LiveString reads the OPERATOR'S CURRENT value for a string setting, falling
// back to the definition's default and then to `fallback`.
func (s *CatalogStore) LiveString(ctx context.Context, key, fallback string) string {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&raw)
	if err == nil && len(raw) > 0 {
		var v string
		if json.Unmarshal(raw, &v) == nil && v != "" {
			return v
		}
		// Value may be stored bare (not JSON-quoted).
		s := string(raw)
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
		if s != "" {
			return s
		}
	}
	if d, ok := s.Setting(ctx, key); ok && len(d.DefaultValue) > 0 {
		var v string
		if json.Unmarshal(d.DefaultValue, &v) == nil && v != "" {
			return v
		}
	}
	return fallback
}

// ResolverKeyFor returns the resolver key configured for a metric, or "" when
// the metric is unknown, inactive, or not enforced.
func (s *CatalogStore) ResolverKeyFor(ctx context.Context, metricKey string) string {
	m, ok := s.Get(ctx).Metrics[metricKey]
	if !ok || !m.IsActive || !m.IsEnforced {
		return ""
	}
	return m.ResolverKey
}

// IntSetting reads the definition's default value as an int, falling back to
// `fallback` when the definition is absent or the default is not numeric.
// Note: this reads the DEFINITION default, not the operator's current value
// (which lives in the `settings` table); callers that need the live value read
// it from the settings repository.
func (s *CatalogStore) IntSetting(ctx context.Context, key string, fallback int64) int64 {
	d, ok := s.Get(ctx).Settings[key]
	if !ok || len(d.DefaultValue) == 0 {
		return fallback
	}
	var n int64
	if err := json.Unmarshal(d.DefaultValue, &n); err != nil {
		return fallback
	}
	return n
}

func (s *CatalogStore) load(ctx context.Context, ver int64) (*Catalog, error) {
	c := emptyCatalog(ver)
	c.Version = ver

	srows, err := s.pool.Query(ctx, `
		SELECT key, kind, min_value, max_value, enum_values, max_len, default_value,
		       group_key, label, description, is_writable, is_secret, sort_order, is_active
		  FROM setting_definitions ORDER BY group_key, sort_order, key`)
	if err != nil {
		return nil, err
	}
	for srows.Next() {
		var d SettingDefinition
		var enumRaw, defRaw []byte
		if err := srows.Scan(&d.Key, &d.Kind, &d.MinValue, &d.MaxValue, &enumRaw, &d.MaxLen,
			&defRaw, &d.GroupKey, &d.Label, &d.Description, &d.IsWritable, &d.IsSecret,
			&d.SortOrder, &d.IsActive); err != nil {
			srows.Close()
			return nil, err
		}
		if len(enumRaw) > 0 {
			_ = json.Unmarshal(enumRaw, &d.EnumValues)
		}
		if len(defRaw) > 0 {
			d.DefaultValue = json.RawMessage(defRaw)
		}
		c.Settings[d.Key] = d
		c.SettingsList = append(c.SettingsList, d)
	}
	srows.Close()
	if err := srows.Err(); err != nil {
		return nil, err
	}

	mrows, err := s.pool.Query(ctx, `
		SELECT key, label, unit, description, is_enforced, COALESCE(resolver_key,''),
		       ratio_adjusted, per_cycle, group_key, sort_order, is_active
		  FROM metric_definitions ORDER BY group_key, sort_order, key`)
	if err != nil {
		return nil, err
	}
	for mrows.Next() {
		var d MetricDefinition
		if err := mrows.Scan(&d.Key, &d.Label, &d.Unit, &d.Description, &d.IsEnforced,
			&d.ResolverKey, &d.RatioAdjusted, &d.PerCycle, &d.GroupKey, &d.SortOrder, &d.IsActive); err != nil {
			mrows.Close()
			return nil, err
		}
		c.Metrics[d.Key] = d
		c.MetricsList = append(c.MetricsList, d)
	}
	mrows.Close()
	if err := mrows.Err(); err != nil {
		return nil, err
	}

	frows, err := s.pool.Query(ctx, `
		SELECT key, label, description, group_key, sort_order, is_active
		  FROM feature_definitions ORDER BY group_key, sort_order, key`)
	if err != nil {
		return nil, err
	}
	for frows.Next() {
		var d FeatureDefinition
		if err := frows.Scan(&d.Key, &d.Label, &d.Description, &d.GroupKey, &d.SortOrder, &d.IsActive); err != nil {
			frows.Close()
			return nil, err
		}
		c.Features[d.Key] = d
		c.FeaturesList = append(c.FeaturesList, d)
	}
	frows.Close()
	if err := frows.Err(); err != nil {
		return nil, err
	}

	return c, nil
}

func emptyCatalog(ver int64) *Catalog {
	return &Catalog{
		Settings: map[string]SettingDefinition{},
		Metrics:  map[string]MetricDefinition{},
		Features: map[string]FeatureDefinition{},
		Version:  ver,
	}
}
