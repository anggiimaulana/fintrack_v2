package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InvalidateFunc is called after a lifecycle change to refresh entitlement
// caches (implemented by the entitlement broadcaster).
type InvalidateFunc func(ctx context.Context, userID uuid.UUID)

// LifecycleService applies legal subscription state transitions and records an
// append-only audit event for each one.
type LifecycleService struct {
	subs       SubscriptionRepository
	tx         TxRunner
	invalidate InvalidateFunc
	now        func() time.Time
}

// NewLifecycleService builds the service. invalidate may be nil.
func NewLifecycleService(subs SubscriptionRepository, tx TxRunner, invalidate InvalidateFunc) *LifecycleService {
	return &LifecycleService{subs: subs, tx: tx, invalidate: invalidate, now: func() time.Time { return time.Now().UTC() }}
}

// allowedTransitions encodes the legal state machine.
//
// `active -> expired` is legal because a subscription whose cycle has ended and
// whose grace window has lapsed must lapse directly: there is nothing to collect
// (no open invoice) and the user falls back to the default plan. Requiring a
// detour through past_due would only add a transient state nobody observes.
var allowedTransitions = map[SubscriptionStatus]map[SubscriptionStatus]bool{
	StatusTrialing: {StatusActive: true, StatusExpired: true},
	StatusActive:   {StatusPastDue: true, StatusCanceled: true, StatusExpired: true},
	StatusPastDue:  {StatusActive: true, StatusGrace: true, StatusExpired: true},
	StatusGrace:    {StatusActive: true, StatusExpired: true},
	StatusCanceled: {StatusExpired: true},
	StatusExpired:  {}, // terminal
}

// CanTransition reports whether from -> to is a legal transition.
func CanTransition(from, to SubscriptionStatus) bool {
	if m, ok := allowedTransitions[from]; ok {
		return m[to]
	}
	return false
}

// TransitionByUserID loads the user's subscription and applies a transition.
// The lifecycle worker works from a due-list (user ids), so it needs this
// idempotent, load-then-transition entry point. An already-terminal or
// already-target status is a no-op, not an error: a sweep may observe a
// subscription another worker just settled.
func (s *LifecycleService) TransitionByUserID(ctx context.Context, userID uuid.UUID, to SubscriptionStatus, eventType, reason string) error {
	sub, err := s.subs.GetByUserID(ctx, userID)
	if err != nil {
		if err == ErrSubscriptionNotFound {
			return nil // no row: nothing to transition
		}
		return err
	}
	if sub == nil || sub.Status == to {
		return nil
	}
	if !CanTransition(sub.Status, to) {
		// Already moved past this state (e.g. expired by another sweep).
		return nil
	}
	return s.Transition(ctx, sub, to, eventType, reason)
}

// Transition moves a subscription to a new status, validating the transition and
// recording an audit event. It invalidates entitlement caches on success.
func (s *LifecycleService) Transition(ctx context.Context, sub *Subscription, to SubscriptionStatus, eventType, reason string) error {
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	if !CanTransition(sub.Status, to) {
		return ErrInvalidStateTransition
	}

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		prev := sub.Status
		sub.Status = to
		if to == StatusCanceled {
			t := s.now()
			sub.CanceledAt = &t
		}
		if err := s.subs.Update(txCtx, sub); err != nil {
			return err
		}
		ev := &SubscriptionEvent{
			SubscriptionID: sub.ID,
			UserID:         sub.UserID,
			EventType:      eventType,
			PreviousState:  map[string]any{"status": string(prev)},
			NewState:       map[string]any{"status": string(to)},
		}
		if reason != "" {
			ev.Reason = &reason
		}
		return s.subs.RecordEvent(txCtx, ev)
	})
	if err != nil {
		return err
	}
	if s.invalidate != nil {
		s.invalidate(ctx, sub.UserID)
	}
	return nil
}

// Renew advances the subscription to a fresh cycle (period = plan.billing days).
// Used by the renewal flow (P05) and by manual/auto extension.
func (s *LifecycleService) Renew(ctx context.Context, sub *Subscription, plan *Plan, from time.Time) error {
	if sub == nil || plan == nil {
		return ErrSubscriptionNotFound
	}
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		start := from
		end := start.AddDate(0, 0, plan.BillingPeriodDays)
		prevStart := sub.CurrentPeriodStart
		sub.CurrentPeriodStart = start
		sub.CurrentPeriodEnd = &end
		if sub.Status != StatusActive {
			sub.Status = StatusActive
		}
		if err := s.subs.Update(txCtx, sub); err != nil {
			return err
		}
		return s.subs.RecordEvent(txCtx, &SubscriptionEvent{
			SubscriptionID: sub.ID,
			UserID:         sub.UserID,
			EventType:      "renewed",
			PreviousState:  map[string]any{"period_start": prevStart.Format(time.RFC3339)},
			NewState:       map[string]any{"period_start": start.Format(time.RFC3339), "period_end": end.Format(time.RFC3339)},
		})
	})
	if err != nil {
		return err
	}
	if s.invalidate != nil {
		s.invalidate(ctx, sub.UserID)
	}
	return nil
}
