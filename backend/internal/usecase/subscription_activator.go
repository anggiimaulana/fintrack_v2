package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/pkg/logger"
)

// SubscriptionActivator bridges successful payments to the billing module:
// it activates or renews a user's subscription on the paid plan and invalidates
// entitlement caches so the change applies immediately.
type SubscriptionActivator struct {
	subs      billing.SubscriptionRepository
	plans     billing.PlanRepository
	lifecycle *billing.LifecycleService
	ent       *entitlement.Resolver
	// quotaSyncer updates the RUNNING token period when a payment changes the
	// plan, so an upgrade is not capped by the previous plan's quota. Optional.
	quotaSyncer *billing.QuotaSyncer
}

// NewSubscriptionActivator builds the activator.
func NewSubscriptionActivator(subs billing.SubscriptionRepository, plans billing.PlanRepository, ent *entitlement.Resolver) *SubscriptionActivator {
	inv := func(ctx context.Context, userID uuid.UUID) {
		if ent != nil {
			ent.Invalidate(ctx, userID)
		}
	}
	lc := billing.NewLifecycleService(subs, nil, inv)
	return &SubscriptionActivator{subs: subs, plans: plans, lifecycle: lc, ent: ent}
}

// WithQuotaSyncer attaches a syncer so a mid-cycle plan change updates the
// running token period's quota in place.
func (a *SubscriptionActivator) WithQuotaSyncer(s *billing.QuotaSyncer) *SubscriptionActivator {
	a.quotaSyncer = s
	return a
}

// ActivateForUser applies a successful payment to the user's subscription.
//
// The outcome depends on what the payment is FOR, derived from the current
// subscription state:
//
//   - no subscription yet -> create one starting now;
//   - same plan, cycle still running -> RENEW: extend from current_period_end so
//     the paid days stack (paying early never costs the user days they hold);
//   - different plan, cycle still running -> PLAN CHANGE: keep the cycle bounds
//     and only swap the plan. The user paid the prorated difference for the
//     REMAINING days (see CalculateUpgradeProration), so the cycle must not be
//     reset — resetting would grant a fresh full cycle for a prorated price and
//     orphan the old token period;
//   - cycle already lapsed -> start a fresh cycle from now (a past window cannot
//     be extended);
//   - a scheduled downgrade is cleared, since the user just paid for a plan.
func (a *SubscriptionActivator) ActivateForUser(ctx context.Context, userID uuid.UUID, planCode string) error {
	if planCode == "" {
		// No plan recorded: nothing to activate.
		return nil
	}
	plan, err := a.plans.GetPlanByCode(ctx, planCode)
	if err != nil {
		return err
	}
	if plan.BillingPeriodDays <= 0 {
		// Never create a zero-length cycle: fall back to a sane month.
		plan.BillingPeriodDays = 30
	}

	now := time.Now().UTC()

	sub, err := a.subs.GetByUserID(ctx, userID)
	if err == billing.ErrSubscriptionNotFound || sub == nil {
		// Create a fresh active subscription.
		end := now.AddDate(0, 0, plan.BillingPeriodDays)
		newSub := &billing.Subscription{
			UserID:             userID,
			PlanID:             plan.ID,
			Status:             billing.StatusActive,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   &end,
			Gateway:            "manual",
		}
		if cerr := a.subs.Create(ctx, newSub); cerr != nil {
			return cerr
		}
		a.invalidate(ctx, userID)
		return nil
	}
	if err != nil {
		return err
	}

	samePlan := sub.PlanID == plan.ID
	cycleRunning := sub.CurrentPeriodEnd != nil && sub.CurrentPeriodEnd.After(now)

	start := now
	end := now.AddDate(0, 0, plan.BillingPeriodDays)
	switch {
	case cycleRunning && samePlan:
		// Renewal: stack onto the end of the running cycle.
		start = *sub.CurrentPeriodEnd
		end = start.AddDate(0, 0, plan.BillingPeriodDays)
	case cycleRunning:
		// Plan change: keep the cycle bounds, only the plan changes.
		start = sub.CurrentPeriodStart
		end = *sub.CurrentPeriodEnd
	}

	sub.PlanID = plan.ID
	sub.Status = billing.StatusActive
	sub.CancelAtPeriodEnd = false
	sub.CanceledAt = nil
	sub.CurrentPeriodStart = start
	sub.CurrentPeriodEnd = &end
	// Paying for a plan clears any scheduled downgrade.
	if sub.Metadata != nil {
		delete(sub.Metadata, "scheduled_plan_code")
		delete(sub.Metadata, "scheduled_plan_id")
	}
	if uerr := a.subs.Update(ctx, sub); uerr != nil {
		return uerr
	}
	// A plan change must reach the RUNNING token period, or the user stays capped
	// at the previous plan's quota until the cycle lapses. Best-effort: the
	// payment already succeeded, so a sync failure is logged, not fatal.
	if a.quotaSyncer != nil {
		if serr := a.quotaSyncer.SyncRunningPeriod(ctx, userID, planCode); serr != nil {
			logger.Warn("activate: quota sync failed",
				zap.String("user_id", userID.String()),
				zap.String("plan_code", planCode), zap.Error(serr))
		}
	}
	a.invalidate(ctx, userID)
	return nil
}

// invalidate refreshes the entitlement cache after a subscription change.
func (a *SubscriptionActivator) invalidate(ctx context.Context, userID uuid.UUID) {
	if a.ent != nil {
		a.ent.Invalidate(ctx, userID)
	}
}
