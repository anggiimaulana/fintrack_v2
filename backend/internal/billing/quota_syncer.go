package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// QuotaSyncer keeps the RUNNING token period's quota in step with the user's
// current plan after a mid-cycle plan change (upgrade or downgrade).
//
// Why this exists: the token period is created with the plan's quota SNAPSHOT
// and is immutable for its life, which is correct for a stable plan. But when
// the plan changes mid-cycle, the running period would keep enforcing the OLD
// plan's ceiling — a user who upgraded to Pro stayed capped at the Free quota
// until the cycle lapsed (up to a month). Replacing the period is not an option:
// it would overlap the running one and orphan its usage. Instead the running
// period's ceiling is updated in place, keeping usage intact.
//
// This is deliberately separate from the ledger: the ledger only reads a period,
// so a plan change must be pushed into the period by whoever applies the change
// (SubscriptionActivator), not discovered lazily on the next AI call.
type QuotaSyncer struct {
	tokens TokenRepository
	plans  PlanRepository
}

// NewQuotaSyncer builds the syncer.
func NewQuotaSyncer(tokens TokenRepository, plans PlanRepository) *QuotaSyncer {
	return &QuotaSyncer{tokens: tokens, plans: plans}
}

// SyncRunningPeriod updates the user's current token period to match planCode's
// quota and ratio. It is a no-op when the user has no period covering now (the
// next AI call creates one from the current plan), or when the quota and ratio
// already match.
//
// Best-effort: a failure here must not fail the payment that triggered it — the
// entitlement cache already reflects the new plan, and the period would be
// refreshed on the next cycle anyway. The error is returned so callers may log.
func (s *QuotaSyncer) SyncRunningPeriod(ctx context.Context, userID uuid.UUID, planCode string) error {
	if s == nil || s.tokens == nil || s.plans == nil || planCode == "" {
		return nil
	}
	plan, err := s.plans.GetPlanByCode(ctx, planCode)
	if err != nil {
		return err
	}
	limits, err := s.plans.GetLimits(ctx, plan.ID)
	if err != nil {
		return err
	}
	quota := limits["ai_tokens_per_cycle"]
	if quota < 0 {
		// Unlimited: the ledger substitutes a large sentinel; keep the period
		// consistent with what Reserve would compute for this plan.
		quota = defaultUnlimitedSentinel
	}
	if quota < 1 {
		quota = 1
	}
	ratio := plan.TokenQuotaRatio
	if ratio < 1 {
		ratio = 1
	}

	period, err := s.tokens.GetCurrentPeriod(ctx, userID, time.Now().UTC().Unix())
	if err != nil {
		return err
	}
	if period == nil {
		return nil // no running period: nothing to sync
	}
	if period.QuotaTokens == quota && period.QuotaRatio == ratio {
		return nil
	}
	logger.Info("quota syncer: updating running period after plan change",
		zap.String("user_id", userID.String()),
		zap.String("plan_code", planCode),
		zap.Int64("old_quota", period.QuotaTokens),
		zap.Int64("new_quota", quota),
		zap.Int64("old_ratio", period.QuotaRatio),
		zap.Int64("new_ratio", ratio))
	return s.tokens.UpdatePeriodQuota(ctx, period.ID, quota, ratio)
}

// defaultUnlimitedSentinel mirrors SettingsQuotaConfig's fallback sentinel, so a
// synced unlimited period matches what the ledger would otherwise use.
const defaultUnlimitedSentinel int64 = 1_000_000_000
