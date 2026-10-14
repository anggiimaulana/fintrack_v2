import { describe, it, expect } from "vitest";
import {
  notificationFor,
  appendNotification,
  unreadCount,
  MAX_NOTIFICATIONS,
  type AppNotification,
} from "./notifications";
import type { DomainEvent } from "./realtime-dispatcher";

const NOW = 1_700_000_000_000;

function ev(event: string, payload?: unknown, id?: string): DomainEvent {
  return { event, payload, id };
}

describe("notificationFor", () => {
  it("maps payment_received to a payment notification with an amount", () => {
    const n = notificationFor(ev("payment_received", { amount_idr: 39000, invoice_number: "INV/1" }, "e1"), NOW);
    expect(n).not.toBeNull();
    expect(n!.title).toMatch(/Pembayaran/i);
    expect(n!.body).toContain("39.000");
    expect(n!.href).toContain("/user/settings/subscription");
    expect(n!.read).toBe(false);
    expect(n!.id).toBe("e1");
  });

  it("maps plan_updated to a plan notification", () => {
    const n = notificationFor(ev("plan_updated", { current_plan: "Pro" }), NOW);
    expect(n!.title).toMatch(/Paket/i);
    expect(n!.body).toContain("Pro");
  });

  it("maps quota_reset with the allocated token count", () => {
    const n = notificationFor(ev("quota_reset", { allocated_tokens: 150000 }), NOW);
    expect(n!.title).toMatch(/Kuota/i);
    expect(n!.body).toContain("150.000");
  });

  it("maps manual_approval_needed for the admin review queue", () => {
    const n = notificationFor(ev("manual_approval_needed", { user_email: "a@b.c", amount_idr: 1000 }), NOW);
    expect(n!.href).toBe("/super-admin/payments/review");
  });

  it("maps ai_route_quarantined to an admin alert naming the model", () => {
    const n = notificationFor(
      ev("ai_route_quarantined", { use_case: "chat", model: "gemini-2.5-flash", reason: "503 dead" }),
      NOW,
    );
    expect(n).not.toBeNull();
    expect(n!.title).toMatch(/dikarantina/i);
    expect(n!.body).toContain("gemini-2.5-flash");
    expect(n!.body).toContain("503 dead");
    expect(n!.href).toBe("/super-admin/ai-gateway");
  });

  it("maps a recovered ai_route_quarantined to a recovery alert", () => {
    const n = notificationFor(
      ev("ai_route_quarantined", { use_case: "chat", model: "m", recovered: true }),
      NOW,
    );
    expect(n!.title).toMatch(/pulih/i);
    expect(n!.body).toContain("m");
  });

  it("returns null for an unknown event", () => {
    expect(notificationFor(ev("some_future_event"), NOW)).toBeNull();
  });

  it("never throws on a malformed payload", () => {
    expect(notificationFor(ev("payment_received", "not-an-object"), NOW)).not.toBeNull();
    expect(notificationFor(ev("quota_reset", null), NOW)).not.toBeNull();
  });

  it("synthesises an id when the event has none", () => {
    const n = notificationFor(ev("plan_updated"), NOW);
    expect(n!.id).toContain("plan_updated");
  });
});

describe("appendNotification", () => {
  const make = (id: string, read = false): AppNotification => ({
    id,
    event: "plan_updated",
    title: "t",
    body: "b",
    createdAt: NOW,
    href: "/x",
    read,
  });

  it("prepends the newest notification", () => {
    const list = appendNotification([make("a")], make("b"));
    expect(list.map((n) => n.id)).toEqual(["b", "a"]);
  });

  it("de-duplicates by id", () => {
    const list = appendNotification([make("a")], make("a"));
    expect(list).toHaveLength(1);
  });

  it("caps the list length", () => {
    let list: AppNotification[] = [];
    for (let i = 0; i < MAX_NOTIFICATIONS + 5; i++) list = appendNotification(list, make(`n${i}`));
    expect(list).toHaveLength(MAX_NOTIFICATIONS);
  });
});

describe("unreadCount", () => {
  it("counts only unread items", () => {
    const items: AppNotification[] = [
      { id: "1", event: "e", title: "t", body: "b", createdAt: 0, href: "/", read: false },
      { id: "2", event: "e", title: "t", body: "b", createdAt: 0, href: "/", read: true },
      { id: "3", event: "e", title: "t", body: "b", createdAt: 0, href: "/", read: false },
    ];
    expect(unreadCount(items)).toBe(2);
    expect(unreadCount([])).toBe(0);
  });
});
