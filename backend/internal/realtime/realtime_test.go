package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/contracts"
)

func testEvent(name string) contracts.Event {
	return contracts.NewEvent(name, time.Now().Unix(), map[string]any{"k": "v"})
}

// ─── hub ─────────────────────────────────────────────────────

func TestHub_SendToUser_DeliversOnlyToThatUser(t *testing.T) {
	h := NewHub()
	alice, bob := uuid.New(), uuid.New()
	ca := h.Register(alice, false)
	cb := h.Register(bob, false)

	if n := h.SendToUser(alice, testEvent(contracts.EventPaymentReceived)); n != 1 {
		t.Fatalf("expected 1 delivery, got %d", n)
	}

	select {
	case <-ca.Events():
	default:
		t.Fatal("alice should have received the event")
	}
	select {
	case ev := <-cb.Events():
		t.Fatalf("bob must not receive alice's event, got %v", ev.Event)
	default:
	}
}

func TestHub_AdminEventsOnlyToAdmins(t *testing.T) {
	h := NewHub()
	user := h.Register(uuid.New(), false)
	admin := h.Register(uuid.New(), true)

	if n := h.SendToAdmins(testEvent(contracts.EventManualApprovalNeeded)); n != 1 {
		t.Fatalf("expected 1 admin delivery, got %d", n)
	}
	select {
	case <-admin.Events():
	default:
		t.Fatal("admin should receive admin events")
	}
	select {
	case ev := <-user.Events():
		t.Fatalf("non-admin must not receive admin events, got %v", ev.Event)
	default:
	}
}

func TestHub_UnregisterEndsStream(t *testing.T) {
	h := NewHub()
	id := uuid.New()
	c := h.Register(id, false)
	h.Unregister(c)

	select {
	case <-c.Done():
	default:
		t.Fatal("stream should be closed after unregister")
	}
	if got := h.ConnCount(id); got != 0 {
		t.Fatalf("conn count = %d, want 0", got)
	}
	// Unregister is idempotent.
	h.Unregister(c)
}

func TestHub_EvictsOldestBeyondLimit(t *testing.T) {
	h := NewHub()
	id := uuid.New()

	// Make connection order deterministic.
	base := time.Now()
	i := 0
	h.now = func() time.Time { i++; return base.Add(time.Duration(i) * time.Second) }

	clients := make([]*Client, 0, maxConnsPerUser+1)
	for range maxConnsPerUser + 1 {
		clients = append(clients, h.Register(id, false))
	}
	if got := h.ConnCount(id); got != maxConnsPerUser {
		t.Fatalf("conn count = %d, want %d", got, maxConnsPerUser)
	}
	// The first registered (oldest) must have been evicted.
	select {
	case <-clients[0].Done():
	default:
		t.Fatal("oldest connection should have been evicted")
	}
	select {
	case <-clients[len(clients)-1].Done():
		t.Fatal("newest connection must stay open")
	default:
	}
}

func TestHub_WaitContextCleansUp(t *testing.T) {
	h := NewHub()
	id := uuid.New()
	c := h.Register(id, false)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { h.WaitContext(ctx, c); close(done) }()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WaitContext did not return after cancel")
	}
	if got := h.ConnCount(id); got != 0 {
		t.Fatalf("conn count = %d, want 0 after context cancel", got)
	}
}

func TestHub_SlowClientDoesNotBlockBroadcast(t *testing.T) {
	h := NewHub()
	id := uuid.New()
	c := h.Register(id, false)

	// Fill the client's buffer without draining it, then one more send must not
	// block or panic.
	for i := 0; i < cap(c.ch)+5; i++ {
		h.SendToUser(id, testEvent(contracts.EventQuotaReset))
	}
}

// ─── event id helpers ────────────────────────────────────────

func TestEventID_RoundTrip(t *testing.T) {
	nanos := time.Now().UnixNano()
	id := NewEventID(nanos, 7)
	got, ok := idNanos(id)
	if !ok || got != nanos {
		t.Fatalf("idNanos(%q) = (%d, %v), want (%d, true)", id, got, ok, nanos)
	}
	if _, ok := idNanos("garbage"); ok {
		t.Fatal("garbage id must not parse")
	}
	if _, ok := idNanos(""); ok {
		t.Fatal("empty id must not parse")
	}
}

// ─── handler ─────────────────────────────────────────────────

func setupStreamRouter(role string, h *Handler) (*gin.Engine, uuid.UUID) {
	gin.SetMode(gin.TestMode)
	uid := uuid.New()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uid)
		c.Set("role", role)
		c.Next()
	})
	r.GET("/stream", h.Stream)
	return r, uid
}

func TestStream_DisabledReturns503(t *testing.T) {
	t.Setenv("REALTIME_ENABLED", "off")
	h := NewHandler(NewHub(), nil)
	r, _ := setupStreamRouter("user", h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when disabled, got %d", w.Code)
	}
}

// TestStream_HeadersAndHeartbeat verifies the SSE response contract and that the
// stream stays open (the heartbeat goroutine is what keeps proxies from cutting
// an idle connection).
func TestStream_HeadersAndHeartbeat(t *testing.T) {
	t.Setenv("REALTIME_ENABLED", "on")
	hub := NewHub()
	h := NewHandler(hub, nil)
	r, uid := setupStreamRouter("user", h)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/stream", nil).WithContext(ctx)

	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { r.ServeHTTP(w, req); close(done) }()

	// Wait for registration, then push an event.
	deadline := time.Now().Add(2 * time.Second)
	for hub.ConnCount(uid) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if hub.ConnCount(uid) == 0 {
		cancel()
		t.Fatal("stream did not register a connection")
	}
	hub.SendToUser(uid, contracts.NewEvent(contracts.EventPlanUpdated, time.Now().Unix(),
		contracts.PlanUpdatedPayload{UserID: uid.String(), CurrentPlan: "pro"}))

	// Give the writer a moment to flush.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if w.Header().Get("X-Accel-Buffering") != "no" {
		t.Fatal("X-Accel-Buffering must be 'no' for proxies")
	}
	body := w.Body.String()
	if !strings.Contains(body, "event: "+contracts.EventPlanUpdated) {
		t.Fatalf("body missing event frame: %q", body)
	}
	if !strings.Contains(body, "data: ") {
		t.Fatalf("body missing data line: %q", body)
	}
	// The event envelope must be valid JSON carrying id + payload.
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev contracts.Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			t.Fatalf("data line is not valid JSON: %v", err)
		}
		if ev.ID == "" || ev.Event != contracts.EventPlanUpdated {
			t.Fatalf("envelope missing id/event: %+v", ev)
		}
	}
}

func TestStream_Unauthenticated401(t *testing.T) {
	t.Setenv("REALTIME_ENABLED", "on")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/stream", NewHandler(NewHub(), nil).Stream) // no auth middleware

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user_id, got %d", w.Code)
	}
}

// ─── buffer id parsing (Redis-free paths) ────────────────────

func TestBuffer_DisabledIsNoop(t *testing.T) {
	b := NewBuffer(nil)
	if b.Enabled() {
		t.Fatal("buffer with nil client must be disabled")
	}
	// Must not panic and must return nothing.
	b.Append(context.Background(), uuid.New(), testEvent(contracts.EventQuotaReset))
	if got := b.Since(context.Background(), uuid.New(), "evt_1_1"); got != nil {
		t.Fatalf("disabled buffer must return no events, got %v", got)
	}
}

// ─── publisher (no Redis: local fallback) ────────────────────

func TestPublisher_LocalFallbackWithoutRedis(t *testing.T) {
	t.Setenv("REALTIME_ENABLED", "on")
	hub := NewHub()
	id := uuid.New()
	c := hub.Register(id, false)

	p := NewPublisher(nil, hub, nil)
	p.PublishUser(context.Background(), id, testEvent(contracts.EventPaymentReceived))

	select {
	case ev := <-c.Events():
		if ev.Event != contracts.EventPaymentReceived {
			t.Fatalf("event = %q", ev.Event)
		}
		if ev.ID == "" {
			t.Fatal("publisher must assign an event id")
		}
	case <-time.After(time.Second):
		t.Fatal("expected local delivery when Redis is unavailable")
	}
}

func TestPublisher_DisabledIsNoop(t *testing.T) {
	t.Setenv("REALTIME_ENABLED", "off")
	hub := NewHub()
	id := uuid.New()
	c := hub.Register(id, false)

	p := NewPublisher(nil, hub, nil)
	p.PublishUser(context.Background(), id, testEvent(contracts.EventPaymentReceived))

	select {
	case ev := <-c.Events():
		t.Fatalf("disabled publisher must not emit, got %v", ev.Event)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPublisher_AdminsChannelLocalFallback(t *testing.T) {
	t.Setenv("REALTIME_ENABLED", "on")
	hub := NewHub()
	admin := hub.Register(uuid.New(), true)

	p := NewPublisher(nil, hub, nil)
	p.PublishAdmins(context.Background(), testEvent(contracts.EventAIEndpointDown))

	select {
	case ev := <-admin.Events():
		if ev.Event != contracts.EventAIEndpointDown {
			t.Fatalf("event = %q", ev.Event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected local admin delivery")
	}
}

func TestUserChannelName(t *testing.T) {
	id := uuid.New()
	if got, want := UserChannel(id.String()), "realtime:u:"+id.String(); got != want {
		t.Fatalf("UserChannel = %q, want %q", got, want)
	}
	if AdminsChannel() != "realtime:admins" {
		t.Fatalf("AdminsChannel = %q", AdminsChannel())
	}
}

var _ = fmt.Sprintf
