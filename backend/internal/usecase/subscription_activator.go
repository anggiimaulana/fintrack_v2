package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/entitlement"
)

// SubscriptionActivator bridges successful payments to the billing module:
// it activates or renews a user's subscription on the paid plan and invalidates
// entitlement caches so the change applies immediately.
type SubscriptionActivator struct {
	subs      billing.SubscriptionRepository
	plans     billing.PlanRepository
	lifecycle *billing.LifecycleService
	ent       *entitlement.Resolver
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

// ActivateForUser ensures the user has an active subscription on planCode,
// renewing the current period. If the user has no subscription, one is created.
func (a *SubscriptionActivator) ActivateForUser(ctx context.Context, userID uuid.UUID, planCode string) error {
	if planCode == "" {
		// No plan recorded: nothing to activate.
		return nil
	}
	plan, err := a.plans.GetPlanByCode(ctx, planCode)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	end := now.AddDate(0, 0, plan.BillingPeriodDays)

	sub, err := a.subs.GetByUserID(ctx, userID)
	if err == billing.ErrSubscriptionNotFound || sub == nil {
		// Create a fresh active subscription.
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
		if a.ent != nil {
			a.ent.Invalidate(ctx, userID)
		}
		return nil
	}
	if err != nil {
		return err
	}

	// Renew the existing subscription on the paid plan.
	sub.PlanID = plan.ID
	sub.Status = billing.StatusActive
	sub.CancelAtPeriodEnd = false
	sub.CurrentPeriodStart = now
	sub.CurrentPeriodEnd = &end
	if uerr := a.subs.Update(ctx, sub); uerr != nil {
		return uerr
	}
	if a.ent != nil {
		a.ent.Invalidate(ctx, userID)
	}
	return nil
}
