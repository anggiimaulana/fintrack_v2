// Package contracts holds cross-module data contracts. It is deliberately a
// leaf package (no imports from internal/) so producers in any module — payment,
// billing, aigateway, admin — can emit domain events without importing the
// realtime transport and creating an import cycle.
package contracts

import "encoding/json"

// Domain event names. These are the wire values carried in the SSE `event:`
// field and in the envelope's `event` property; the frontend dispatcher keys
// off them, so they are part of the public contract.
const (
	EventPaymentReceived      = "payment_received"
	EventPlanUpdated          = "plan_updated"
	EventAIEndpointDown       = "ai_endpoint_down"
	EventAIRouteQuarantined   = "ai_route_quarantined"
	EventManualApprovalNeeded = "manual_approval_needed"
	EventQuotaReset           = "quota_reset"
	// UsageChanged is emitted per-user whenever real-time quota consumption
	// moves (a transaction was created/deleted, or AI tokens/requests were
	// consumed). The client refreshes ONLY its entitlements query — never the
	// transaction/dashboard/report lists — so the quota bars stay live without a
	// cache storm.
	EventUsageChanged = "usage_changed"
	// Notification-inbox lifecycle events. Emitted per-user so the notifications
	// page can patch a single row instead of refetching the whole list.
	EventNotificationCreated = "notification_created"
	EventNotificationRead    = "notification_read"
	EventNotificationDeleted = "notification_deleted"
	// ConfigChanged is emitted when the configuration catalog changes (a
	// setting, metric, feature, plan, or pricing rule). Clients reload the
	// affected queries immediately instead of waiting for a TTL, so an operator
	// sees their change take effect at once (P-13).
	EventConfigChanged = "config_changed"
	// EventInvoiceIssued announces a newly issued invoice (an automatic renewal
	// invoice, or one created at checkout). The user is told to pay before the
	// due date so the subscription does not lapse.
	EventInvoiceIssued = "invoice_issued"
	EventUserStatusUpdated = "user_status_updated"
	EventUserCreated       = "user_created"
	EventRoleUpdated       = "role_updated"
	EventTokensAdjusted    = "tokens_adjusted"
)

// Event is the envelope every domain event shares. Field order and names are
// part of the wire contract.
type Event struct {
	// Event is the event name (one of the Event* constants).
	Event string `json:"event"`
	// Timestamp is the emission time in Unix seconds.
	Timestamp int64 `json:"timestamp"`
	// ID is a monotonically increasing event id, used for SSE resume
	// (`Last-Event-ID`). Format: evt_<unixnano>_<seq>.
	ID string `json:"id"`
	// Payload is the event-specific body. Kept as raw JSON so producers can use
	// their own structs without this package depending on them.
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ─── Payloads ─────────────────────────────────────────────────

// PaymentReceivedPayload accompanies EventPaymentReceived.
type PaymentReceivedPayload struct {
	PaymentID     string `json:"payment_id"`
	InvoiceID     string `json:"invoice_id,omitempty"`
	InvoiceNumber string `json:"invoice_number,omitempty"`
	AmountIDR     int64  `json:"amount_idr"`
	Status        string `json:"status"`
}

// PlanUpdatedPayload accompanies EventPlanUpdated.
type PlanUpdatedPayload struct {
	UserID       string `json:"user_id"`
	PreviousPlan string `json:"previous_plan,omitempty"`
	CurrentPlan  string `json:"current_plan"`
	PlanID       string `json:"plan_id,omitempty"`
	// PlanWide is true when the change affected every subscriber of a plan
	// (plan entitlements edited) rather than one user's subscription.
	PlanWide bool `json:"plan_wide,omitempty"`
}

// AIEndpointDownPayload accompanies EventAIEndpointDown.
type AIEndpointDownPayload struct {
	EndpointID     string `json:"endpoint_id"`
	FailureCount   int    `json:"failure_count,omitempty"`
	Status         string `json:"status"`
	FallbackTarget string `json:"fallback_target,omitempty"`
}

// AIRouteQuarantinedPayload accompanies EventAIRouteQuarantined.
//
// It is emitted by the smart-failover route breaker when a single (endpoint,
// model) route is quarantined after repeated failures, and again (with
// Recovered=true) when the route starts working again after its cooldown.
type AIRouteQuarantinedPayload struct {
	UseCase    string `json:"use_case"`
	Model      string `json:"model"`
	Endpoint   string `json:"endpoint,omitempty"`
	EndpointID string `json:"endpoint_id,omitempty"`
	Combo      string `json:"combo,omitempty"`
	// Reason is the last upstream error, when known.
	Reason string `json:"reason,omitempty"`
	// Recovered is true when the route left quarantine (started working again).
	Recovered bool `json:"recovered,omitempty"`
}

// ManualApprovalNeededPayload accompanies EventManualApprovalNeeded.
type ManualApprovalNeededPayload struct {
	PaymentID  string `json:"payment_id"`
	UserEmail  string `json:"user_email,omitempty"`
	AmountIDR  int64  `json:"amount_idr"`
	InvoiceNum string `json:"invoice_number,omitempty"`
}

// QuotaResetPayload accompanies EventQuotaReset.
type QuotaResetPayload struct {
	UserID          string `json:"user_id"`
	PeriodStart     int64  `json:"new_cycle_start"`
	PeriodEnd       int64  `json:"new_cycle_end"`
	AllocatedTokens int64  `json:"allocated_tokens"`
}

// InvoiceIssuedPayload accompanies EventInvoiceIssued.
type InvoiceIssuedPayload struct {
	InvoiceID     string `json:"invoice_id"`
	InvoiceNumber string `json:"invoice_number"`
	AmountIDR     int64  `json:"amount_idr"`
	PlanName      string `json:"plan_name,omitempty"`
	// DueAt is the payment deadline (Unix seconds), so the client can render a
	// countdown without a second fetch.
	DueAt int64 `json:"due_at,omitempty"`
	// Renewal marks an automatically issued renewal invoice (vs a checkout).
	Renewal bool `json:"renewal"`
}

// ConfigChangedPayload accompanies EventConfigChanged. It names WHAT changed so
// the client reloads only the affected queries (never a blanket refetch).
type ConfigChangedPayload struct {
	// Scope is the configuration area: "settings" | "metrics" | "features" |
	// "plans" | "pricing" | "all".
	Scope string `json:"scope"`
	// Keys lists the specific entries that changed (empty = the whole scope).
	Keys []string `json:"keys,omitempty"`
}

// UsageChangedPayload accompanies EventUsageChanged. It is a lightweight hint,
// not a full snapshot: the client reacts by refetching GET /me/entitlements
// (the authoritative, already-existing usage source), so the payload only needs
// to say *what* moved — not the new totals.
type UsageChangedPayload struct {
	// LimitKeys lists the limit keys whose consumption changed (e.g.
	// "max_transactions", "ai_tokens_per_cycle"). Clients that patch in place
	// can use it to decide which bars to touch; clients that refetch ignore it.
	LimitKeys []string `json:"limit_keys"`
	// Reason is a short, human-readable origin tag for observability
	// (e.g. "transaction_created", "ai_chat").
	Reason string `json:"reason,omitempty"`
}

// NotificationChangedPayload accompanies the notification-inbox lifecycle events
// (created / read / deleted). It carries the affected rows so the client can
// patch its cache in place:
//   - created: Item holds the full new notification.
//   - read:    IDs lists the rows marked read (one or many).
//   - deleted: IDs lists the rows removed (one or many).
//
// A client that cannot patch (e.g. it does not have the row cached) falls back to
// a targeted refetch, which is why IDs is always present.
type NotificationChangedPayload struct {
	IDs  []string      `json:"ids"`
	Item *Notification `json:"item,omitempty"`
	// Unread is the user's unread count after the change, so the bell badge can
	// update without a separate request.
	Unread int `json:"unread"`
}

// Notification mirrors domain.Notification on the wire for realtime payloads.
// Declared here (not imported from internal/domain) to keep contracts a leaf
// package.
type Notification struct {
	ID        string  `json:"id"`
	Event     string  `json:"event"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	Href      string  `json:"href"`
	ReadAt    *string `json:"read_at,omitempty"`
	CreatedAt string  `json:"created_at"`
}

// NewEvent builds an envelope for the given name, marshalling the payload.
// A payload that cannot be marshalled is omitted rather than failing the emit —
// realtime is an accelerator, not the source of truth, so a malformed payload
// must not break the caller's primary operation.
func NewEvent(name string, ts int64, payload any) Event {
	ev := Event{Event: name, Timestamp: ts}
	if payload != nil {
		if raw, err := json.Marshal(payload); err == nil {
			ev.Payload = raw
		}
	}
	return ev
}
