import { describe, it, expect, vi } from "vitest";
import { keysForEvent, dispatchDomainEvent, type DomainEvent, type QueryInvalidator } from "./realtime-dispatcher";
import { queryKeys } from "./cache-keys";

describe("keysForEvent", () => {
  it("invalidates entitlements for payment_received", () => {
    const keys = keysForEvent("payment_received");
    expect(keys).toContainEqual(queryKeys.entitlements.me);
  });

  it("invalidates entitlements for plan_updated", () => {
    const keys = keysForEvent("plan_updated");
    expect(keys).toContainEqual(queryKeys.entitlements.me);
    // The billing page renders the subscription record, so it must refresh too.
    expect(keys).toContainEqual(queryKeys.billing.subscription);
  });

  it("refreshes billing history when a payment settles", () => {
    const keys = keysForEvent("payment_received");
    expect(keys).toContainEqual(queryKeys.billing.payments);
    expect(keys).toContainEqual(queryKeys.billing.invoices);
  });

  it("invalidates transaction and report caches on quota_reset", () => {
    const keys = keysForEvent("quota_reset");
    expect(keys).toContainEqual(queryKeys.transactions.all);
    expect(keys).toContainEqual(queryKeys.reports.all);
    // A new cycle also shifts budgets and the SPK score.
    expect(keys).toContainEqual(queryKeys.budgets.all);
    expect(keys).toContainEqual(queryKeys.spk.all);
  });

  it("targets super-admin gateway keys for ai_endpoint_down", () => {
    const keys = keysForEvent("ai_endpoint_down");
    expect(keys).toContainEqual(queryKeys.superAdmin.gatewayEndpoints);
  });

  it("refreshes the routing matrix for ai_route_quarantined", () => {
    const keys = keysForEvent("ai_route_quarantined");
    expect(keys).toContainEqual(queryKeys.superAdmin.gatewayRoutes);
    expect(keys).toContainEqual(queryKeys.superAdmin.gatewayEndpoints);
  });

  it("targets the payments review queue for manual_approval_needed", () => {
    const keys = keysForEvent("manual_approval_needed");
    expect(keys).toContainEqual(queryKeys.superAdmin.paymentsReview());
  });

  it("refreshes entitlements, usage cache, voice config and voice history on usage_changed", () => {
    const keys = keysForEvent("usage_changed");
    expect(keys).toContainEqual(queryKeys.entitlements.me);
    expect(keys).toContainEqual(queryKeys.quota.snapshot);
    expect(keys).toContainEqual(queryKeys.usage.all);
    // The voice config carries the live daily counter, so it must refresh too.
    expect(keys).toContainEqual(queryKeys.voice.config);
    // A server-mode recording is written backend-side; the history panel must
    // pick it up without a manual refresh.
    expect(keys).toContainEqual(queryKeys.voice.historyAll);
    // Must not refetch data lists — only quota/usage surfaces moved.
    expect(keys).toHaveLength(5);
    expect(keys).not.toContainEqual(queryKeys.transactions.all);
    expect(keys).not.toContainEqual(queryKeys.dashboard.all);
  });

  it("does not invalidate voice config when usage_changed is only for ai_chat or transactions", () => {
    const chatKeys = keysForEvent("usage_changed", { limit_keys: ["ai_tokens_per_cycle", "ai_requests_per_cycle"] });
    expect(chatKeys).toContainEqual(queryKeys.entitlements.me);
    expect(chatKeys).toContainEqual(queryKeys.quota.snapshot);
    expect(chatKeys).toContainEqual(queryKeys.usage.all);
    expect(chatKeys).not.toContainEqual(queryKeys.voice.config);
    expect(chatKeys).not.toContainEqual(queryKeys.voice.historyAll);
    expect(chatKeys).toHaveLength(3);

    const txKeys = keysForEvent("usage_changed", { limit_keys: ["max_transactions"] });
    expect(txKeys).not.toContainEqual(queryKeys.voice.config);
    expect(txKeys).toHaveLength(3);
  });

  it("invalidates entitlements, quota snapshot, and usage on quota_degraded and quota_blocked", () => {
    const degKeys = keysForEvent("quota_degraded");
    expect(degKeys).toContainEqual(queryKeys.entitlements.me);
    expect(degKeys).toContainEqual(queryKeys.quota.snapshot);
    expect(degKeys).toContainEqual(queryKeys.usage.all);

    const blkKeys = keysForEvent("quota_blocked");
    expect(blkKeys).toContainEqual(queryKeys.entitlements.me);
    expect(blkKeys).toContainEqual(queryKeys.quota.snapshot);
    expect(blkKeys).toContainEqual(queryKeys.usage.all);
  });

  it("invalidates plans on cycle_config_changed", () => {
    const keys = keysForEvent("cycle_config_changed");
    expect(keys).toContainEqual(queryKeys.superAdmin.plans);
    expect(keys).toContainEqual(queryKeys.quota.snapshot);
  });
  it("resyncs notifications on stream_reconnected", () => {
    const keys = keysForEvent("stream_reconnected");
    expect(keys).toContainEqual(queryKeys.notifications.all);
    // A reconnect is a resync hint only — it must not storm every cache.
    expect(keys).toHaveLength(1);
  });
  it("invalidates admin user and stats for user_status_updated", () => {
    const keys = keysForEvent("user_status_updated");
    expect(keys).toContainEqual(queryKeys.admin.users);
    expect(keys).toContainEqual(queryKeys.admin.stats);
  });

  it("invalidates super-admin token balances for tokens_adjusted", () => {
    const keys = keysForEvent("tokens_adjusted");
    expect(keys).toContainEqual(queryKeys.superAdmin.tokensUsers());
    expect(keys).toContainEqual(queryKeys.superAdmin.tokensLedger());
  });

  it("invalidates payment lists for payment_status_changed", () => {
    const keys = keysForEvent("payment_status_changed");
    expect(keys).toContainEqual(queryKeys.superAdmin.payments());
    expect(keys).toContainEqual(queryKeys.superAdmin.paymentsReview());
  });

  it("returns nothing for an unknown event", () => {
    expect(keysForEvent("some_future_event")).toEqual([]);
  });

  it("never returns an empty/undefined key (no blanket invalidation)", () => {
    for (const name of [
      "payment_received",
      "plan_updated",
      "quota_reset",
      "usage_changed",
      "ai_endpoint_down",
      "ai_route_quarantined",
      "manual_approval_needed",
    ]) {
      for (const key of keysForEvent(name)) {
        expect(Array.isArray(key)).toBe(true);
        expect(key.length).toBeGreaterThan(0);
      }
    }
  });
});

describe("dispatchDomainEvent", () => {
  it("invalidates exactly the mapped keys, with no unfiltered call", () => {
    const invalidate = vi.fn();
    const fakeClient: QueryInvalidator = { invalidateQueries: invalidate };

    const event: DomainEvent = { event: "plan_updated", id: "evt_1_1" };
    const applied = dispatchDomainEvent(fakeClient, event);

    // One invalidateQueries call per mapped key, and the return value agrees.
    expect(applied.length).toBeGreaterThan(0);
    expect(invalidate).toHaveBeenCalledTimes(applied.length);
    // Every call must carry a specific queryKey — never a bare call.
    for (const call of invalidate.mock.calls) {
      expect(call[0]).toHaveProperty("queryKey");
      expect(call[0].queryKey).toBeDefined();
    }
  });

  it("does not call the client at all for an unknown event", () => {
    const invalidate = vi.fn();
    const fakeClient: QueryInvalidator = { invalidateQueries: invalidate };

    const applied = dispatchDomainEvent(fakeClient, { event: "unmapped" });
    expect(applied).toEqual([]);
    expect(invalidate).not.toHaveBeenCalled();
  });

  it("patches user status in-memory for user_status_updated event", () => {
    const invalidate = vi.fn();
    let cachedUsers = [
      { id: "u-1", full_name: "John", is_active: true },
      { id: "u-2", full_name: "Jane", is_active: false },
    ];

    const setQueriesData = vi.fn((_filters: unknown, updater: (old: unknown) => unknown) => {
      cachedUsers = updater(cachedUsers) as typeof cachedUsers;
    });

    const fakeClient: QueryInvalidator = {
      invalidateQueries: invalidate,
      setQueriesData,
    };

    const event: DomainEvent = {
      event: "user_status_updated",
      payload: { id: "u-1", is_active: false },
    };

    dispatchDomainEvent(fakeClient, event);

    expect(setQueriesData).toHaveBeenCalled();
    expect(cachedUsers[0].is_active).toBe(false);
    expect(cachedUsers[1].is_active).toBe(false);
  });

  it("patches user status in-memory to false for user_suspended event", () => {
    const invalidate = vi.fn();
    let cachedUsers = [
      { id: "u-1", full_name: "John", is_active: true },
    ];

    const setQueriesData = vi.fn((_filters: unknown, updater: (old: unknown) => unknown) => {
      cachedUsers = updater(cachedUsers) as typeof cachedUsers;
    });

    const fakeClient: QueryInvalidator = {
      invalidateQueries: invalidate,
      setQueriesData,
    };

    const event: DomainEvent = {
      event: "user_suspended",
      payload: { user_id: "u-1" },
    };

    dispatchDomainEvent(fakeClient, event);

    expect(setQueriesData).toHaveBeenCalled();
    expect(cachedUsers[0].is_active).toBe(false);
    expect(invalidate).toHaveBeenCalled();
  });
});
