package admin

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/pkg/response"
)

// ─── Audit trail viewer (P06-017) ─────────────────────────────
//
// audit_logs is append-only (enforced by trigger), so this surface is
// read-only: list with filters, detail, and CSV export for compliance.

type auditLogDTO struct {
	ID        string         `json:"id"`
	ActorID   *string        `json:"actor_id,omitempty"`
	ActorName *string        `json:"actor_name,omitempty"`
	ActorMail *string        `json:"actor_email,omitempty"`
	Action    string         `json:"action"`
	Entity    string         `json:"entity"`
	EntityID  *string        `json:"entity_id,omitempty"`
	Before    map[string]any `json:"before,omitempty"`
	After     map[string]any `json:"after,omitempty"`
	IP        *string        `json:"ip,omitempty"`
	UserAgent *string        `json:"user_agent,omitempty"`
	RequestID *string        `json:"request_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type auditFilter struct {
	actorID  *uuid.UUID
	email    string
	action   string
	entity   string
	entityID string
	from     *time.Time
	to       *time.Time
}

// buildAuditWhere assembles the WHERE clause + args for the viewer query.
// Actor email and action are compared case-insensitively.
func buildAuditWhere(f auditFilter) (string, []any) {
	var sb strings.Builder
	args := []any{}
	add := func(clause string, v any) {
		args = append(args, v)
		sb.WriteString(" AND " + clause + " $" + strconv.Itoa(len(args)))
	}
	addExpr := func(expr string, v any) {
		args = append(args, v)
		sb.WriteString(" AND " + strings.Replace(expr, "?", "$"+strconv.Itoa(len(args)), 1))
	}
	sb.WriteString(" WHERE 1=1")
	if f.actorID != nil {
		add("a.actor_id =", *f.actorID)
	}
	if f.email != "" {
		addExpr("u.email ILIKE ('%' || ? || '%')", f.email)
	}
	if f.action != "" {
		addExpr("lower(a.action) = lower(?)", f.action)
	}
	if f.entity != "" {
		add("a.entity =", f.entity)
	}
	if f.entityID != "" {
		add("a.entity_id =", f.entityID)
	}
	if f.from != nil {
		add("a.created_at >=", *f.from)
	}
	if f.to != nil {
		add("a.created_at <=", *f.to)
	}
	return sb.String(), args
}

// auditFilterFrom reads the TASK-P06-018 filter set: start_date/end_date
// (RFC3339 or YYYY-MM-DD), user_id, email, entity, action.
func auditFilterFrom(c *gin.Context) (auditFilter, bool) {
	var f auditFilter
	for _, key := range []string{"user_id", "actor_id"} {
		if raw := c.Query(key); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				response.BadRequest(c, key+" tidak valid", "validation")
				return f, false
			}
			f.actorID = &id
			break
		}
	}
	f.email = strings.TrimSpace(c.Query("email"))
	f.action = strings.TrimSpace(c.Query("action"))
	f.entity = strings.TrimSpace(c.Query("entity"))
	f.entityID = strings.TrimSpace(c.Query("entity_id"))
	f.from, f.to = dateRange(c)
	return f, true
}

// ListAuditLogs handles GET /admin/audit-logs (append-only trail, read-only).
//
// Sensitive fields are masked before persistence by internal/audit, so the
// stored before/after states are already PDP-safe (K-30).
func (h *Handler) ListAuditLogs(c *gin.Context) {
	f, ok := auditFilterFrom(c)
	if !ok {
		return
	}
	page, perPage := paginationParams(c)
	offset := (page - 1) * perPage

	where, args := buildAuditWhere(f)
	ctx := c.Request.Context()

	var total int64
	if err := h.svc.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs a LEFT JOIN users u ON u.id = a.actor_id`+where,
		args...).Scan(&total); err != nil {
		response.InternalError(c, "Gagal memuat audit log")
		return
	}

	q := `SELECT a.id, a.actor_id, u.full_name, u.email, a.action, a.entity, a.entity_id,
	             a.before, a.after, a.ip, a.user_agent, a.request_id, a.created_at
	      FROM audit_logs a LEFT JOIN users u ON u.id = a.actor_id` + where +
		` ORDER BY ` + orderBy(c, map[string]string{
		"created_at": "a.created_at",
		"action":     "a.action",
		"entity":     "a.entity",
		"actor":      "u.email",
	}, "a.created_at DESC") + ` LIMIT $` + strconv.Itoa(len(args)+1) +
		` OFFSET $` + strconv.Itoa(len(args)+2)
	rows, err := h.svc.pool.Query(ctx, q, append(args, perPage, offset)...)
	if err != nil {
		response.InternalError(c, "Gagal memuat audit log")
		return
	}
	defer rows.Close()

	out := []auditLogDTO{}
	for rows.Next() {
		var (
			d      auditLogDTO
			actor  *uuid.UUID
			before []byte
			after  []byte
		)
		if err := rows.Scan(&d.ID, &actor, &d.ActorName, &d.ActorMail, &d.Action, &d.Entity,
			&d.EntityID, &before, &after, &d.IP, &d.UserAgent, &d.RequestID, &d.CreatedAt); err != nil {
			response.InternalError(c, "Gagal memuat audit log")
			return
		}
		if actor != nil {
			s := actor.String()
			d.ActorID = &s
		}
		d.Before = decodeJSONMap(before)
		d.After = decodeJSONMap(after)
		out = append(out, d)
	}
	response.Paginated(c, "Audit log berhasil dimuat", out, int(total), page, perPage)
}

// ─── helpers ──────────────────────────────────────────────────

// decodeJSONMap unmarshals a JSONB column into a map, tolerating NULL/invalid.
func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}
