import type { SpkMetricIndicator, SpkMetricModel, SpkStatusThreshold } from "@/hooks/useSpkMetrics";

// ─── SPK scoring, mirrored on the client ──────────────────────
//
// The backend is authoritative for the stored score. This module exists so the
// UI can show a *what-if* projection instantly (without a round trip): given the
// user's current breakdown and a hypothetical change, recompute the weighted
// score using the same rules the Go engine uses.
//
// Keeping the formula in one tested place means the simulator and the real
// engine agree; the parity test in spk-score.test.ts pins the key cases.

/**
 * scoreIndicator converts a raw value into a 0..100 sub-score against the
 * indicator's target and direction. Mirrors `scoreIndicator` in spk_usecase.go.
 */
export function scoreIndicator(ind: SpkMetricIndicator, raw: number): number {
  if (ind.target <= 0) return 0;
  if ((ind.direction ?? "higher_better") === "lower_better") {
    if (raw <= 0 || raw <= ind.target) return 100;
    return Math.min(100, (ind.target / raw) * 100);
  }
  if (raw <= 0) return 0;
  return Math.min(100, (raw / ind.target) * 100);
}

/**
 * weightedScore combines sub-scores into a 0..100 total using the model weights.
 * `subScores` maps indicator key → its 0..100 sub-score.
 */
export function weightedScore(model: SpkMetricModel, subScores: Record<string, number>): number {
  let total = 0;
  for (const ind of model.indicators) {
    const s = subScores[ind.key] ?? 0;
    total += s * ind.weight;
  }
  return Math.max(0, Math.min(100, total));
}

/**
 * statusFor maps a score to a label using the model thresholds, matching
 * `determineStatus` in the Go engine (highest Min the score reaches; the lowest
 * threshold when it reaches none).
 */
export function statusFor(thresholds: SpkStatusThreshold[], score: number): string {
  if (thresholds.length === 0) return "—";
  let best = thresholds[0];
  let matched = false;
  let lowest = thresholds[0];
  for (const th of thresholds) {
    if (th.min < lowest.min) lowest = th;
    if (score >= th.min && (!matched || th.min > best.min)) {
      best = th;
      matched = true;
    }
  }
  return (matched ? best : lowest).label;
}

/**
 * invertSubScore recovers the raw indicator value that produced a given
 * sub-score, so the simulator can scale it by a percentage. Returns null when
 * the relationship cannot be inverted (e.g. a lower_better score at full marks).
 */
export function invertSubScore(ind: SpkMetricIndicator, subScore: number): number | null {
  if (ind.target <= 0 || subScore <= 0) return null;
  if ((ind.direction ?? "higher_better") === "lower_better") {
    if (subScore >= 100) return null;
    return (ind.target * 100) / subScore;
  }
  return (subScore * ind.target) / 100;
}

/**
 * projectScore applies a set of hypothetical changes to the current breakdown
 * and returns the projected total score plus the per-indicator before/after.
 *
 * `changes` maps an indicator key to a percentage adjustment of its *raw* value
 * (e.g. `{ "needs_vs_wants": -10 }` = "cut this category's spend by 10%").
 */
export interface ScoreProjection {
  current: number;
  projected: number;
  delta: number;
  currentStatus: string;
  projectedStatus: string;
  perIndicator: { key: string; label: string; before: number; after: number }[];
}

export function projectScore(
  model: SpkMetricModel,
  currentSubScores: Record<string, number>,
  changes: Record<string, number>
): ScoreProjection {
  const after: Record<string, number> = { ...currentSubScores };
  const perIndicator: ScoreProjection["perIndicator"] = [];

  for (const ind of model.indicators) {
    const before = currentSubScores[ind.key] ?? 0;
    const pct = changes[ind.key];
    let next = before;
    if (pct) {
      const raw = invertSubScore(ind, before);
      if (raw !== null) {
        const scaledRaw = raw * (1 + pct / 100);
        next = scoreIndicator(ind, scaledRaw);
      }
    }
    after[ind.key] = next;
    perIndicator.push({ key: ind.key, label: ind.label, before, after: next });
  }

  const current = weightedScore(model, currentSubScores);
  const projected = weightedScore(model, after);
  return {
    current: Math.round(current * 100) / 100,
    projected: Math.round(projected * 100) / 100,
    delta: Math.round((projected - current) * 100) / 100,
    currentStatus: statusFor(model.status_thresholds, current),
    projectedStatus: statusFor(model.status_thresholds, projected),
    perIndicator,
  };
}
