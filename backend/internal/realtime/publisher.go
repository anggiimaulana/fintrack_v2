package realtime

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/logger"
)

// Redis channels. Per-user events go to their own channel; admin events share a
// single channel because the set of admins changes over time and publishing
// per-admin would require knowing that set on every emit.
const (
	channelPrefix = "realtime:u:"
	channelAdmins = "realtime:admins"
)

// UserChannel returns the pub/sub channel for a user's events.
func UserChannel(userID string) string { return channelPrefix + userID }

// AdminsChannel is the channel for admin-facing events.
func AdminsChannel() string { return channelAdmins }

// Publisher emits domain events: to Redis for cross-instance fan-out, and (when
// Redis is unavailable) directly to the local hub so single-instance
// deployments still work.
type Publisher struct {
	ps      *infraRedis.PubSub
	hub     *Hub
	buffer  *Buffer
	enabled bool
	seq     atomic.Uint64
}

// NewPublisher builds a publisher. hub may be nil in tests that only assert on
// Redis traffic.
func NewPublisher(ps *infraRedis.PubSub, hub *Hub, buf *Buffer) *Publisher {
	return &Publisher{ps: ps, hub: hub, buffer: buf, enabled: Enabled()}
}

// Enabled reads REALTIME_ENABLED (default on). When off, publishing is a no-op
// and the stream endpoint returns 503 — an incident valve mirroring
// ENTITLEMENT_ENFORCEMENT.
func Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("REALTIME_ENABLED")))
	switch v {
	case "off", "false", "0", "no":
		return false
	default:
		return true
	}
}

// nextID produces a monotonic event id: evt_<unixnano>_<seq>.
func (p *Publisher) nextID() string {
	return NewEventID(time.Now().UnixNano(), p.seq.Add(1))
}

// PublishUser emits an event to one user: buffered for resume, fanned out via
// Redis, and delivered locally.
func (p *Publisher) PublishUser(ctx context.Context, userID uuid.UUID, ev contracts.Event) {
	if !p.enabled {
		return
	}
	if ev.ID == "" {
		ev.ID = p.nextID()
	}
	if ev.Timestamp == 0 {
		ev.Timestamp = time.Now().Unix()
	}

	if p.buffer != nil {
		p.buffer.Append(ctx, userID, ev)
	}
	if p.publish(ctx, UserChannel(userID.String()), ev) {
		return
	}
	// Redis unavailable: deliver locally so single-instance still streams.
	if p.hub != nil {
		p.hub.SendToUser(userID, ev)
	}
}

// PublishAdmins emits an admin-facing event.
func (p *Publisher) PublishAdmins(ctx context.Context, ev contracts.Event) {
	if !p.enabled {
		return
	}
	if ev.ID == "" {
		ev.ID = p.nextID()
	}
	if ev.Timestamp == 0 {
		ev.Timestamp = time.Now().Unix()
	}

	if p.publish(ctx, AdminsChannel(), ev) {
		return
	}
	if p.hub != nil {
		p.hub.SendToAdmins(ev)
	}
}

// PublishConfigChanged announces a configuration-catalog change to EVERY
// connected client so the UI reloads immediately instead of waiting for a TTL.
//
// It is deliberately broadcast to all streams, not just admins: a regular user
// viewing their quota/subscription page must see an operator's change (a new
// ratio or quota) without a manual refresh. Best-effort, like every other
// realtime emit — the catalog version is the authoritative signal.
func (p *Publisher) PublishConfigChanged(ctx context.Context, scope string, keys []string) {
	ev := contracts.NewEvent(contracts.EventConfigChanged, time.Now().Unix(),
		contracts.ConfigChangedPayload{Scope: scope, Keys: keys})
	if !p.enabled {
		return
	}
	if ev.ID == "" {
		ev.ID = p.nextID()
	}
	// A global event has no single user channel, so Redis pub/sub cannot target
	// every subscriber: deliver through the hub, which reaches every locally
	// connected stream. Peers receive the change via the catalog version bump.
	if p.hub != nil {
		p.hub.SendToAll(ev)
	}
}

// publish sends to Redis, reporting whether it succeeded.
func (p *Publisher) publish(ctx context.Context, channel string, ev contracts.Event) bool {
	if p.ps == nil || !p.ps.Enabled() {
		return false
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		logger.Warn("realtime: marshal event failed", zap.Error(err))
		return false
	}
	if err := p.ps.Publish(ctx, channel, string(raw)); err != nil {
		logger.Warn("realtime: publish failed", zap.String("channel", channel), zap.Error(err))
		return false
	}
	return true
}

// Subscriber listens on Redis channels and forwards messages to the local hub.
type Subscriber struct {
	ps  *infraRedis.PubSub
	hub *Hub
}

// NewSubscriber builds a subscriber.
func NewSubscriber(ps *infraRedis.PubSub, hub *Hub) *Subscriber {
	return &Subscriber{ps: ps, hub: hub}
}

// Start subscribes to the admin channel and to per-user channels via pattern
// subscription. Returns a stop function; no-op when Redis is disabled.
func (s *Subscriber) Start(ctx context.Context) (stop func()) {
	if s.ps == nil || !s.ps.Enabled() || s.hub == nil {
		return func() {}
	}

	// Per-user events arrive on realtime:u:{id}; a pattern subscription avoids
	// subscribing/unsubscribing as users connect and disconnect.
	stopUsers := s.ps.SubscribePattern(ctx, channelPrefix+"*", func(channel, msg string) {
		s.dispatch(channel, msg, false)
	})
	stopAdmins := s.ps.Subscribe(ctx, AdminsChannel(), func(msg string) {
		s.dispatch(AdminsChannel(), msg, true)
	})

	return func() {
		stopUsers()
		stopAdmins()
	}
}

// dispatch routes a raw message to the right local recipients.
func (s *Subscriber) dispatch(channel, msg string, admins bool) {
	var ev contracts.Event
	if err := json.Unmarshal([]byte(msg), &ev); err != nil {
		logger.Warn("realtime: bad event payload", zap.String("channel", channel), zap.Error(err))
		return
	}
	if admins {
		s.hub.SendToAdmins(ev)
		return
	}
	id, ok := strings.CutPrefix(channel, channelPrefix)
	if !ok {
		return
	}
	userID, err := uuid.Parse(id)
	if err != nil {
		return
	}
	s.hub.SendToUser(userID, ev)
}
