import type { PublicPlan } from "@/hooks/use-billing";

/**
 * Display metadata for a plan limit, mirroring the backend's entitlement
 * catalog (`LimitKey`). `enforced` is authoritative: a limit the backend does
 * not compute usage for must never be advertised as a plan benefit.
 */
export interface PlanLimitMeta {
  key: string;
  label: string;
  unit: string;
  enforced: boolean;
}

/**
 * Returns the plan limits the UI may advertise: only those the backend actually
 * enforces.
 *
 * A limit absent from `limit_meta` (older backend, no metadata) is treated as
 * enforced to stay backwards compatible. A limit explicitly marked
 * `enforced: false` is dropped — e.g. `max_categories` (categories are
 * admin-managed, not per-user) and `max_wallets` (no wallet entity exists yet).
 * Keys are sorted for a stable render.
 */
export function enforcedLimitEntries(plan: PublicPlan): Array<[string, number]> {
  return Object.entries(plan.limits)
    .filter(([key]) => plan.limit_meta?.[key]?.enforced !== false)
    .sort(([a], [b]) => a.localeCompare(b));
}

/**
 * Label for a limit key: the server-provided label when present, else a
 * built-in fallback, else the raw key.
 */
export function limitLabel(plan: PublicPlan, key: string, fallback: Record<string, string> = {}): string {
  return plan.limit_meta?.[key]?.label ?? fallback[key] ?? key;
}
