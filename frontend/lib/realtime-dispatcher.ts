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
  | "usage_changed"
  | "ai_endpoint_down"
  | "ai_route_quarantined"
  | "manual_approval_needed"
  | "config_changed"
  // Client-synthesised: emitted by the stream hook when the SSE connection is
  // re-established after a drop, so consumers can resync anything missed. It is
  // never sent by the server.
  | "stream_reconnected";

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
      // The payment row flips to settled and the plan/quota may change, so both
      // the billing history and the entitlements must refresh.
      return [
        queryKeys.entitlements.me,
        queryKeys.dashboard.all,
        queryKeys.billing.payments,
        queryKeys.billing.invoices,
        queryKeys.billing.subscription,
      ];

    case "plan_updated":
      // Entitlements drive feature gating in the UI; refresh them immediately.
      // The subscription record is what the billing page renders.
      return [
        queryKeys.entitlements.me,
        queryKeys.dashboard.all,
        queryKeys.billing.subscription,
        queryKeys.billing.payments,
      ];

    case "quota_reset":
      // A new cycle: entitlements, transaction counters, reports and budgets all
      // move.
      return [
        queryKeys.entitlements.me,
        queryKeys.transactions.all,
        queryKeys.dashboard.all,
        queryKeys.reports.all,
        queryKeys.budgets.all,
        queryKeys.spk.all,
      ];

    case "usage_changed":
      // Consumption moved (transaction created/deleted, AI tokens/requests, a
      // voice recording). Refresh the entitlements query — the authoritative
      // usage source — the voice config (which carries the daily counter), and
      // the voice history (a server-mode recording is written backend-side and
      // must appear in the panel without a manual refresh). The history query is
      // only active while its panel is open, so a closed panel issues no request.
      return [queryKeys.entitlements.me, queryKeys.voice.config, queryKeys.voice.historyAll];

    case "config_changed":
      // The configuration catalog changed (setting/metric/feature/plan/pricing).
      // Reload ONLY the catalog queries plus entitlements and the settings
      // VALUES (an operator on the Settings page must see a peer's change) —
      // never a blanket refetch. The voice config is included because every
      // voice knob is a dynamic setting.
      return [
        queryKeys.superAdmin.entitlementCatalog,
        queryKeys.superAdmin.settingDefinitions,
        queryKeys.superAdmin.metricDefinitions,
        queryKeys.superAdmin.featureDefinitions,
        queryKeys.superAdmin.settings,
        queryKeys.entitlements.me,
        queryKeys.voice.config,
      ];

    case "stream_reconnected":
      // The SSE stream reconnected after a gap. Buffered events are replayed on
      // resume, but a gap longer than the server's buffer TTL (or a Redis-less
      // deployment) can drop events, so resync the notification inbox exactly
      // once. This is what replaces polling.
      return [queryKeys.notifications.all];

    case "ai_endpoint_down":
      // Admin dashboard: gateway health + metrics.
      return [queryKeys.superAdmin.gatewayEndpoints, queryKeys.superAdmin.metricsOverview()];

    case "ai_route_quarantined":
      // A single (endpoint, model) route was quarantined or recovered: refresh
      // the routing matrix (route_state) and the endpoint list (breaker state).
      return [queryKeys.superAdmin.gatewayRoutes, queryKeys.superAdmin.gatewayEndpoints];

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
