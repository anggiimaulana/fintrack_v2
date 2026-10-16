// Package admin implements the Super Admin Dashboard backend (P06): metrics,
// plan/subscription/payment/token administration, AI-gateway operations, the
// PDP-sanitised audit viewer, and global settings.
//
// Every mutating operation records an entry in the append-only `audit_logs`
// table (K-29) and all monetary values are integer Rupiah (K-27).
package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/aigateway/gateway"
	"github.com/fintrack/backend/internal/audit"
	"github.com/fintrack/backend/internal/featureflag"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/runtimeconfig"
	"github.com/fintrack/backend/internal/storage"
	"github.com/fintrack/backend/pkg/contracts"
)

// Service bundles the dependencies shared by all admin handlers.
type Service struct {
	pool   *pgxpool.Pool
	cache  *infraRedis.CacheEngine
	audit  AuditWriter
	manual *payment.ManualService
	aigw   *gateway.Service

	killSwitch *KillSwitchStore
	flags      *featureflag.Service
	encKey     []byte
	// entInvalidator bumps the entitlement cache version (per user and per
	// plan). Without it a plan or subscription change is invisible until the
	// resolver's 24h TTL lapses.
	entInvalidator EntitlementInvalidator
	// quotaSyncer pushes a plan change into the user's running token period so a
	// forced plan change is not capped by the previous plan's quota (nil-safe).
	quotaSyncer QuotaSyncer
	// emitter publishes realtime domain events (nil-safe: checks before use).
	emitter RealtimeEmitter

	// storage reports proof-storage driver status; storageSwitcher invalidates
	// its cached selection when `storage_driver` changes.
	storage         StorageStatus
	storageSwitcher StorageSwitcher

	// runtimeConfig invalidates the DB-over-env resolver cache so a settings
	// change applies immediately (nil-safe).
	runtimeConfig RuntimeInvalidator
	// runtime is the shared DB-over-env resolver used by the config status
	// endpoint to report effective values and secret presence.
	runtime *runtimeconfig.Resolver

	// pricingChanged, when set, reloads the active pricing rule in the running
	// engine so a Super Admin change applies to the next commit (F5).
	pricingChanged func(ctx context.Context)

	// breakerReload, when set, re-applies AI gateway breaker tuning after a
	// settings change so the new thresholds take effect without a restart.
	breakerReload func(ctx context.Context)

	// meteringReload, when set, re-applies AI token estimation heuristics
	// after a settings change so the new parameters take effect immediately.
	meteringReload func(ctx context.Context)

	// timeoutsReload, when set, re-applies the AI timeout budget (gateway
	// per-use-case timeouts + HTTP write deadline) after a settings change, so a
	// Super Admin retune applies to in-flight routing within seconds.
	timeoutsReload func(ctx context.Context)

	// settingsBroadcaster, when set, notifies peer instances that a settings row
	// changed so they re-apply their in-memory knobs (zero-downtime fan-out).
	settingsBroadcaster SettingsBroadcaster

	// catalog is the data-driven configuration catalog (settings/metrics/
	// features). When set, validators and key lists come from the DB instead of
	// hardcoded maps. Nil falls back to the built-in defaults.
	catalog CatalogProvider

	now func() time.Time
}

// CatalogProvider exposes the dynamic configuration catalog to admin handlers.
// Declared here (not imported) so the admin package stays decoupled from the
// configcatalog implementation and is easy to fake in tests.
type CatalogProvider interface {
	// IntSetting reads a setting DEFINITION's default as an int (does not see
	// operator overrides; use LiveInt when the current value matters).
	IntSetting(ctx context.Context, key string, fallback int64) int64
	// LiveInt reads the operator's CURRENT value for an integer setting,
	// falling back to the definition default, then to `fallback`.
	LiveInt(ctx context.Context, key string, fallback int64) int64
	// LiveString reads the operator's CURRENT value for a string setting.
	LiveString(ctx context.Context, key, fallback string) string
	// SettingDefinition returns a definition (ok=false when unknown).
	SettingDefinition(ctx context.Context, key string) (SettingDef, bool)
	// Definitions returns the ordered setting catalog.
	Definitions(ctx context.Context) []SettingDef
	// Invalidate bumps the catalog version so peers reload.
	Invalidate(ctx context.Context)
	// MetricDefinitions / FeatureDefinitions return the ordered catalogs.
	MetricDefinitions(ctx context.Context) []MetricDef
	FeatureDefinitions(ctx context.Context) []FeatureDef
	// UpsertMetric / DeactivateMetric add or soft-delete a metric. The write
	// invalidates the cache so the change applies without a deploy.
	UpsertMetric(ctx context.Context, m MetricDef) error
	DeactivateMetric(ctx context.Context, key string) error
	// UpsertFeature / DeactivateFeature add or soft-delete a feature.
	UpsertFeature(ctx context.Context, f FeatureDef) error
	DeactivateFeature(ctx context.Context, key string) error
}

// SettingDef is the admin-facing view of a setting definition.
type SettingDef struct {
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

// MetricDef is the admin-facing view of a metric definition.
type MetricDef struct {
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

// FeatureDef is the admin-facing view of a feature definition.
type FeatureDef struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	GroupKey    string `json:"group_key"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

// WithCatalog installs the dynamic configuration catalog.
func WithCatalog(c CatalogProvider) Option {
	return func(s *Service) { s.catalog = c }
}

// AuditWriter records administrative actions. *audit.Service satisfies this
// interface (asynchronous, masking sensitive fields before persistence).
type AuditWriter interface {
	Log(ctx context.Context, e audit.Event)
}

// Option customises the Service.
type Option func(*Service)

// WithManualService installs the manual-payment review service used by the
// payment approval/rejection endpoints.
func WithManualService(m *payment.ManualService) Option { return func(s *Service) { s.manual = m } }

// WithGateway installs the AI gateway service (breaker access + route cache
// invalidation).
func WithGateway(g *gateway.Service) Option { return func(s *Service) { s.aigw = g } }

// WithFeatureFlags installs the feature-flag service (toggle + cache invalidation).
func WithFeatureFlags(f *featureflag.Service) Option { return func(s *Service) { s.flags = f } }

// WithEncryptionKey installs the master key used to encrypt endpoint API keys.
func WithEncryptionKey(key []byte) Option { return func(s *Service) { s.encKey = key } }

// WithCache installs a Redis-backed cache engine.
func WithCache(c *infraRedis.CacheEngine) Option { return func(s *Service) { s.cache = c } }

// StorageSwitcher lets the settings handler invalidate the storage driver cache
// so a `storage_driver` change applies immediately.
type StorageSwitcher interface {
	Invalidate()
}

// StorageStatus reports the active/available proof storage drivers.
type StorageStatus interface {
	Status(ctx context.Context) storage.Status
	DriverNames() []string
}

// WithStorage installs the proof-storage provider (runtime driver switch).
func WithStorage(p StorageStatus, switcher StorageSwitcher) Option {
	return func(s *Service) { s.storage = p; s.storageSwitcher = switcher }
}

// RuntimeInvalidator clears caches that resolve config with DB-over-env
// precedence (the runtime resolver + the R2 store), so a settings change applies
// immediately instead of waiting for the cache TTL.
type RuntimeInvalidator interface {
	Invalidate()
}

// WithRuntimeConfig installs the runtime config invalidator.
func WithRuntimeConfig(r RuntimeInvalidator) Option {
	return func(s *Service) { s.runtimeConfig = r }
}

// WithPricingReload installs a callback invoked after a pricing-rule mutation so
// the running PricingEngine reloads the active rule immediately (F5).
func WithPricingReload(fn func(ctx context.Context)) Option {
	return func(s *Service) { s.pricingChanged = fn }
}

// WithBreakerReload installs a callback invoked after a settings update so the
// running AI gateway re-reads its breaker tuning immediately.
func WithBreakerReload(fn func(ctx context.Context)) Option {
	return func(s *Service) { s.breakerReload = fn }
}

// WithMeteringReload installs a callback invoked after a settings update so
// token metering heuristics are retuned immediately.
func WithMeteringReload(fn func(ctx context.Context)) Option {
	return func(s *Service) { s.meteringReload = fn }
}

// WithTimeoutsReload installs a callback invoked after a settings update so the
// AI timeout budget (gateway per-use-case timeouts + HTTP write deadline) is
// re-applied immediately, without a restart.
func WithTimeoutsReload(fn func(ctx context.Context)) Option {
	return func(s *Service) { s.timeoutsReload = fn }
}

// SettingsBroadcaster fans a "settings changed" hint out to peer instances so
// they re-apply their in-memory knobs. *infraRedis.PubSub satisfies it.
type SettingsBroadcaster interface {
	Publish(ctx context.Context, channel, msg string) error
}

// WithSettingsBroadcaster installs the cross-instance settings notifier.
func WithSettingsBroadcaster(b SettingsBroadcaster) Option {
	return func(s *Service) { s.settingsBroadcaster = b }
}

// EntitlementInvalidator invalidates cached entitlements. *entitlement.Broadcaster
// satisfies it; a narrow interface keeps the admin package free of a hard
// dependency on the entitlement transport.
type EntitlementInvalidator interface {
	// InvalidateUser makes one user's cached entitlements stale and notifies
	// peer instances.
	InvalidateUser(ctx context.Context, userID uuid.UUID)
	// InvalidatePlan makes every subscriber of a plan stale (O(1)).
	InvalidatePlan(ctx context.Context, planID string)
}

// WithEntitlementInvalidator installs the entitlement cache invalidator.
func WithEntitlementInvalidator(inv EntitlementInvalidator) Option {
	return func(s *Service) { s.entInvalidator = inv }
}

// QuotaSyncer pushes a plan change into the user's RUNNING token period, so a
// forced downgrade/upgrade is not capped by the previous plan's quota.
type QuotaSyncer interface {
	SyncRunningPeriod(ctx context.Context, userID uuid.UUID, planCode string) error
}

// WithQuotaSyncer installs the quota syncer used after a plan change.
func WithQuotaSyncer(q QuotaSyncer) Option {
	return func(s *Service) { s.quotaSyncer = q }
}

// RealtimeEmitter publishes domain events to connected clients. Declared
// narrowly so the admin package does not depend on the transport.
type RealtimeEmitter interface {
	PublishUser(ctx context.Context, userID uuid.UUID, ev contracts.Event)
	PublishAdmins(ctx context.Context, ev contracts.Event)
	// PublishConfigChanged announces that the configuration catalog changed, so
	// clients reload immediately (P-13). Scope names the area (settings/metrics/
	// features/plans/pricing); keys lists the specific entries when known.
	PublishConfigChanged(ctx context.Context, scope string, keys []string)
}

// WithRealtimeEmitter installs the realtime emitter (plan_updated broadcasts).
func WithRealtimeEmitter(e RealtimeEmitter) Option {
	return func(s *Service) { s.emitter = e }
}

// WithClock overrides the clock (tests).
func WithClock(now func() time.Time) Option { return func(s *Service) { s.now = now } }

// NewService builds the admin service.
func NewService(pool *pgxpool.Pool, auditWriter AuditWriter, opts ...Option) *Service {
	s := &Service{pool: pool, audit: auditWriter, now: func() time.Time { return time.Now().UTC() }}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Pool exposes the underlying pool (used by tests).
func (s *Service) Pool() *pgxpool.Pool { return s.pool }

// encryptionKey returns the endpoint-key master key.
func (s *Service) encryptionKey() []byte { return s.encKey }

// record writes one audit event. Log never blocks the caller and drops on
// back-pressure, so an audit hiccup cannot stall an administrative action.
func (s *Service) record(ctx context.Context, e audit.Event) {
	if s.audit == nil {
		return
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = s.now()
	}
	s.audit.Log(ctx, e)
}

// ─── Role resolution (K-14) ───────────────────────────────────

// roleCacheTTL bounds how long a stale privilege grant can survive a demotion.
const roleCacheTTL = 60 * time.Second

// RoleResolver reads a user's current role from the DB, caching it briefly in
// Redis under `user:role_version:<id>`. Deleting that key revokes privileged
// access immediately.
type RoleResolver struct {
	pool  *pgxpool.Pool
	cache *infraRedis.CacheEngine
}

// NewRoleResolver builds a resolver. A nil cache degrades to direct DB reads.
func NewRoleResolver(pool *pgxpool.Pool, cache *infraRedis.CacheEngine) *RoleResolver {
	return &RoleResolver{pool: pool, cache: cache}
}

func roleKey(userID string) string { return "user:role_version:" + userID }

// CurrentRole returns the live role for a user.
func (r *RoleResolver) CurrentRole(ctx context.Context, userID uuid.UUID) (string, error) {
	key := roleKey(userID.String())
	if r.cache != nil {
		if b, ok := r.cache.Get(ctx, key); ok {
			return string(b), nil
		}
	}
	var role string
	if err := r.pool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, userID).Scan(&role); err != nil {
		return "", err
	}
	if r.cache != nil {
		r.cache.Set(ctx, key, []byte(role), roleCacheTTL)
	}
	return role, nil
}

// Invalidate drops the cached role for a user (call after a role change).
func (r *RoleResolver) Invalidate(ctx context.Context, userID uuid.UUID) {
	if r.cache != nil {
		r.cache.DeletePattern(ctx, roleKey(userID.String()))
	}
}

// jsonMap decodes a JSONB column into a map (nil when empty).
func jsonMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}
