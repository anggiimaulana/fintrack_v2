package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// CatalogService exposes the public plan catalog (dynamic, from the DB) so the
// pricing UI no longer hardcodes names, prices, features, or limits.
type CatalogService struct {
	plans     billing.PlanRepository
	entSource PlanEntitlementSource
}

// PlanEntitlementSource reads a plan's features and limits.
type PlanEntitlementSource interface {
	GetEntitlements(ctx context.Context, planID uuid.UUID) (map[string]bool, error)
	GetLimits(ctx context.Context, planID uuid.UUID) (map[string]int64, error)
}

// NewCatalogService builds the catalog service.
func NewCatalogService(plans billing.PlanRepository, entSource PlanEntitlementSource) *CatalogService {
	return &CatalogService{plans: plans, entSource: entSource}
}

// PublicPlan is a plan plus its features/limits, shaped for the pricing UI.
type PublicPlan struct {
	Code              string           `json:"code"`
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	PriceMonthlyIDR   int64            `json:"price_monthly_idr"`
	PriceYearlyIDR    int64            `json:"price_yearly_idr"`
	BillingPeriodDays int              `json:"billing_period_days"`
	IsPublic          bool             `json:"is_public"`
	SortOrder         int              `json:"sort_order"`
	FallbackGateway   string           `json:"fallback_gateway"`
	Features          map[string]bool  `json:"features"`
	Limits            map[string]int64 `json:"limits"`
}

// ListPublicPlans returns active, public plans ordered for display, each with
// its feature flags and numeric limits.
func (s *CatalogService) ListPublicPlans(ctx context.Context) ([]*PublicPlan, error) {
	plans, err := s.plans.ListActivePlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*PublicPlan, 0, len(plans))
	for _, p := range plans {
		if !p.IsPublic {
			continue
		}
		pp := &PublicPlan{
			Code:              p.Code,
			Name:              p.Name,
			Description:       p.Description,
			PriceMonthlyIDR:   p.PriceMonthlyIDR,
			PriceYearlyIDR:    p.PriceYearlyIDR,
			BillingPeriodDays: p.BillingPeriodDays,
			IsPublic:          p.IsPublic,
			SortOrder:         p.SortOrder,
			FallbackGateway:   p.FallbackGateway,
			Features:          map[string]bool{},
			Limits:            map[string]int64{},
		}
		if s.entSource != nil {
			if feats, ferr := s.entSource.GetEntitlements(ctx, p.ID); ferr == nil && feats != nil {
				pp.Features = feats
			}
			if lims, lerr := s.entSource.GetLimits(ctx, p.ID); lerr == nil && lims != nil {
				pp.Limits = lims
			}
		}
		out = append(out, pp)
	}
	return out, nil
}

// ─── User subscription self-service (upgrade/downgrade/cancel) ─

// SubscriptionService backs the user's plan-change actions. It reuses the
// billing lifecycle so all the money/soft-limit rules live in one place.
type SubscriptionService struct {
	subs       billing.SubscriptionRepository
	plans      billing.PlanRepository
	lifecycle  *billing.LifecycleService
	invalidate func(ctx context.Context, userID uuid.UUID)
}

// NewSubscriptionService builds the user subscription service.
func NewSubscriptionService(
	subs billing.SubscriptionRepository,
	plans billing.PlanRepository,
	lifecycle *billing.LifecycleService,
	invalidate func(ctx context.Context, userID uuid.UUID),
) *SubscriptionService {
	return &SubscriptionService{subs: subs, plans: plans, lifecycle: lifecycle, invalidate: invalidate}
}

// ScheduleDowngrade schedules a move to a lower plan at the end of the current
// cycle (soft-limit: access is not revoked now, no data is deleted).
func (s *SubscriptionService) ScheduleDowngrade(ctx context.Context, userID uuid.UUID, planCode string) error {
	sub, err := s.subs.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	plan, err := s.plans.GetPlanByCode(ctx, planCode)
	if err != nil {
		return err
	}
	if plan.ID == sub.PlanID {
		return fmt.Errorf("%w: already on this plan", billing.ErrInvalidStateTransition)
	}
	if sub.Metadata == nil {
		sub.Metadata = map[string]any{}
	}
	sub.Metadata["scheduled_plan_code"] = plan.Code
	sub.Metadata["scheduled_plan_id"] = plan.ID.String()
	sub.CancelAtPeriodEnd = true
	if err := s.subs.Update(ctx, sub); err != nil {
		return err
	}
	_ = s.subs.RecordEvent(ctx, &billing.SubscriptionEvent{
		SubscriptionID: sub.ID,
		UserID:         userID,
		EventType:      "downgrade_scheduled",
		PreviousState:  map[string]any{"plan_id": sub.PlanID.String()},
		NewState:       map[string]any{"scheduled_plan_code": plan.Code, "effective": "end_of_period"},
	})
	s.invalidateNow(ctx, userID)
	return nil
}

// CancelDowngrade removes a scheduled downgrade (the user keeps their plan).
func (s *SubscriptionService) CancelDowngrade(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.subs.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if sub.Metadata != nil {
		delete(sub.Metadata, "scheduled_plan_code")
		delete(sub.Metadata, "scheduled_plan_id")
	}
	sub.CancelAtPeriodEnd = false
	if err := s.subs.Update(ctx, sub); err != nil {
		return err
	}
	_ = s.subs.RecordEvent(ctx, &billing.SubscriptionEvent{
		SubscriptionID: sub.ID,
		UserID:         userID,
		EventType:      "downgrade_canceled",
		PreviousState:  map[string]any{"cancel_at_period_end": true},
		NewState:       map[string]any{"cancel_at_period_end": false},
	})
	s.invalidateNow(ctx, userID)
	return nil
}

// CancelSubscription cancels the subscription at period end (access continues
// until current_period_end; nothing is deleted).
func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.subs.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	sub.CancelAtPeriodEnd = true
	if err := s.subs.Update(ctx, sub); err != nil {
		return err
	}
	_ = s.subs.RecordEvent(ctx, &billing.SubscriptionEvent{
		SubscriptionID: sub.ID,
		UserID:         userID,
		EventType:      "cancel_scheduled",
		PreviousState:  map[string]any{"status": string(sub.Status)},
		NewState:       map[string]any{"cancel_at_period_end": true},
	})
	s.invalidateNow(ctx, userID)
	return nil
}

func (s *SubscriptionService) invalidateNow(ctx context.Context, userID uuid.UUID) {
	if s.invalidate != nil {
		s.invalidate(ctx, userID)
	}
}
