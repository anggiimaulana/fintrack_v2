package realtime

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	goredis "github.com/redis/go-redis/v9"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/pkg/contracts"
)

const (
	// bufferTTL is how long events are retained for resume. The spec promises
	// clients can reconnect within 10 minutes and recover everything missed.
	bufferTTL = 10 * time.Minute
	// bufferMax caps entries per user so a long-lived stream cannot grow the
	// key without bound.
	bufferMax = 200
)

// Buffer stores recent events per user so a reconnecting client can replay what
// it missed via Last-Event-ID.
//
// When Redis is unavailable the buffer degrades to a no-op: Appends are dropped
// and Since returns nothing, so the client simply resumes with live events only.
// That is a documented limitation, not an error — the UI re-syncs on its next
// fetch.
type Buffer struct {
	client *infraRedis.Client
}

// NewBuffer builds a buffer bound to a Redis client (may be disabled).
func NewBuffer(client *infraRedis.Client) *Buffer {
	return &Buffer{client: client}
}

// Enabled reports whether the buffer can persist events.
func (b *Buffer) Enabled() bool {
	return b != nil && b.client != nil && b.client.Enabled()
}

func bufferKey(userID uuid.UUID) string { return "realtime:buf:" + userID.String() }

// Append records an event for a user. Best-effort: failures are ignored so a
// Redis blip cannot fail the caller's primary operation.
func (b *Buffer) Append(ctx context.Context, userID uuid.UUID, ev contracts.Event) {
	if !b.Enabled() {
		return
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return
	}
	key := bufferKey(userID)
	rdb := b.client.Raw()

	// Score is the event's nanosecond timestamp so range queries align with the
	// id ordering (ids are evt_<unixnano>_<seq>).
	if err := rdb.ZAdd(ctx, key, goredis.Z{Score: float64(eventNanos(ev)), Member: string(raw)}).Err(); err != nil {
		return
	}
	// Keep the newest bufferMax entries and refresh the TTL.
	_ = rdb.ZRemRangeByRank(ctx, key, 0, int64(-(bufferMax + 1))).Err()
	_ = rdb.Expire(ctx, key, bufferTTL).Err()
}

// Since returns the events recorded after lastID, oldest first. An empty or
// unparseable lastID returns the whole buffer.
func (b *Buffer) Since(ctx context.Context, userID uuid.UUID, lastID string) []contracts.Event {
	if !b.Enabled() {
		return nil
	}
	key := bufferKey(userID)
	rdb := b.client.Raw()

	// Start from the lastID's timestamp when known so Redis does the filtering;
	// fall back to the whole buffer otherwise.
	minScore := "-inf"
	if nanos, ok := idNanos(lastID); ok {
		minScore = strconv.FormatInt(nanos, 10)
	}

	raw, err := rdb.ZRangeByScore(ctx, key, &goredis.ZRangeBy{Min: minScore, Max: "+inf"}).Result()
	if err != nil {
		return nil
	}

	out := make([]contracts.Event, 0, len(raw))
	seenLast := lastID == ""
	for _, item := range raw {
		var ev contracts.Event
		if err := json.Unmarshal([]byte(item), &ev); err != nil {
			continue
		}
		if !seenLast {
			// Skip up to and including the client's last seen event. Comparing
			// ids (not just scores) keeps this exact even when two events share
			// a timestamp.
			if ev.ID == lastID {
				seenLast = true
			}
			continue
		}
		out = append(out, ev)
	}
	return out
}

// ─── id helpers ───────────────────────────────────────────────

// NewEventID builds a monotonic-ish event id: evt_<unixnano>_<seq>.
func NewEventID(nanos int64, seq uint64) string {
	return "evt_" + strconv.FormatInt(nanos, 10) + "_" + strconv.FormatUint(seq, 10)
}

// idNanos extracts the nanosecond component from an event id.
func idNanos(id string) (int64, bool) {
	if id == "" {
		return 0, false
	}
	parts := strings.Split(id, "_")
	if len(parts) < 3 {
		return 0, false
	}
	n, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// eventNanos derives the buffer score for an event: the id's timestamp when
// present, else now.
func eventNanos(ev contracts.Event) int64 {
	if n, ok := idNanos(ev.ID); ok {
		return n
	}
	return time.Now().UnixNano()
}
