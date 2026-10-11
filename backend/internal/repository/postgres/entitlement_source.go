package postgres

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/pkg/logger"
)

// EntitlementSource implements entitlement.Source over PostgreSQL by joining the
// user's active subscription with their plan.
type EntitlementSource struct {
	repo *BillingRepo
	// defaultPlanCode is the plan assigned to users with no subscription row.
	// Empty means "no fallback" (deny-all), which is the pre-P07 behaviour.
	defaultPlanCode string
}

// NewEntitlementSource builds the source with the `free` fallback plan.
func NewEntitlementSource(repo *BillingRepo) *EntitlementSource {
	return &EntitlementSource{repo: repo, defaultPlanCode: DefaultPlanCode()}
}

// DefaultPlanCode reads DEFAULT_PLAN_CODE (default "free"). An empty value
// disables the fallback, restoring deny-all for users without a subscription.
func DefaultPlanCode() string {
	if v := strings.TrimSpace(os.Getenv("DEFAULT_PLAN_CODE")); v != "" {
		return v
	}
	return "free"
}

// GetActivePlanForUser returns the plan + status for a user. A user with no
// subscription row falls back to the default (Free) plan with StatusNone, so
// enabling entitlement enforcement does not lock out accounts created before
// subscriptions existed (see the P07 entitlement spec §3.1).
//
// Fail-closed: if the fallback plan itself is missing or unreadable, this
// returns (nil, "", nil) so the resolver yields deny-all. Access is never
// widened on error.
func (s *EntitlementSource) GetActivePlanForUser(ctx context.Context, userID uuid.UUID) (*billing.Plan, billing.SubscriptionStatus, error) {
	var planID uuid.UUID
	var status billing.SubscriptionStatus
	err := q(ctx, s.repo.pool).QueryRow(ctx,
		`SELECT plan_id, status FROM subscriptions WHERE user_id = $1`, userID).
		Scan(&planID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.defaultPlan(ctx, userID)
	}
	if err != nil {
		return nil, "", err
	}
	if !status.GrantsPaidAccess() {
		// Canceled/expired subscription: the plan no longer applies. Fall back
		// to the default plan instead of silently honouring a lapsed plan, which
		// would grant paid features indefinitely.
		logger.Info("subscription does not grant paid access; using default plan",
			zap.String("user_id", userID.String()), zap.String("status", string(status)))
		return s.defaultPlan(ctx, userID)
	}
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, "", err
	}
	return plan, status, nil
}

// defaultPlan resolves the fallback plan for a user with no subscription.
func (s *EntitlementSource) defaultPlan(ctx context.Context, userID uuid.UUID) (*billing.Plan, billing.SubscriptionStatus, error) {
	if s.defaultPlanCode == "" {
		return nil, "", nil // fallback disabled -> deny-all
	}
	plan, err := s.repo.GetPlanByCode(ctx, s.defaultPlanCode)
	if err != nil {
		if errors.Is(err, billing.ErrPlanNotFound) {
			// Missing default plan must not grant access; deny-all and surface it.
			logger.Warn("default plan not found; denying entitlements",
				zap.String("plan_code", s.defaultPlanCode), zap.String("user_id", userID.String()))
			return nil, "", nil
		}
		return nil, "", err
	}
	if plan == nil {
		return nil, "", nil
	}
	return plan, billing.StatusNone, nil
}

func (s *EntitlementSource) GetEntitlements(ctx context.Context, planID uuid.UUID) (map[string]bool, error) {
	return s.repo.GetEntitlements(ctx, planID)
}

func (s *EntitlementSource) GetLimits(ctx context.Context, planID uuid.UUID) (map[string]int64, error) {
	return s.repo.GetLimits(ctx, planID)
}

// compile-time assertion.
var _ entitlement.Source = (*EntitlementSource)(nil)
