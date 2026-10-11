package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/response"
)

// heartbeatInterval keeps idle streams alive through reverse proxies, which
// typically cut silent connections after 30-60s.
const heartbeatInterval = 15 * time.Second

// Handler serves the SSE stream endpoint.
type Handler struct {
	hub    *Hub
	buffer *Buffer
}

// NewHandler builds the SSE handler.
func NewHandler(hub *Hub, buf *Buffer) *Handler {
	return &Handler{hub: hub, buffer: buf}
}

// Stream handles GET /api/v1/realtime/stream.
//
// Auth is provided by the existing AuthMiddleware (httpOnly `ft_at` cookie,
// Bearer fallback) — no token is accepted via query string.
func (h *Handler) Stream(c *gin.Context) {
	if !Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false, "error": "realtime_disabled",
			"message": "Realtime stream sedang dinonaktifkan.",
		})
		return
	}

	userIDVal, ok := c.Get("user_id")
	if !ok {
		response.Unauthorized(c, "Missing authentication")
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Unauthorized(c, "Invalid authentication")
		return
	}
	role, _ := c.Get("role")
	roleStr, _ := role.(string)
	isAdmin := roleStr == "admin" || roleStr == "super_admin"

	// Resume support: the browser sends Last-Event-ID automatically for its own
	// reconnects; our client reconnects manually (controlled backoff), so it
	// passes the id as a query param instead. Header wins when both are present.
	lastID := c.GetHeader("Last-Event-ID")
	if lastID == "" {
		lastID = c.Query("last_event_id")
	}

	client := h.hub.Register(userID, isAdmin)
	defer h.hub.Unregister(client)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.InternalError(c, "Streaming not supported")
		return
	}
	// Flush immediately: without this the response headers stay buffered until
	// the first write, so the client would not see the stream as open (and the
	// browser's `open` event would not fire) until the first heartbeat — 15s of
	// apparent hang.
	flusher.Flush()

	// Replay anything missed while disconnected, before live events.
	if h.buffer != nil && lastID != "" {
		replayed := h.buffer.Since(c.Request.Context(), userID, lastID)
		for i, ev := range replayed {
			if !writeEvent(c.Writer, ev, uint64(i+1)) {
				return
			}
		}
		flusher.Flush()
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	var seq uint64
	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-client.Done():
			return
		case ev := <-client.Events():
			seq++
			if !writeEvent(c.Writer, ev, seq) {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			// SSE comment line: ignored by EventSource, keeps TCP alive.
			if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// writeEvent renders one event in the W3C SSE frame format.
//
// An id is mandatory: the client relies on it for Last-Event-ID resume, so a
// missing one is filled here rather than silently emitting an unresumable frame.
func writeEvent(w gin.ResponseWriter, ev contracts.Event, seq uint64) bool {
	if ev.ID == "" {
		ev.ID = NewEventID(time.Now().UnixNano(), seq)
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return true // skip malformed event, keep the stream
	}
	if _, err := fmt.Fprintf(w, "id: %s\n", ev.ID); err != nil {
		return false
	}
	if ev.Event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", ev.Event); err != nil {
			return false
		}
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err == nil
}
