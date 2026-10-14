import { describe, it, expect } from "vitest";
import { activeNavHref, isNavItemActive, USER_NAV_ITEMS } from "./nav";

describe("activeNavHref", () => {
  it("selects the most specific (longest) matching item", () => {
    // Regression: /user/settings/subscription used to highlight BOTH
    // "Langganan" and the (now removed) "Pengaturan" entry.
    expect(activeNavHref("/user/settings/subscription")).toBe("/user/settings/subscription");
    // A sibling under /user/settings is not a child of the subscription route,
    // so it falls back to the Dashboard.
    expect(activeNavHref("/user/settings/other")).toBe("/user");
  });

  it("matches exact routes", () => {
    expect(activeNavHref("/user")).toBe("/user");
    expect(activeNavHref("/user/transactions")).toBe("/user/transactions");
    expect(activeNavHref("/user/profile")).toBe("/user/profile");
  });

  it("matches nested routes under an item", () => {
    expect(activeNavHref("/user/transactions/123")).toBe("/user/transactions");
  });

  it("returns null when nothing matches", () => {
    expect(activeNavHref("/login")).toBeNull();
    expect(activeNavHref("/")).toBeNull();
  });

  it("does not treat a sibling prefix as a child (no /user/settingsX -> /user/settings)", () => {
    // /user/settingsX is under /user (matches) but NOT under
    // /user/settings/subscription, so it resolves to the Dashboard.
    expect(activeNavHref("/user/settingsX")).toBe("/user");
  });

  it("no longer has a separate Pengaturan entry (merged into profile)", () => {
    expect(USER_NAV_ITEMS.some((i) => i.href === "/user/settings")).toBe(false);
  });
});

describe("isNavItemActive", () => {
  it("activates exactly one item for a nested route", () => {
    const active = USER_NAV_ITEMS.filter((i) => isNavItemActive("/user/settings/subscription", i.href));
    expect(active.map((i) => i.label)).toEqual(["Langganan"]);
  });

  it("activates Profil on the merged profile route", () => {
    const active = USER_NAV_ITEMS.filter((i) => isNavItemActive("/user/profile", i.href));
    expect(active.map((i) => i.label)).toEqual(["Profil"]);
  });
});
