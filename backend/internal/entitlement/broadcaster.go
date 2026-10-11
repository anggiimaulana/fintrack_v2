package entitlement

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/pkg/logger"
)

// ChannelInvalidate is the pub/sub channel used to broadcast entitlement
// invalidation across instances.
const ChannelInvalidate = "entitlement:invalidations"

// InvalidationEvent is the payload broadcast when a user's entitlement changes.
type InvalidationEvent struct {
	UserID  uuid.UUID `json:"user_id"`
	Version int64     `json:"version"`
}

// Broadcaster publishes entitlement invalidations to peer instances and, on the
// receiving side, drops the L1 cache entry for the affected user.
type Broadcaster struct {
	ps *infraRedis.PubSub
	r  *Resolver
}

// NewBroadcaster builds a broadcaster bound to a pub/sub client and resolver.
func NewBroadcaster(ps *infraRedis.PubSub, r *Resolver) *Broadcaster {
	return &Broadcaster{ps: ps, r: r}
}

// PublishInvalidate bumps the user's version and notifies peers.
func (b *Broadcaster) PublishInvalidate(ctx context.Context, userID uuid.UUID) {
	if b.r != nil {
		b.r.Invalidate(ctx, userID)
	}
	if b.ps == nil || !b.ps.Enabled() {
		return
	}
	payload, _ := json.Marshal(InvalidationEvent{UserID: userID})
	if err := b.ps.Publish(ctx, ChannelInvalidate, string(payload)); err != nil {
		logger.Warn("entitlement invalidation publish failed: " + err.Error())
	}
}

// StartSubscriber listens for peer invalidations and clears the local L1 cache.
// Returns a stop function.
func (b *Broadcaster) StartSubscriber(ctx context.Context) (stop func()) {
	if b.ps == nil || !b.ps.Enabled() {
		return func() {}
	}
	return b.ps.Subscribe(ctx, ChannelInvalidate, func(msg string) {
		var ev InvalidationEvent
		if err := json.Unmarshal([]byte(msg), &ev); err != nil {
			return
		}
		if b.r != nil {
			b.r.dropL1(ev.UserID)
		}
	})
}

// dropL1 removes only the local L1 entry (Redis is already version-bumped).
func (r *Resolver) dropL1(userID uuid.UUID) {
	r.l1mu.Lock()
	delete(r.l1, userID)
	r.l1mu.Unlock()
}

// SetL1TTL overrides the L1 cache TTL (tests).
func (r *Resolver) SetL1TTL(d time.Duration) { r.l1ttl = d }
