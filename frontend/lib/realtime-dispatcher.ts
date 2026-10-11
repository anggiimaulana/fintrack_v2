import { queryKeys } from "@/lib/cache-keys";

/**
 * Maps realtime domain events to the TanStack Query keys they invalidate.
 *
 * Surgical by design: every entry names specific keys so an event never triggers
 * a blanket `invalidateQueries()` (which would refetch the whole cache — the
 * "cache storm" P07 forbids). The backend gate is authoritative; this only keeps
 * the UI fresh.
 *
 * Pure function — no React, no client instance — so it is unit-testable.
 */
export type DomainEventName =
  | "payment_received"
  | "plan_updated"
  | "quota_reset"
  | "ai_endpoint_down"
  | "manual_approval_needed";

export interface DomainEvent {
  event: DomainEventName | string;
  timestamp?: number;
  id?: string;
  payload?: unknown;
}

/**
 * The minimal slice of TanStack Query's client the dispatcher needs. Declared as
 * a named interface so consumers (and tests) depend on a contract rather than on
 * the concrete client type.
 */
export interface QueryInvalidator {
  invalidateQueries: (filters: { queryKey: readonly unknown[] }) => void;
}

/** Query keys invalidated by each event. Empty array = deliberately no-op. */
export function keysForEvent(event: string): readonly (readonly unknown[])[] {
  switch (event) {
    case "payment_received":
      // The user's plan/quota may have changed, and dashboard totals shift.
      return [queryKeys.entitlements.me, queryKeys.dashboard.all];

    case "plan_updated":
      // Entitlements drive feature gating in the UI; refresh them immediately.
      return [queryKeys.entitlements.me, queryKeys.dashboard.all];

    case "quota_reset":
      // A new cycle: entitlements, transaction counters and reports all move.
      return [
        queryKeys.entitlements.me,
        queryKeys.transactions.all,
        queryKeys.dashboard.all,
        queryKeys.reports.all,
      ];

    case "ai_endpoint_down":
      // Admin dashboard: gateway health + metrics.
      return [queryKeys.superAdmin.gatewayEndpoints, queryKeys.superAdmin.metricsOverview()];

    case "manual_approval_needed":
      // Admin dashboard: the review queue gains an item.
      return [queryKeys.superAdmin.paymentsReview(), queryKeys.superAdmin.metricsOverview()];

    default:
      return [];
  }
}

/**
 * Applies the invalidation for one event. Returns the keys it invalidated, which
 * makes the behaviour assertable in tests.
 */
export function dispatchDomainEvent(queryClient: QueryInvalidator, event: DomainEvent): readonly (readonly unknown[])[] {
  const keys = keysForEvent(event.event);
  for (const key of keys) {
    queryClient.invalidateQueries({ queryKey: key });
  }
  return keys;
}
