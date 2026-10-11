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
  });

  it("invalidates transaction and report caches on quota_reset", () => {
    const keys = keysForEvent("quota_reset");
    expect(keys).toContainEqual(queryKeys.transactions.all);
    expect(keys).toContainEqual(queryKeys.reports.all);
  });

  it("targets super-admin gateway keys for ai_endpoint_down", () => {
    const keys = keysForEvent("ai_endpoint_down");
    expect(keys).toContainEqual(queryKeys.superAdmin.gatewayEndpoints);
  });

  it("targets the payments review queue for manual_approval_needed", () => {
    const keys = keysForEvent("manual_approval_needed");
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
      "ai_endpoint_down",
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

    expect(applied.length).toBe(2);
    expect(invalidate).toHaveBeenCalledTimes(2);
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
});
