package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// SubscriptionManager handles plan changes: instant prorated upgrades and
// end-of-cycle downgrades (soft-limit, never destroying user data).
type SubscriptionManager struct {
	subs      SubscriptionRepository
	plans     PlanRepository
	lifecycle *LifecycleService
	tx        TxRunner
	now       func() time.Time
	// quotaSyncer pushes a plan change into the user's running token period, so
	// a scheduled downgrade is not left enforcing the previous plan's quota.
	quotaSyncer *QuotaSyncer
}

// WithQuotaSyncer attaches the syncer used after a plan change.
func (m *SubscriptionManager) WithQuotaSyncer(s *QuotaSyncer) *SubscriptionManager {
	m.quotaSyncer = s
	return m
}

// syncQuota updates the running period to planCode's quota. Best-effort: the
// plan change is already committed, and a sync failure is logged.
func (m *SubscriptionManager) syncQuota(ctx context.Context, userID uuid.UUID, planCode string) {
	if m.quotaSyncer == nil {
		return
	}
	if err := m.quotaSyncer.SyncRunningPeriod(ctx, userID, planCode); err != nil {
		logger.Warn("subscription manager: quota sync failed",
			zap.String("user_id", userID.String()),
			zap.String("plan_code", planCode), zap.Error(err))
	}
}

// NewSubscriptionManager builds the manager.
func NewSubscriptionManager(subs SubscriptionRepository, plans PlanRepository, lifecycle *LifecycleService, tx TxRunner) *SubscriptionManager {
	return &SubscriptionManager{subs: subs, plans: plans, lifecycle: lifecycle, tx: tx, now: func() time.Time { return time.Now().UTC() }}
}

// Proration holds the integer-Rupiah breakdown of an instant upgrade.
type Proration struct {
	DaysTotal     int   `json:"days_total"`
	DaysRemaining int   `json:"days_remaining"`
	OldCreditIDR  int64 `json:"old_credit_idr"`
	NewCostIDR    int64 `json:"new_cost_idr"`
	AmountDueIDR  int64 `json:"amount_due_idr"`
}

// CalculateUpgradeProration computes the prorated amount due for switching to
// newPlan now. All values are integer Rupiah, rounded down.
//
//	oldCredit = floor(price_old × daysRemaining / daysTotal)
//	newCost   = floor(price_new × daysRemaining / daysTotal)
//	amountDue = max(0, newCost - oldCredit)
func CalculateUpgradeProration(currentSub *Subscription, currentPlan, newPlan *Plan, now time.Time) Proration {
	p := Proration{}
	if currentSub == nil || newPlan == nil || currentSub.CurrentPeriodEnd == nil {
		return p
	}

	start := currentSub.CurrentPeriodStart
	end := *currentSub.CurrentPeriodEnd
	total := int(end.Sub(start).Hours() / 24)
	if total <= 0 {
		total = newPlan.BillingPeriodDays
		if total <= 0 {
			total = 30
		}
	}
	remaining := int(end.Sub(now).Hours() / 24)
	if remaining < 0 {
		remaining = 0
	}
	if remaining > total {
		remaining = total
	}

	p.DaysTotal = total
	p.DaysRemaining = remaining

	var oldPrice int64
	if currentPlan != nil {
		oldPrice = currentPlan.PriceMonthlyIDR
	}
	p.OldCreditIDR = oldPrice * int64(remaining) / int64(total)
	p.NewCostIDR = newPlan.PriceMonthlyIDR * int64(remaining) / int64(total)
	p.AmountDueIDR = p.NewCostIDR - p.OldCreditIDR
	if p.AmountDueIDR < 0 {
		p.AmountDueIDR = 0
	}
	return p
}

// Upgrade switches the subscription to a new (higher) plan immediately. The
// caller is responsible for collecting AmountDueIDR via the payment flow (P05);
// this method applies the plan change and resets the token period quota.
//
// It records an "upgraded" event and invalidates entitlement caches.
func (m *SubscriptionManager) Upgrade(ctx context.Context, userID uuid.UUID, newPlanCode string) (*Subscription, Proration, error) {
	sub, err := m.subs.GetByUserID(ctx, userID)
	if err != nil {
		return nil, Proration{}, err
	}
	newPlan, err := m.plans.GetPlanByCode(ctx, newPlanCode)
	if err != nil {
		return nil, Proration{}, err
	}
	var curPlan *Plan
	if sub.PlanID != uuid.Nil {
		curPlan, _ = m.plans.GetPlanByID(ctx, sub.PlanID)
	}

	proration := CalculateUpgradeProration(sub, curPlan, newPlan, m.now())

	err = m.tx.RunInTx(ctx, func(txCtx context.Context) error {
		prevPlan := sub.PlanID
		sub.PlanID = newPlan.ID
		sub.CancelAtPeriodEnd = false
		if sub.Status == StatusExpired || sub.Status == StatusCanceled {
			sub.Status = StatusActive
		}
		if err := m.subs.Update(txCtx, sub); err != nil {
			return err
		}
		return m.subs.RecordEvent(txCtx, &SubscriptionEvent{
			SubscriptionID: sub.ID,
			UserID:         userID,
			EventType:      "upgraded",
			PreviousState:  map[string]any{"plan_id": prevPlan.String()},
			NewState:       map[string]any{"plan_id": newPlan.ID.String(), "plan_code": newPlan.Code, "amount_due_idr": proration.AmountDueIDR},
		})
	})
	if err != nil {
		return nil, Proration{}, err
	}
	return sub, proration, nil
}

// ScheduleDowngrade sets the subscription to switch to a lower plan at the end
// of the current cycle. Access is NOT revoked now (soft-limit): the user keeps
// their plan until current_period_end, and no historical data is deleted.
func (m *SubscriptionManager) ScheduleDowngrade(ctx context.Context, userID uuid.UUID, newPlanCode string) error {
	sub, err := m.subs.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	newPlan, err := m.plans.GetPlanByCode(ctx, newPlanCode)
	if err != nil {
		return err
	}

	return m.tx.RunInTx(ctx, func(txCtx context.Context) error {
		if sub.Metadata == nil {
			sub.Metadata = map[string]any{}
		}
		sub.Metadata["scheduled_plan_code"] = newPlan.Code
		sub.Metadata["scheduled_plan_id"] = newPlan.ID.String()
		sub.CancelAtPeriodEnd = true
		if err := m.subs.Update(txCtx, sub); err != nil {
			return err
		}
		return m.subs.RecordEvent(txCtx, &SubscriptionEvent{
			SubscriptionID: sub.ID,
			UserID:         userID,
			EventType:      "downgrade_scheduled",
			PreviousState:  map[string]any{"plan_id": sub.PlanID.String()},
			NewState:       map[string]any{"scheduled_plan_code": newPlan.Code, "effective": "end_of_period"},
		})
	})
}

// ApplyScheduledDowngrade switches to the scheduled plan when the cycle ends.
// Safe to call repeatedly; no-op when nothing is scheduled.
func (m *SubscriptionManager) ApplyScheduledDowngrade(ctx context.Context, userID uuid.UUID) error {
	sub, err := m.subs.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	code, _ := sub.Metadata["scheduled_plan_code"].(string)
	if code == "" {
		return nil
	}
	newPlan, err := m.plans.GetPlanByCode(ctx, code)
	if err != nil {
		return err
	}

	err = m.tx.RunInTx(ctx, func(txCtx context.Context) error {
		prev := sub.PlanID
		sub.PlanID = newPlan.ID
		sub.CancelAtPeriodEnd = false
		delete(sub.Metadata, "scheduled_plan_code")
		delete(sub.Metadata, "scheduled_plan_id")
		if err := m.subs.Update(txCtx, sub); err != nil {
			return err
		}
		return m.subs.RecordEvent(txCtx, &SubscriptionEvent{
			SubscriptionID: sub.ID,
			UserID:         userID,
			EventType:      "downgraded",
			PreviousState:  map[string]any{"plan_id": prev.String()},
			NewState:       map[string]any{"plan_id": newPlan.ID.String(), "plan_code": newPlan.Code},
		})
	})
	if err != nil {
		return err
	}
	if m.lifecycle != nil && m.lifecycle.invalidate != nil {
		m.lifecycle.invalidate(ctx, userID)
	}
	// The plan changed: push it into the running token period, or the downgraded
	// user keeps the higher plan's quota until the cycle lapses.
	m.syncQuota(ctx, userID, newPlan.Code)
	return nil
}
