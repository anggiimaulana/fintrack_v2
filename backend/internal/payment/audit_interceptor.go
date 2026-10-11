package payment

import (
	"context"

	"github.com/google/uuid"
)

// AuditEvent is a financial audit record.
type AuditEvent struct {
	ActorID   *uuid.UUID
	Action    string
	Entity    string
	EntityID  string
	Before    map[string]any
	After     map[string]any
	IP        string
	UserAgent string
}

// AuditRecorder records financial audit events (bridged to internal/audit).
type AuditRecorder interface {
	Record(ctx context.Context, e AuditEvent) error
}

// AuditInterceptor is a thin facade over the audit recorder with named helpers
// for the financial events that matter.
type AuditInterceptor struct {
	recorder AuditRecorder
}

// NewAuditInterceptor builds the interceptor.
func NewAuditInterceptor(recorder AuditRecorder) *AuditInterceptor {
	return &AuditInterceptor{recorder: recorder}
}

// Log records a financial action; a nil recorder is a no-op.
func (a *AuditInterceptor) Log(ctx context.Context, e AuditEvent) {
	if a == nil || a.recorder == nil {
		return
	}
	_ = a.recorder.Record(ctx, e)
}

// Named helpers ------------------------------------------------------------

func (a *AuditInterceptor) PaymentCreate(ctx context.Context, paymentID uuid.UUID, amount int64) {
	a.Log(ctx, AuditEvent{Action: "PAYMENT_CREATE", Entity: "payment", EntityID: paymentID.String(), After: map[string]any{"amount_idr": amount}})
}

func (a *AuditInterceptor) WebhookReceive(ctx context.Context, ref string, valid bool) {
	a.Log(ctx, AuditEvent{Action: "WEBHOOK_RECEIVE", Entity: "payment", EntityID: ref, After: map[string]any{"signature_valid": valid}})
}

func (a *AuditInterceptor) PaymentSettled(ctx context.Context, paymentID uuid.UUID, amount int64) {
	a.Log(ctx, AuditEvent{Action: "PAYMENT_SETTLED", Entity: "payment", EntityID: paymentID.String(), After: map[string]any{"amount_idr": amount}})
}

func (a *AuditInterceptor) ManualProofUpload(ctx context.Context, paymentID, userID uuid.UUID) {
	a.Log(ctx, AuditEvent{ActorID: &userID, Action: "MANUAL_PROOF_UPLOAD", Entity: "payment", EntityID: paymentID.String()})
}

func (a *AuditInterceptor) ManualReview(ctx context.Context, paymentID, adminID uuid.UUID, approved bool, reason string) {
	action := "MANUAL_REJECT"
	after := map[string]any{"approved": false, "reason": reason}
	if approved {
		action = "MANUAL_APPROVE"
		after = map[string]any{"approved": true}
	}
	a.Log(ctx, AuditEvent{ActorID: &adminID, Action: action, Entity: "payment", EntityID: paymentID.String(), After: after})
}

func (a *AuditInterceptor) InvoiceVoid(ctx context.Context, invoiceID uuid.UUID) {
	a.Log(ctx, AuditEvent{Action: "INVOICE_VOID", Entity: "invoice", EntityID: invoiceID.String()})
}
