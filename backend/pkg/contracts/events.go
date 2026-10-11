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
	EventManualApprovalNeeded = "manual_approval_needed"
	EventQuotaReset           = "quota_reset"
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
