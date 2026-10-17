import type { PlanLimitCycle } from "@/hooks/usePlanCycles";

/**
 * Shared shape of the public plan catalog (`GET /api/v1/plans` and the public
 * `GET /api/v1/public/plans`).
 *
 * Single source of truth: `hooks/use-billing.ts` re-exports these so existing
 * imports keep working, and `lib/plans-server.ts` consumes them on the server.
 * All amounts are integer Rupiah (never floats).
 */

/** Display metadata for a plan limit, including whether it is enforced. */
export interface PlanLimitMeta {
  key: string;
  label: string;
  unit: string;
  enforced: boolean;
}

export interface PublicPlan {
  code: string;
  name: string;
  description?: string;
  price_monthly_idr: number;
  price_yearly_idr: number;
  billing_period_days: number;
  is_public: boolean;
  sort_order: number;
  fallback_gateway: string;
  features: Record<string, boolean>;
  limits: Record<string, number>;
  /** Per-limit label/unit/enforced, from the backend entitlement catalog. */
  limit_meta?: Record<string, PlanLimitMeta>;
  /** Cycle configurations (e.g. weekly/monthly, degraded mode) per limit key. */
  cycles?: Record<string, PlanLimitCycle>;
  /** True when tax is applied at checkout (drives the separate tax line). */
  tax_enabled?: boolean;
  /** Display name for the tax (e.g. "PPN"). */
  tax_label?: string;
  /** PPN rate applied at checkout (0 when tax is off). */
  tax_rate_percent?: number;
  /** Tax amount for one billing period at the current rate. */
  tax_idr?: number;
  /** Grand total (subtotal + tax) for one billing period. */
  total_idr?: number;
}
