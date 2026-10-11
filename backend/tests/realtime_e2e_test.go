// Package tests — realtime SSE end-to-end.
//
// Boots the real router with the realtime hub, publisher and Redis pub/sub, then
// exercises the stream over HTTP. Gated on TEST_DATABASE_URL so `go test ./...`
// stays green without infrastructure.
//
// Redis is required for the cross-instance/resume assertions; when REDIS_URL is
// unset those subtests skip (the local-fallback path is covered by unit tests).
package tests

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/infrastructure/auth"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/internal/realtime"
	"github.com/fintrack/backend/pkg/contracts"
)

const rtTestSecret = "test-secret-at-least-32-characters-long"

type rtHarness struct {
	t     *testing.T
	pool  *pgxpool.Pool
	hub   *realtime.Hub
	pub   *realtime.Publisher
	buf   *realtime.Buffer
	srv   *httptest.Server
	redis *infraRedis.Client
}

func newRTHarness(t *testing.T) *rtHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping realtime E2E tests")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(pool.Close)

	// Redis is optional: without it we still verify auth, headers and the local
	// delivery path.
	var rclient *infraRedis.Client
	if url := os.Getenv("REDIS_URL"); url != "" {
		c, err := infraRedis.New(ctx, infraRedis.ConfigFromEnv(os.Getenv, func(string, int) int { return 0 }))
		if err == nil {
			rclient = c
			t.Cleanup(func() { _ = c.Close() })
		}
	}

	hub := realtime.NewHub()
	buf := realtime.NewBuffer(rclient)
	ps := infraRedis.NewPubSub(rclient)
	pub := realtime.NewPublisher(ps, hub, buf)
	sub := realtime.NewSubscriber(ps, hub)
	stop := sub.Start(ctx)
	t.Cleanup(stop)
	t.Cleanup(hub.CloseAll)

	// Minimal engine: only the stream route plus auth, mirroring how the real
	// router mounts it (outside the cached group).
	engine := gin.New()
	api := engine.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(rtTestSecret))
	api.GET("/realtime/stream", realtime.NewHandler(hub, buf).Stream)

	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)

	return &rtHarness{t: t, pool: pool, hub: hub, pub: pub, buf: buf, srv: srv, redis: rclient}
}

func (h *rtHarness) token(role string) (uuid.UUID, string) {
	h.t.Helper()
	id := uuid.New()
	email := fmt.Sprintf("rt-%s@fintrack.test", uuid.NewString())
	tok, err := auth.IssueAccessToken(id, role, email, []byte(rtTestSecret), 30*time.Minute)
	if err != nil {
		h.t.Fatalf("issue token: %v", err)
	}
	return id, tok
}

// openStream starts an SSE request and returns a reader plus a cancel func.
func (h *rtHarness) openStream(tok, query string) (*bufio.Reader, context.CancelFunc, *http.Response) {
	h.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())

	url := h.srv.URL + "/api/v1/realtime/stream"
	if query != "" {
		url += "?" + query
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel()
		h.t.Fatalf("request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)

	resp, err := h.srv.Client().Do(req)
	if err != nil {
		cancel()
		h.t.Fatalf("open stream: %v", err)
	}
	return bufio.NewReader(resp.Body), cancel, resp
}

// waitForLine reads until a line containing want appears, or the deadline passes.
func (h *rtHarness) waitForLine(r *bufio.Reader, want string, timeout time.Duration) string {
	h.t.Helper()
	deadline := time.Now().Add(timeout)
	var seen []string
	for time.Now().Before(deadline) {
		line, err := r.ReadString('\n')
		if line != "" {
			seen = append(seen, strings.TrimRight(line, "\n"))
			if strings.Contains(line, want) {
				return strings.TrimRight(line, "\n")
			}
		}
		if err != nil {
			break
		}
	}
	h.t.Fatalf("did not see %q within %s; saw:\n%s", want, timeout, strings.Join(seen, "\n"))
	return ""
}

// ─── tests ───────────────────────────────────────────────────

func TestRealtime_StreamAuthRequired(t *testing.T) {
	h := newRTHarness(t)

	resp, err := h.srv.Client().Get(h.srv.URL + "/api/v1/realtime/stream")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401", resp.StatusCode)
	}
}

func TestRealtime_DisabledReturns503(t *testing.T) {
	h := newRTHarness(t)
	t.Setenv("REALTIME_ENABLED", "off")

	_, tok := h.token("user")
	_, cancel, resp := h.openStream(tok, "")
	defer cancel()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("disabled: got %d, want 503", resp.StatusCode)
	}
}

// TestRealtime_DeliversEventToUser is the core path: publish → the connected
// client receives it with a well-formed SSE frame.
func TestRealtime_DeliversEventToUser(t *testing.T) {
	h := newRTHarness(t)
	uid, tok := h.token("user")

	r, cancel, resp := h.openStream(tok, "")
	defer cancel()
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if resp.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatal("X-Accel-Buffering must be 'no'")
	}

	// Wait for the server to register the connection before publishing.
	waitForConn(t, h.hub, uid)

	h.pub.PublishUser(context.Background(), uid, contracts.NewEvent(
		contracts.EventPaymentReceived, time.Now().Unix(),
		contracts.PaymentReceivedPayload{PaymentID: uuid.NewString(), AmountIDR: 49000, Status: "settled"}))

	line := h.waitForLine(r, "event: "+contracts.EventPaymentReceived, 5*time.Second)
	if !strings.Contains(line, contracts.EventPaymentReceived) {
		t.Fatalf("unexpected event line: %q", line)
	}
	dataLine := h.waitForLine(r, "data: ", 5*time.Second)
	if !strings.Contains(dataLine, "settled") {
		t.Fatalf("payload missing: %q", dataLine)
	}
}

// TestRealtime_UserIsolation verifies one user's stream never receives another
// user's events.
func TestRealtime_UserIsolation(t *testing.T) {
	h := newRTHarness(t)
	uidA, tokA := h.token("user")
	uidB, tokB := h.token("user")

	rA, cancelA, respA := h.openStream(tokA, "")
	defer cancelA()
	defer respA.Body.Close()
	rB, cancelB, respB := h.openStream(tokB, "")
	defer cancelB()
	defer respB.Body.Close()

	waitForConn(t, h.hub, uidA)
	waitForConn(t, h.hub, uidB)

	// Event for B only.
	h.pub.PublishUser(context.Background(), uidB, contracts.NewEvent(
		contracts.EventQuotaReset, time.Now().Unix(),
		contracts.QuotaResetPayload{UserID: uidB.String(), AllocatedTokens: 5000}))

	// B receives it.
	h.waitForLine(rB, contracts.EventQuotaReset, 5*time.Second)

	// A must not. Read with a short deadline and assert nothing arrives.
	if got := readWithin(rA, 700*time.Millisecond); strings.Contains(got, contracts.EventQuotaReset) {
		t.Fatalf("user A received user B's event: %q", got)
	}
}

// TestRealtime_AdminEventsNotDeliveredToPlainUsers verifies the admin channel is
// not leaked to regular users.
func TestRealtime_AdminEventsNotDeliveredToPlainUsers(t *testing.T) {
	h := newRTHarness(t)
	uidUser, tokUser := h.token("user")
	uidAdmin, tokAdmin := h.token("admin")

	rUser, cancelU, respU := h.openStream(tokUser, "")
	defer cancelU()
	defer respU.Body.Close()
	rAdmin, cancelA, respA := h.openStream(tokAdmin, "")
	defer cancelA()
	defer respA.Body.Close()

	waitForConn(t, h.hub, uidUser)
	waitForConn(t, h.hub, uidAdmin)

	h.pub.PublishAdmins(context.Background(), contracts.NewEvent(
		contracts.EventManualApprovalNeeded, time.Now().Unix(),
		contracts.ManualApprovalNeededPayload{PaymentID: uuid.NewString(), AmountIDR: 500000}))

	// Admin receives it.
	h.waitForLine(rAdmin, contracts.EventManualApprovalNeeded, 5*time.Second)

	// Plain user must not.
	if got := readWithin(rUser, 700*time.Millisecond); strings.Contains(got, contracts.EventManualApprovalNeeded) {
		t.Fatalf("plain user received an admin event: %q", got)
	}
}

// TestRealtime_ResumeReplaysBacklog verifies Last-Event-ID recovery: an event
// published while the client was disconnected is replayed on reconnect.
func TestRealtime_ResumeReplaysBacklog(t *testing.T) {
	h := newRTHarness(t)
	if h.redis == nil || !h.buf.Enabled() {
		t.Skip("REDIS_URL not set; resume requires the ring-buffer")
	}
	uid, tok := h.token("user")

	// Connect, capture the id of the first event, then disconnect.
	r1, cancel1, resp1 := h.openStream(tok, "")
	waitForConn(t, h.hub, uid)

	h.pub.PublishUser(context.Background(), uid, contracts.NewEvent(
		contracts.EventPlanUpdated, time.Now().Unix(),
		contracts.PlanUpdatedPayload{UserID: uid.String(), CurrentPlan: "pro"}))
	idLine := h.waitForLine(r1, "id: ", 5*time.Second)
	lastID := strings.TrimSpace(strings.TrimPrefix(idLine, "id:"))

	cancel1()
	resp1.Body.Close()
	waitForNoConn(t, h.hub, uid)

	// Publish while the client is away.
	h.pub.PublishUser(context.Background(), uid, contracts.NewEvent(
		contracts.EventQuotaReset, time.Now().Unix(),
		contracts.QuotaResetPayload{UserID: uid.String(), AllocatedTokens: 9000}))

	// Reconnect with last_event_id; the missed event must be replayed.
	r2, cancel2, resp2 := h.openStream(tok, "last_event_id="+lastID)
	defer cancel2()
	defer resp2.Body.Close()

	line := h.waitForLine(r2, contracts.EventQuotaReset, 5*time.Second)
	if !strings.Contains(line, contracts.EventQuotaReset) {
		t.Fatalf("expected replayed event, got %q", line)
	}
}

// ─── helpers ─────────────────────────────────────────────────

func waitForConn(t *testing.T, hub *realtime.Hub, uid uuid.UUID) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if hub.ConnCount(uid) > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("stream did not register")
}

func waitForNoConn(t *testing.T, hub *realtime.Hub, uid uuid.UUID) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if hub.ConnCount(uid) == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("stream did not deregister after disconnect")
}

// readWithin reads whatever arrives within d and returns it (empty when idle).
func readWithin(r *bufio.Reader, d time.Duration) string {
	type res struct{ s string }
	ch := make(chan res, 1)
	go func() {
		var b strings.Builder
		for {
			line, err := r.ReadString('\n')
			b.WriteString(line)
			if err != nil {
				break
			}
			// Stop once we have a complete frame (blank line) or heartbeat.
			if strings.HasSuffix(b.String(), "\n\n") {
				break
			}
		}
		ch <- res{b.String()}
	}()
	select {
	case v := <-ch:
		return v.s
	case <-time.After(d):
		return ""
	}
}
