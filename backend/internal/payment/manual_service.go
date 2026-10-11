package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ManualService handles the manual-payment flow: uploading proof and the admin
// review (approve/reject).
type ManualService struct {
	repo     Repository
	activate SubscriptionActivator
	tx       TxRunner
	// store persists proof bytes and returns a retrievable URL/path.
	store ProofStore
	// emitter publishes realtime domain events (never nil; no-op by default).
	emitter RealtimeEmitter
}

// ProofStore persists uploaded proof files.
type ProofStore interface {
	Save(ctx context.Context, relPath string, content []byte) (string, error)
}

// NewManualService builds the manual service.
func NewManualService(repo Repository, activate SubscriptionActivator, tx TxRunner, store ProofStore) *ManualService {
	return &ManualService{repo: repo, activate: activate, tx: tx, store: store, emitter: noopEmitter{}}
}

// UploadProof validates and stores a proof file, then moves the payment to
// pending_review. Ownership is enforced by the caller (handler).
func (s *ManualService) UploadProof(ctx context.Context, paymentID, userID uuid.UUID, content []byte, filename string) (*Payment, error) {
	pay, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if pay == nil || pay.UserID != userID {
		return nil, ErrPaymentNotFound
	}
	if pay.Gateway != "manual" {
		return nil, fmt.Errorf("%w: payment is not manual", ErrInvalidState)
	}
	if pay.Status.IsFinal() {
		return nil, ErrInvalidState
	}

	_, ext, err := ValidateProof(content, filename)
	if err != nil {
		return nil, err
	}

	relPath := SafeProofFilename(paymentID.String(), ext)
	var url string
	if s.store != nil {
		url, err = s.store.Save(ctx, relPath, content)
		if err != nil {
			return nil, err
		}
	} else {
		url = relPath
	}

	if err := s.repo.SetPaymentProof(ctx, paymentID, url); err != nil {
		return nil, err
	}
	pay.ProofURL = &url
	pay.Status = PaymentPendingReview

	// Tell admins there is a transfer to review. Best-effort: a failure here
	// must not fail the upload (the payment row is already updated).
	emitManualApprovalNeeded(ctx, s.emitter, pay, "", "")
	return pay, nil
}

// Approve settles a manual payment and activates the subscription (idempotent
// under a row lock).
func (s *ManualService) Approve(ctx context.Context, paymentID uuid.UUID) (*Payment, error) {
	return s.review(ctx, paymentID, true, "")
}

// Reject marks a manual payment as failed with a required reason.
func (s *ManualService) Reject(ctx context.Context, paymentID uuid.UUID, reason string) (*Payment, error) {
	if reason == "" {
		return nil, fmt.Errorf("rejection reason is required")
	}
	return s.review(ctx, paymentID, false, reason)
}

func (s *ManualService) review(ctx context.Context, paymentID uuid.UUID, approve bool, reason string) (*Payment, error) {
	var out *Payment
	var approved *Payment
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Row lock: concurrent admins reviewing the same payment serialize here.
		// The loser re-reads the committed state and fails with ErrInvalidState.
		pay, err := s.repo.GetPaymentByIDForUpdate(txCtx, paymentID)
		if err != nil {
			return err
		}
		if pay == nil {
			return ErrPaymentNotFound
		}
		if pay.Status != PaymentPendingReview {
			return fmt.Errorf("%w: expected pending_review, got %s", ErrInvalidState, pay.Status)
		}

		if approve {
			if err := s.repo.UpdatePaymentStatus(txCtx, pay.ID, PaymentSettled, nil, nil, map[string]any{"approved": true}); err != nil {
				return err
			}
			if err := s.repo.MarkInvoicePaid(txCtx, pay.InvoiceID); err != nil {
				return err
			}
			if s.activate != nil {
				inv, ierr := s.repo.GetInvoiceByID(txCtx, pay.InvoiceID)
				if ierr == nil && inv != nil {
					if aerr := s.activate.ActivateForUser(txCtx, pay.UserID, planCodeFromInvoice(inv)); aerr != nil {
						return aerr
					}
				}
			}
			pay.Status = PaymentSettled
			approved = pay
		} else {
			if err := s.repo.UpdatePaymentStatus(txCtx, pay.ID, PaymentFailed, nil, nil, map[string]any{"rejected": true, "reason": reason}); err != nil {
				return err
			}
			pay.Status = PaymentFailed
		}
		out = pay
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Emitted after commit so no event escapes for a rolled-back approval.
	if approved != nil {
		emitPaymentReceived(ctx, s.emitter, approved, "")
	}
	return out, nil
}

// AdminService exposes admin review operations (thin wrapper for clarity).
type AdminService struct {
	manual *ManualService
}

// NewAdminService builds the admin service.
func NewAdminService(manual *ManualService) *AdminService {
	return &AdminService{manual: manual}
}

// Approve approves a manual payment.
func (a *AdminService) Approve(ctx context.Context, paymentID uuid.UUID) (*Payment, error) {
	return a.manual.Approve(ctx, paymentID)
}

// Reject rejects a manual payment.
func (a *AdminService) Reject(ctx context.Context, paymentID uuid.UUID, reason string) (*Payment, error) {
	return a.manual.Reject(ctx, paymentID, reason)
}

// ListPending returns payments awaiting review.
func (a *AdminService) ListPending(ctx context.Context, limit int) ([]*Payment, error) {
	return a.manual.repo.ListPaymentsPendingReview(ctx, limit)
}
