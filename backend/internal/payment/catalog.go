package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
)

// LimitMeta is display metadata for a plan limit key: its human label, unit,
// and — crucially — whether the backend actually enforces it. A limit that is
// not enforced must never be advertised to users as a plan benefit (F4).
type LimitMeta struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Unit     string `json:"unit"`
	Enforced bool   `json:"enforced"`
}

// CatalogService exposes the public plan catalog (dynamic, from the DB) so the
// pricing UI no longer hardcodes names, prices, features, or limits.
type CatalogService struct {
	plans     billing.PlanRepository
	entSource PlanEntitlementSource
	// limitMeta maps a limit key to its display metadata. Sourced from the
	// authoritative entitlement catalog so the UI can label limits and hide the
	// ones the backend does not enforce.
	limitMeta map[string]LimitMeta
	// tax computes the same tax the checkout will charge, so the UI can show the
	// real total *before* an invoice is issued (no price surprise on checkout).
	tax *TaxCalculator
}

// PlanEntitlementSource reads a plan's features and limits.
type PlanEntitlementSource interface {
	GetEntitlements(ctx context.Context, planID uuid.UUID) (map[string]bool, error)
	GetLimits(ctx context.Context, planID uuid.UUID) (map[string]int64, error)
}

// NewCatalogService builds the catalog service. limitMeta may be nil (the
// response then omits limit_meta and the UI falls back to raw keys). tax may be
// nil (the response then reports a 0% rate and total == subtotal).
func NewCatalogService(plans billing.PlanRepository, entSource PlanEntitlementSource, limitMeta []LimitMeta, tax *TaxCalculator) *CatalogService {
	m := make(map[string]LimitMeta, len(limitMeta))
	for _, lm := range limitMeta {
		m[lm.Key] = lm
	}
	return &CatalogService{plans: plans, entSource: entSource, limitMeta: m, tax: tax}
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
	// LimitMeta annotates each limit key with label/unit/enforced so the UI can
	// present only the limits the backend actually enforces.
	LimitMeta map[string]LimitMeta `json:"limit_meta,omitempty"`
	// TaxEnabled is true when the operator has switched PPN on. The UI hides the
	// tax line entirely when false, rather than showing "+ 0%".
	TaxEnabled bool `json:"tax_enabled"`
	// TaxLabel is the display name for the tax (e.g. "PPN").
	TaxLabel string `json:"tax_label,omitempty"`
	// TaxRatePercent is the PPN rate applied at checkout. Exposed so the pricing
	// UI can show the same total the invoice will carry.
	TaxRatePercent int64 `json:"tax_rate_percent"`
	// TaxIDR / TotalIDR are the tax and grand total for one billing period at
	// the current rate (subtotal = PriceMonthlyIDR).
	TaxIDR   int64 `json:"tax_idr"`
	TotalIDR int64 `json:"total_idr"`
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
			LimitMeta:         map[string]LimitMeta{},
		}
		if s.entSource != nil {
			if feats, ferr := s.entSource.GetEntitlements(ctx, p.ID); ferr == nil && feats != nil {
				pp.Features = feats
			}
			if lims, lerr := s.entSource.GetLimits(ctx, p.ID); lerr == nil && lims != nil {
				pp.Limits = lims
			}
		}
		// Attach metadata only for limits this plan actually defines.
		for key := range pp.Limits {
			if lm, ok := s.limitMeta[key]; ok {
				pp.LimitMeta[key] = lm
			}
		}
		// Show the tax-inclusive total the checkout will charge, so the price on
		// the card matches the invoice exactly. When tax is disabled the rate is
		// 0 and the total equals the subtotal, so the UI hides the tax line.
		if s.tax != nil {
			pp.TaxEnabled = s.tax.Enabled(ctx)
			pp.TaxLabel = s.tax.Label(ctx)
			pp.TaxRatePercent = s.tax.RatePercent(ctx)
			tax, total := s.tax.Total(ctx, pp.PriceMonthlyIDR)
			pp.TaxIDR = tax
			pp.TotalIDR = total
		} else {
			pp.TotalIDR = pp.PriceMonthlyIDR
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
