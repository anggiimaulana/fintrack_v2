import { describe, it, expect } from "vitest";
import { notificationsListKey } from "./useNotifications";

/**
 * The realtime cache-patching logic patches specific query keys. These tests pin
 * the KEY SHAPE, because a mismatch between the key used to write and the key
 * used to read is a silent no-op (the exact bug class found earlier in the
 * gateway dispatcher).
 */
describe("notificationsListKey", () => {
  it("is stable for the same filter set regardless of property order", () => {
    const a = notificationsListKey({ page: 2, perPage: 20, sort: "created_at", order: "desc", search: "x", status: "unread" });
    const b = notificationsListKey({ status: "unread", search: "x", order: "desc", sort: "created_at", perPage: 20, page: 2 });
    expect(a).toBe(b);
  });

  it("differs when any meaningful field differs", () => {
    const base = notificationsListKey({ page: 1, perPage: 20 });
    expect(notificationsListKey({ page: 2, perPage: 20 })).not.toBe(base);
    expect(notificationsListKey({ page: 1, perPage: 50 })).not.toBe(base);
    expect(notificationsListKey({ page: 1, perPage: 20, status: "unread" })).not.toBe(base);
    expect(notificationsListKey({ page: 1, perPage: 20, search: "q" })).not.toBe(base);
  });

  it("applies documented defaults", () => {
    // The default key is what patchCreated targets for a new row.
    const withDefaults = notificationsListKey({ page: 1, perPage: 20, sort: "created_at", order: "desc" });
    expect(withDefaults).toBe(notificationsListKey({ page: 1, perPage: 20 }));
    expect(withDefaults).toContain('"sort":"created_at"');
    expect(withDefaults).toContain('"order":"desc"');
    expect(withDefaults).toContain('"status":"all"');
  });
});
