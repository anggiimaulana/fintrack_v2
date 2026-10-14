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
	// emitter publishes realtime domain events (nil-safe: checks before use).
	emitter RealtimeEmitter

	// storage reports proof-storage driver status; storageSwitcher invalidates
	// its cached selection when `storage_driver` changes.
	storage         StorageStatus
	storageSwitcher StorageSwitcher

	// pricingChanged, when set, reloads the active pricing rule in the running
	// engine so a Super Admin change applies to the next commit (F5).
	pricingChanged func(ctx context.Context)

	now func() time.Time
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

// WithPricingReload installs a callback invoked after a pricing-rule mutation so
// the running PricingEngine reloads the active rule immediately (F5).
func WithPricingReload(fn func(ctx context.Context)) Option {
	return func(s *Service) { s.pricingChanged = fn }
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

// RealtimeEmitter publishes domain events to connected clients. Declared
// narrowly so the admin package does not depend on the transport.
type RealtimeEmitter interface {
	PublishUser(ctx context.Context, userID uuid.UUID, ev contracts.Event)
	PublishAdmins(ctx context.Context, ev contracts.Event)
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
