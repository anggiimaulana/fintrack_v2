// Package realtime implements the Server-Sent Events hub: per-user and
// admin-wide streams, Redis pub/sub fan-out across instances, heartbeat, and
// Last-Event-ID resume.
//
// Realtime is a UX accelerator, not a source of truth. Losing an event (client
// offline, Redis down) never loses data — the underlying row is still in
// PostgreSQL and the UI picks it up on its next fetch.
package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/logger"
	"go.uber.org/zap"
)

// maxConnsPerUser bounds concurrent streams per user. Every connection holds a
// goroutine plus a heartbeat ticker, so an unbounded number (a client opening
// tabs in a loop) would leak server resources. The oldest stream is evicted.
const maxConnsPerUser = 5

// Client is one connected SSE stream.
type Client struct {
	UserID  uuid.UUID
	IsAdmin bool
	// ch receives events to write to this stream. Buffered so a slow client
	// cannot block the hub's broadcast path.
	ch chan contracts.Event
	// closed guards against double-close.
	closed chan struct{}
	once   sync.Once
	// connectedAt orders eviction when a user exceeds maxConnsPerUser.
	connectedAt time.Time
}

// Events exposes the receive channel (read-only).
func (c *Client) Events() <-chan contracts.Event { return c.ch }

// Done is closed when the client is unregistered.
func (c *Client) Done() <-chan struct{} { return c.closed }

// close signals the stream to end. Safe to call more than once.
func (c *Client) close() {
	c.once.Do(func() { close(c.closed) })
}

// Hub tracks live streams and fans events out to them.
type Hub struct {
	mu sync.RWMutex
	// byUser maps a user to their streams.
	byUser map[uuid.UUID]map[*Client]struct{}
	// admins holds streams belonging to admin/super_admin users.
	admins map[*Client]struct{}

	// now is injectable for tests.
	now func() time.Time
}

// NewHub builds an empty hub.
func NewHub() *Hub {
	return &Hub{
		byUser: make(map[uuid.UUID]map[*Client]struct{}),
		admins: make(map[*Client]struct{}),
		now:    time.Now,
	}
}

// Register adds a client to the hub. When the user already has
// maxConnsPerUser streams, the oldest is evicted and its stream ended.
func (h *Hub) Register(userID uuid.UUID, isAdmin bool) *Client {
	c := &Client{
		UserID:      userID,
		IsAdmin:     isAdmin,
		ch:          make(chan contracts.Event, 16),
		closed:      make(chan struct{}),
		connectedAt: h.now(),
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	set := h.byUser[userID]
	if set == nil {
		set = make(map[*Client]struct{})
		h.byUser[userID] = set
	}
	if len(set) >= maxConnsPerUser {
		if oldest := oldestClient(set); oldest != nil {
			delete(set, oldest)
			oldest.close()
		}
	}
	set[c] = struct{}{}

	if isAdmin {
		h.admins[c] = struct{}{}
	}
	return c
}

// Unregister removes a client and ends its stream. Idempotent.
func (h *Hub) Unregister(c *Client) {
	if c == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeLocked(c)
	c.close()
}

func (h *Hub) removeLocked(c *Client) {
	if set, ok := h.byUser[c.UserID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.byUser, c.UserID)
		}
	}
	delete(h.admins, c)
}

// SendToUser delivers an event to every stream of one user.
func (h *Hub) SendToUser(userID uuid.UUID, ev contracts.Event) int {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.byUser[userID]))
	for c := range h.byUser[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	return h.deliver(targets, ev)
}

// SendToAdmins delivers an event to every admin stream.
func (h *Hub) SendToAdmins(ev contracts.Event) int {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.admins))
	for c := range h.admins {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	return h.deliver(targets, ev)
}

// deliver pushes to each target without blocking. A full buffer means the client
// is too slow: the event is dropped for that client rather than stalling the
// broadcaster (the client re-syncs on its next fetch).
func (h *Hub) deliver(targets []*Client, ev contracts.Event) int {
	sent := 0
	for _, c := range targets {
		select {
		case <-c.closed:
			continue
		default:
		}
		select {
		case c.ch <- ev:
			sent++
		default:
			logger.Warn("realtime: dropped event for slow client",
				zap.String("user_id", c.UserID.String()), zap.String("event", ev.Event))
		}
	}
	return sent
}

// ConnCount returns the number of live streams for a user (tests/diagnostics).
func (h *Hub) ConnCount(userID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.byUser[userID])
}

// AdminCount returns the number of live admin streams.
func (h *Hub) AdminCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.admins)
}

// CloseAll ends every stream (shutdown).
func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, set := range h.byUser {
		for c := range set {
			c.close()
		}
	}
	h.byUser = make(map[uuid.UUID]map[*Client]struct{})
	h.admins = make(map[*Client]struct{})
}

// oldestClient returns the earliest-connected client in a set.
func oldestClient(set map[*Client]struct{}) *Client {
	var oldest *Client
	for c := range set {
		if oldest == nil || c.connectedAt.Before(oldest.connectedAt) {
			oldest = c
		}
	}
	return oldest
}

// WaitContext blocks until ctx is done, then unregisters the client. Handlers
// call this to guarantee cleanup when the HTTP request context ends.
func (h *Hub) WaitContext(ctx context.Context, c *Client) {
	<-ctx.Done()
	h.Unregister(c)
}
