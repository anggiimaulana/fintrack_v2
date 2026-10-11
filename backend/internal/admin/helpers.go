package admin

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fintrack/backend/internal/audit"
	"github.com/fintrack/backend/pkg/response"
)

// contextT aliases context.Context for terse signatures.
type contextT = context.Context

// querier is the read subset shared by *pgxpool.Pool and pgx.Tx.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// parseUUIDParam reads a UUID path parameter, writing a 400 on malformed input.
func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		response.BadRequest(c, "ID tidak valid", "invalid_id")
		return uuid.Nil, false
	}
	return id, true
}

// actorContext captures the request-scoped audit metadata.
type actorContext struct {
	actorID   *uuid.UUID
	ip        string
	userAgent string
	requestID string
}

// jsonOrEmpty marshals a map to JSONB bytes, defaulting to an empty object.
func jsonOrEmpty(m map[string]any) []byte {
	if m == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// actorFrom extracts audit metadata from the gin context.
func actorFrom(c *gin.Context) actorContext {
	a := actorContext{ip: c.ClientIP(), userAgent: c.Request.UserAgent()}
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			a.actorID = &id
		}
	}
	a.requestID = c.GetHeader("X-Request-ID")
	if a.requestID == "" {
		if id, ok := c.Get("request_id"); ok {
			if s, ok := id.(string); ok {
				a.requestID = s
			}
		}
	}
	return a
}

// auditAny records an action whose before/after payloads are arbitrary
// structs, normalised to maps via JSON so the audit repository can persist them.
func (h *Handler) auditAny(c *gin.Context, action, entity, entityID string, before, after any) {
	h.audit(c, action, entity, entityID, toMap(before), toMap(after))
}

// toMap normalises a struct/pointer to map[string]any (nil-safe).
func toMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

// audit records an administrative action with the request's actor metadata.
func (h *Handler) audit(c *gin.Context, action, entity, entityID string, before, after map[string]any) {
	a := actorFrom(c)
	h.svc.record(c.Request.Context(), audit.Event{
		ActorID:   a.actorID,
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		Before:    before,
		After:     after,
		IP:        a.ip,
		UserAgent: a.userAgent,
		RequestID: a.requestID,
	})
}

// orderBy translates the DataTable `sort`/`order` URL params into an ORDER BY
// clause. Only whitelisted expressions reach the SQL string; anything else
// falls back to the endpoint default, so the parameter can never be injected.
func orderBy(c *gin.Context, columns map[string]string, def string) string {
	col, ok := columns[c.Query("sort")]
	if !ok {
		return def
	}
	dir := "DESC"
	if strings.EqualFold(c.Query("order"), "asc") {
		dir = "ASC"
	}
	return col + " " + dir
}

// dateRange reads start_date/end_date (RFC3339 or YYYY-MM-DD) and returns a
// half-open interval [from, to): the end day is inclusive for the user, so a
// bare date is advanced by 24h.
func dateRange(c *gin.Context) (*time.Time, *time.Time) {
	parse := func(raw string, endOfDay bool) *time.Time {
		if raw = strings.TrimSpace(raw); raw == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			t, err = time.Parse("2006-01-02", raw)
			if err != nil {
				return nil
			}
			if endOfDay {
				t = t.Add(24 * time.Hour)
			}
		}
		return &t
	}
	return parse(c.Query("start_date"), false), parse(c.Query("end_date"), true)
}
