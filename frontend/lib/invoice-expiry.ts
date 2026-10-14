/**
 * Invoice expiry helpers.
 *
 * Kept pure and dependency-free so the countdown is unit-testable and the same
 * rule ("is this invoice still payable?") is applied consistently in the UI.
 */

/** remainingMs returns ms until `dueAt`, clamped at 0 (never negative). */
export function remainingMs(dueAt: string | Date | null | undefined, now: number = Date.now()): number {
  if (!dueAt) return 0;
  const due = typeof dueAt === "string" ? new Date(dueAt).getTime() : dueAt.getTime();
  if (Number.isNaN(due)) return 0;
  return Math.max(0, due - now);
}

/** isExpired reports whether the due date has passed. */
export function isExpired(dueAt: string | Date | null | undefined, now: number = Date.now()): boolean {
  if (!dueAt) return false;
  const due = typeof dueAt === "string" ? new Date(dueAt).getTime() : dueAt.getTime();
  if (Number.isNaN(due)) return false;
  return due <= now;
}

/**
 * formatCountdown renders the remaining time compactly: "2h 15m", "45m 10s",
 * "12s". Returns "" when nothing remains (so callers can hide the element).
 */
export function formatCountdown(ms: number): string {
  if (ms <= 0) return "";
  const totalSecs = Math.floor(ms / 1000);
  const hours = Math.floor(totalSecs / 3600);
  const mins = Math.floor((totalSecs % 3600) / 60);
  const secs = totalSecs % 60;
  if (hours > 0) return `${hours}j ${mins}m`;
  if (mins > 0) return `${mins}m ${secs}s`;
  return `${secs}s`;
}

/**
 * urgencyFor classifies a countdown for colouring: "expired" (gone),
 * "critical" (< 1h), "warning" (< 6h), or "normal".
 */
export function urgencyFor(ms: number): "expired" | "critical" | "warning" | "normal" {
  if (ms <= 0) return "expired";
  if (ms < 60 * 60 * 1000) return "critical";
  if (ms < 6 * 60 * 60 * 1000) return "warning";
  return "normal";
}

/** The invoice statuses a user may still cancel (matches the backend rule). */
export function isCancellable(status: string): boolean {
  return status === "issued" || status === "past_due";
}
