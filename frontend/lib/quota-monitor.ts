/**
 * Pure helpers for the Super Admin quota monitor.
 *
 * Extracted so the severity banding and summary counts are unit-testable
 * without rendering the table.
 */

/** A row of quota consumption, as returned by GET /admin/quota-monitor. */
export interface QuotaRowLike {
  used_percent: number;
}

/** Severity band for a usage percentage: >=90 critical, otherwise warning. */
export function quotaSeverity(percent: number): "critical" | "warning" {
  return percent >= 90 ? "critical" : "warning";
}

export interface QuotaSummary {
  total: number;
  critical: number;
  warning: number;
}

/** Counts how many monitored rows are critical (>=90%) vs merely near the limit. */
export function summariseQuota(rows: QuotaRowLike[]): QuotaSummary {
  let critical = 0;
  for (const r of rows) {
    if (quotaSeverity(r.used_percent) === "critical") critical++;
  }
  return { total: rows.length, critical, warning: rows.length - critical };
}
