package quota

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EvalParams provides context for quota evaluation.
type EvalParams struct {
	UserID              uuid.UUID
	PlanID              uuid.UUID
	LimitKey            string
	MonthlyLimit        int64
	CurrentMonthlyUsage int64
	WeeklyAnchor        *time.Time
	MonthlyPeriodStart  time.Time
	MonthlyPeriodEnd    time.Time
	Now                 time.Time
}

// DualCycleEnforcer evaluates whether a request is allowed, degraded, or blocked
// across rolling weekly and monthly cycles.
type DualCycleEnforcer struct {
	cycleStore CycleStore
	usageStore UsageCounterStore
}

// NewDualCycleEnforcer builds the enforcer.
func NewDualCycleEnforcer(cycleStore CycleStore, usageStore UsageCounterStore) *DualCycleEnforcer {
	return &DualCycleEnforcer{
		cycleStore: cycleStore,
		usageStore: usageStore,
	}
}

// CalculateWeeklyWindow computes the rolling 7-day window from anchor.
func CalculateWeeklyWindow(anchor, now time.Time) (time.Time, time.Time) {
	if now.Before(anchor) {
		return anchor, anchor.Add(7 * 24 * time.Hour)
	}
	elapsed := now.Sub(anchor)
	weekCount := elapsed / (7 * 24 * time.Hour)
	start := anchor.Add(time.Duration(weekCount) * 7 * 24 * time.Hour)
	end := start.Add(7 * 24 * time.Hour)
	return start, end
}

// Evaluate checks quota limits and computes effective remaining quota.
func (e *DualCycleEnforcer) Evaluate(ctx context.Context, p EvalParams) (*EvalResult, error) {
	now := p.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	threshold := 80
	if e.cycleStore != nil {
		if t, err := e.cycleStore.GetWarningThreshold(ctx, p.PlanID); err == nil && t > 0 {
			threshold = t
		}
	}

	// 1. Unlimited bypass (-1)
	if p.MonthlyLimit < 0 {
		return &EvalResult{
			Allowed:          true,
			MonthlyLimit:     -1,
			MonthlyRemaining: -1,
			Effective:        -1,
			Status:           StatusNormal,
			ResetAt:          p.MonthlyPeriodEnd,
		}, nil
	}

	// 2. Fetch cycle config for plan
	var cycle *PlanLimitCycle
	if e.cycleStore != nil {
		c, err := e.cycleStore.GetPlanCycle(ctx, p.PlanID, p.LimitKey)
		if err == nil {
			cycle = c
		}
	}

	// Get or create monthly counter
	var mCounter *UsageCounter
	if e.usageStore != nil && !p.MonthlyPeriodStart.IsZero() && !p.MonthlyPeriodEnd.IsZero() {
		c, err := e.usageStore.GetOrCreateCounter(ctx, p.UserID, p.LimitKey, CycleTypeMonthly, p.MonthlyPeriodStart, p.MonthlyPeriodEnd)
		if err != nil {
			return nil, err
		}
		mCounter = c
	}

	monthlyUsed := p.CurrentMonthlyUsage
	if mCounter != nil && mCounter.Used > monthlyUsed {
		monthlyUsed = mCounter.Used
	}
	monthlyRemaining := p.MonthlyLimit - monthlyUsed
	if monthlyRemaining < 0 {
		monthlyRemaining = 0
	}

	isWeekly := cycle != nil && cycle.CycleType == CycleWeeklyMonthly && cycle.WeeklyLimit != nil && *cycle.WeeklyLimit > 0

	// Case A: Monthly only
	if !isWeekly {
		resetAt := p.MonthlyPeriodEnd
		if monthlyUsed >= p.MonthlyLimit {
			return &EvalResult{
				Allowed:          false,
				MonthlyUsed:      monthlyUsed,
				MonthlyLimit:     p.MonthlyLimit,
				MonthlyRemaining: 0,
				Effective:        0,
				ResetAt:          resetAt,
				BlockReason:      BlockReasonMonthlyExhausted,
				Status:           StatusBlocked,
			}, nil
		}

		status := StatusNormal
		if p.MonthlyLimit > 0 && (monthlyRemaining*100)/p.MonthlyLimit <= int64(100-threshold) {
			status = StatusWarning
		}

		return &EvalResult{
			Allowed:          true,
			MonthlyUsed:      monthlyUsed,
			MonthlyLimit:     p.MonthlyLimit,
			MonthlyRemaining: monthlyRemaining,
			Effective:        monthlyRemaining,
			ResetAt:          resetAt,
			Status:           status,
		}, nil
	}

	// Case B: Weekly + Monthly
	weeklyLimit := *cycle.WeeklyLimit
	anchor := p.MonthlyPeriodStart
	if p.WeeklyAnchor != nil && !p.WeeklyAnchor.IsZero() {
		anchor = *p.WeeklyAnchor
	}
	wStart, wEnd := CalculateWeeklyWindow(anchor, now)

	var wCounter *UsageCounter
	if e.usageStore != nil {
		c, err := e.usageStore.GetOrCreateCounter(ctx, p.UserID, p.LimitKey, CycleTypeWeekly, wStart, wEnd)
		if err != nil {
			return nil, err
		}
		wCounter = c
	}

	weeklyUsed := int64(0)
	degradedUsed := int64(0)
	if wCounter != nil {
		weeklyUsed = wCounter.Used
		degradedUsed = wCounter.DegradedUsed
	}

	// In the first week of the subscription period, weekly usage is at least the monthly usage
	if (wStart.Equal(p.MonthlyPeriodStart) || wStart.Before(p.MonthlyPeriodStart)) && p.CurrentMonthlyUsage > weeklyUsed {
		weeklyUsed = p.CurrentMonthlyUsage
	}

	weeklyRemaining := weeklyLimit - weeklyUsed
	if weeklyRemaining < 0 {
		weeklyRemaining = 0
	}

	res := &EvalResult{
		WeeklyUsed:       weeklyUsed,
		WeeklyLimit:      weeklyLimit,
		WeeklyRemaining:  weeklyRemaining,
		MonthlyUsed:      monthlyUsed,
		MonthlyLimit:     p.MonthlyLimit,
		MonthlyRemaining: monthlyRemaining,
		ResetAt:          wEnd,
	}

	// Monthly hard cap check first
	if monthlyUsed >= p.MonthlyLimit {
		res.Allowed = false
		res.Effective = 0
		res.BlockReason = BlockReasonMonthlyExhausted
		res.Status = StatusBlocked
		res.ResetAt = p.MonthlyPeriodEnd
		return res, nil
	}

	// Weekly exhaustion check
	if weeklyUsed >= weeklyLimit {
		degMode := DegradedHardBlock
		if cycle.DegradedMode != nil {
			degMode = *cycle.DegradedMode
		}

		if degMode == DegradedHardBlock {
			res.Allowed = false
			res.Effective = 0
			res.BlockReason = BlockReasonWeeklyExhausted
			res.Status = StatusBlocked
			return res, nil
		}

		// Throttle or Model Downgrade
		if cycle.MaxDegradedUsage != nil && *cycle.MaxDegradedUsage > 0 && degradedUsed >= *cycle.MaxDegradedUsage {
			res.Allowed = false
			res.Effective = 0
			res.BlockReason = BlockReasonDegradedCap
			res.Status = StatusBlocked
			return res, nil
		}

		res.Allowed = true
		res.Degraded = true
		res.DegradedMode = degMode
		if cycle.FallbackModel != nil {
			res.FallbackModel = *cycle.FallbackModel
		}

		if degMode == DegradedThrottle {
			res.Status = StatusDegradedThrottle
		} else {
			res.Status = StatusDegradedDowngrade
		}
		return res, nil
	}

	// Both available: monthly caps weekly!
	effective := weeklyRemaining
	if monthlyRemaining < effective {
		effective = monthlyRemaining
	}

	res.Allowed = true
	res.Effective = effective

	limitBasis := weeklyLimit
	if p.MonthlyLimit > 0 && p.MonthlyLimit < limitBasis {
		limitBasis = p.MonthlyLimit
	}
	if limitBasis > 0 && (effective*100)/limitBasis <= int64(100-threshold) {
		res.Status = StatusWarning
	} else {
		res.Status = StatusNormal
	}

	return res, nil
}
// RecordWeeklyUsage increments weekly usage counter if the user's plan has a weekly cycle configured for limitKey.
func (e *DualCycleEnforcer) RecordWeeklyUsage(ctx context.Context, userID, planID uuid.UUID, limitKey string, delta int64, weeklyAnchor *time.Time, degraded bool) error {
	if delta <= 0 || e.cycleStore == nil || e.usageStore == nil {
		return nil
	}
	cycle, err := e.cycleStore.GetPlanCycle(ctx, planID, limitKey)
	if err != nil || cycle == nil || cycle.CycleType != CycleWeeklyMonthly || cycle.WeeklyLimit == nil {
		return nil
	}

	now := time.Now().UTC()
	anchor := now
	if weeklyAnchor != nil && !weeklyAnchor.IsZero() {
		anchor = *weeklyAnchor
	}
	wStart, wEnd := CalculateWeeklyWindow(anchor, now)

	counter, err := e.usageStore.GetOrCreateCounter(ctx, userID, limitKey, CycleTypeWeekly, wStart, wEnd)
	if err != nil {
		return err
	}

	if degraded {
		maxDeg := int64(0)
		if cycle.MaxDegradedUsage != nil {
			maxDeg = *cycle.MaxDegradedUsage
		}
		return e.usageStore.IncrementDegradedUsage(ctx, counter.ID, delta, maxDeg)
	}

	return e.usageStore.IncrementUsage(ctx, counter.ID, delta, *cycle.WeeklyLimit)
}
