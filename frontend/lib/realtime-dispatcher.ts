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
  | "user_status_updated"
  | "user_created"
  | "role_updated"
  | "tokens_adjusted"
  | "payment_status_changed"
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
  setQueriesData?: <T>(filters: { queryKey: readonly unknown[] }, updater: (old: T | undefined) => T | undefined) => void;
}

/** Query keys invalidated by each event. Empty array = deliberately no-op. */
export function keysForEvent(event: string, payload?: unknown): readonly (readonly unknown[])[] {
  switch (event) {
    case "payment_received":
      // The payment row flips to settled, plan/quota and usage counters update.
      return [
        queryKeys.entitlements.me,
        queryKeys.dashboard.all,
        queryKeys.billing.payments,
        queryKeys.billing.invoices,
        queryKeys.billing.subscription,
        queryKeys.usage.all,
      ];

    case "plan_updated":
      // Entitlements drive feature gating in the UI; refresh them immediately.
      // The subscription record is what the billing page renders.
      return [
        queryKeys.entitlements.me,
        queryKeys.dashboard.all,
        queryKeys.billing.subscription,
        queryKeys.billing.payments,
        queryKeys.billing.invoices,
        queryKeys.usage.all,
      ];

    case "quota_reset":
      // A new cycle: entitlements, transaction counters, reports and budgets all
      // move, and usage charts/history reset for the new window.
      return [
        queryKeys.entitlements.me,
        queryKeys.transactions.all,
        queryKeys.dashboard.all,
        queryKeys.reports.all,
        queryKeys.budgets.all,
        queryKeys.spk.all,
        queryKeys.usage.all,
        queryKeys.billing.subscription,
      ];

    case "usage_changed": {
      // Consumption moved (transaction created/deleted, AI tokens/requests, a
      // voice recording). Refresh entitlements and usage charts/ledger.
      const keys: (readonly unknown[])[] = [
        queryKeys.entitlements.me,
        queryKeys.usage.all,
      ];
      // Only invalidate voice config & history if voice quota specifically moved
      // (or if limit_keys is omitted for backward compatibility).
      const p = payload as { limit_keys?: string[] } | undefined;
      const touchesVoice = !p || !p.limit_keys || p.limit_keys.length === 0 || p.limit_keys.includes("voice_requests_per_day");
      if (touchesVoice) {
        keys.push(queryKeys.voice.config, queryKeys.voice.historyAll);
      }
      return keys;
    }
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
    case "user_status_updated":
      return [queryKeys.admin.users, queryKeys.admin.stats];

    case "user_created":
      return [queryKeys.admin.users, queryKeys.admin.stats, queryKeys.admin.recentUsers];

    case "role_updated":
      return [queryKeys.admin.users, queryKeys.entitlements.me];

    case "tokens_adjusted":
      return [queryKeys.superAdmin.tokensUsers(), queryKeys.superAdmin.tokensLedger(), queryKeys.superAdmin.metricsOverview()];
    case "payment_status_changed":
      return [
        queryKeys.superAdmin.payments(),
        queryKeys.superAdmin.paymentsReview(),
        queryKeys.superAdmin.metricsOverview(),
        queryKeys.superAdmin.metricsTimeseries(),
      ];

    default:
      return [];
  }
}

/**
 * Applies the invalidation for one event. Returns the keys it invalidated, which
 * makes the behaviour assertable in tests.
 */
interface UserCacheRow {
  id: string;
  is_active: boolean;
  [key: string]: unknown;
}

interface UserPaginatedCache {
  users?: UserCacheRow[];
  data?: UserCacheRow[];
}

export function dispatchDomainEvent(queryClient: QueryInvalidator, event: DomainEvent): readonly (readonly unknown[])[] {
  if (event.event === "user_status_updated" && queryClient.setQueriesData && event.payload && typeof event.payload === "object" && "id" in event.payload && "is_active" in event.payload) {
    const targetId = event.payload.id;
    const nextActive = Boolean(event.payload.is_active);

    if (typeof targetId === "string") {
      queryClient.setQueriesData({ queryKey: queryKeys.admin.users }, (old: unknown) => {
        if (!old || typeof old !== "object") return old;
        if (Array.isArray(old)) {
          return old.map((u: unknown) => {
            if (u && typeof u === "object" && "id" in u && u.id === targetId) {
              return { ...u, is_active: nextActive };
            }
            return u;
          });
        }
        const obj = old as UserPaginatedCache;
        if (Array.isArray(obj.users)) {
          return {
            ...obj,
            users: obj.users.map((u) => (u.id === targetId ? { ...u, is_active: nextActive } : u)),
          };
        }
        if (Array.isArray(obj.data)) {
          return {
            ...obj,
            data: obj.data.map((u) => (u.id === targetId ? { ...u, is_active: nextActive } : u)),
          };
        }
        return old;
      });
    }
  }
  const keys = keysForEvent(event.event, event.payload);
  for (const key of keys) {
    queryClient.invalidateQueries({ queryKey: key });
  }
  return keys;
}
