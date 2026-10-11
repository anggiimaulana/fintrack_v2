package entitlement

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

// Resolver resolves and caches entitlements with a versioned cache.
//
// Cache key: entitlement:u:{userID}:v{version}. A version bump (on plan change)
// makes all previous entries stale instantly, across instances (the version is
// stored in the shared cache engine).
type Resolver struct {
	src   Source
	cache *infraRedis.CacheEngine
	ttl   time.Duration

	// L1 in-process cache keyed by userID (short TTL) to avoid Redis round-trips
	// on the hot path.
	l1mu  sync.RWMutex
	l1    map[uuid.UUID]l1Entry
	l1ttl time.Duration
}

type l1Entry struct {
	entitlement *Resolved
	expires     time.Time
}

// NewResolver builds a resolver. cache may be nil (no caching).
func NewResolver(src Source, cache *infraRedis.CacheEngine) *Resolver {
	return &Resolver{
		src:   src,
		cache: cache,
		ttl:   24 * time.Hour,
		l1:    map[uuid.UUID]l1Entry{},
		l1ttl: 60 * time.Second,
	}
}

// Resolve returns the effective entitlement for a user, using cache when fresh.
func (r *Resolver) Resolve(ctx context.Context, userID uuid.UUID) (*Resolved, error) {
	version := int64(0)
	if r.cache != nil {
		version = r.cache.GetVersion(ctx, "ent:"+userID.String())
	}

	// L1 hit.
	r.l1mu.RLock()
	if e, ok := r.l1[userID]; ok && time.Now().Before(e.expires) && e.entitlement.Version == version {
		r.l1mu.RUnlock()
		return e.entitlement, nil
	}
	r.l1mu.RUnlock()

	key := entitlementKey(userID, version)
	if r.cache != nil {
		if raw, ok := r.cache.Get(ctx, key); ok {
			var res Resolved
			if err := json.Unmarshal(raw, &res); err == nil {
				r.storeL1(userID, &res)
				return &res, nil
			}
		}
	}

	// Miss: resolve from source.
	res, err := r.resolveFromSource(ctx, userID)
	if err != nil {
		return nil, err
	}
	res.Version = version

	if r.cache != nil {
		if raw, err := json.Marshal(res); err == nil {
			r.cache.Set(ctx, key, raw, r.ttl)
		}
	}
	r.storeL1(userID, res)
	return res, nil
}

func (r *Resolver) resolveFromSource(ctx context.Context, userID uuid.UUID) (*Resolved, error) {
	plan, status, err := r.src.GetActivePlanForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	res := &Resolved{
		UserID:   userID,
		Features: map[string]bool{},
		Limits:   map[string]int64{},
		Status:   string(status),
	}
	if plan == nil {
		// No plan -> deny-all (fail-closed).
		return res, nil
	}
	res.PlanCode = plan.Code
	res.PlanName = plan.Name

	features, err := r.src.GetEntitlements(ctx, plan.ID)
	if err != nil {
		return nil, err
	}
	limits, err := r.src.GetLimits(ctx, plan.ID)
	if err != nil {
		return nil, err
	}
	res.Features = features
	res.Limits = limits
	return res, nil
}

func (r *Resolver) storeL1(userID uuid.UUID, res *Resolved) {
	r.l1mu.Lock()
	r.l1[userID] = l1Entry{entitlement: res, expires: time.Now().Add(r.l1ttl)}
	r.l1mu.Unlock()
}

// Invalidate bumps the user's entitlement version, making all cached entries
// stale instantly. Call this whenever the plan/subscription changes.
func (r *Resolver) Invalidate(ctx context.Context, userID uuid.UUID) {
	if r.cache != nil {
		r.cache.BumpVersion(ctx, "ent:"+userID.String())
	}
	r.l1mu.Lock()
	delete(r.l1, userID)
	r.l1mu.Unlock()
}

func entitlementKey(userID uuid.UUID, version int64) string {
	return "entitlement:u:" + userID.String() + ":v" + itoa(version)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// compile-time assertion that the resolver satisfies a plan lookup helper.
var _ = billing.Unlimited
