package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/entitlement"
)

// EntitlementSource implements entitlement.Source over PostgreSQL by joining the
// user's active subscription with their plan.
type EntitlementSource struct {
	repo *BillingRepo
}

// NewEntitlementSource builds the source.
func NewEntitlementSource(repo *BillingRepo) *EntitlementSource {
	return &EntitlementSource{repo: repo}
}

// GetActivePlanForUser returns the plan + status for a user, or (nil, "", nil)
// when the user has no subscription.
func (s *EntitlementSource) GetActivePlanForUser(ctx context.Context, userID uuid.UUID) (*billing.Plan, billing.SubscriptionStatus, error) {
	var planID uuid.UUID
	var status billing.SubscriptionStatus
	err := q(ctx, s.repo.pool).QueryRow(ctx,
		`SELECT plan_id, status FROM subscriptions WHERE user_id = $1`, userID).
		Scan(&planID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, "", err
	}
	return plan, status, nil
}

func (s *EntitlementSource) GetEntitlements(ctx context.Context, planID uuid.UUID) (map[string]bool, error) {
	return s.repo.GetEntitlements(ctx, planID)
}

func (s *EntitlementSource) GetLimits(ctx context.Context, planID uuid.UUID) (map[string]int64, error) {
	return s.repo.GetLimits(ctx, planID)
}

// compile-time assertion.
var _ entitlement.Source = (*EntitlementSource)(nil)
