import { describe, it, expect } from "vitest";
import {
  activeNavHref,
  isNavItemActive,
  USER_NAV_ITEMS,
  MOBILE_FAB_HREF,
  MOBILE_BAR_EXCLUDED_HREFS,
  MOBILE_BOTTOM_NAV_ITEMS,
} from "./nav";

describe("activeNavHref", () => {
  it("selects the most specific (longest) matching item", () => {
    // Regression: a nested route under /user/settings used to highlight BOTH
    // "Langganan" and the (now removed) "Pengaturan" entry.
    expect(activeNavHref("/user/subscription")).toBe("/user/subscription");
    // A sibling under /user/settings is not a child of any nav route, so it
    // falls back to the Dashboard.
    expect(activeNavHref("/user/settings/other")).toBe("/user");
  });

  it("matches exact routes", () => {
    expect(activeNavHref("/user")).toBe("/user");
    expect(activeNavHref("/user/transactions")).toBe("/user/transactions");
    expect(activeNavHref("/user/profile")).toBe("/user/profile");
    expect(activeNavHref("/user/usage")).toBe("/user/usage");
  });

  it("matches nested routes under an item", () => {
    expect(activeNavHref("/user/transactions/123")).toBe("/user/transactions");
  });

  it("returns null when nothing matches", () => {
    expect(activeNavHref("/login")).toBeNull();
    expect(activeNavHref("/")).toBeNull();
  });

  it("does not treat a sibling prefix as a child (no /user/settingsX -> /user/settings)", () => {
    // /user/settingsX is under /user (matches) but NOT under any nested route,
    // so it resolves to the Dashboard.
    expect(activeNavHref("/user/settingsX")).toBe("/user");
  });

  it("no longer has a separate Pengaturan entry (merged into profile)", () => {
    expect(USER_NAV_ITEMS.some((i) => i.href === "/user/settings")).toBe(false);
  });
});

describe("isNavItemActive", () => {
  it("activates exactly one item for the subscription route", () => {
    const active = USER_NAV_ITEMS.filter((i) => isNavItemActive("/user/subscription", i.href));
    expect(active.map((i) => i.label)).toEqual(["Langganan"]);
  });

  it("activates Penggunaan for the usage route", () => {
    const active = USER_NAV_ITEMS.filter((i) => isNavItemActive("/user/usage", i.href));
    expect(active.map((i) => i.label)).toEqual(["Penggunaan"]);
  });

  it("activates Profil on the merged profile route", () => {
    const active = USER_NAV_ITEMS.filter((i) => isNavItemActive("/user/profile", i.href));
    expect(active.map((i) => i.label)).toEqual(["Profil"]);
  });
});

describe("mobile bottom navigation split", () => {
  it("lifts AI Chat out of the bar into the FAB", () => {
    expect(MOBILE_FAB_HREF).toBe("/user/ai");
    expect(MOBILE_BOTTOM_NAV_ITEMS.some((i) => i.href === MOBILE_FAB_HREF)).toBe(false);
  });

  it("drops Profil from the bar (reached via the header avatar instead)", () => {
    expect(MOBILE_BOTTOM_NAV_ITEMS.some((i) => i.href === "/user/profile")).toBe(false);
    expect(MOBILE_BAR_EXCLUDED_HREFS).toContain("/user/profile");
  });

  it("keeps the remaining items in original order", () => {
    expect(MOBILE_BOTTOM_NAV_ITEMS.map((i) => i.label)).toEqual([
      "Dashboard",
      "Transaksi",
      "Laporan",
      "Penggunaan",
      "Langganan",
    ]);
  });

  it("keeps the bar small enough for full-width tap targets", () => {
    // At most five slots: six would push each below the ~44px minimum on a
    // 360px-wide phone.
    expect(MOBILE_BOTTOM_NAV_ITEMS.length).toBeLessThanOrEqual(5);
  });

  it("partitions the full nav into bar items, the FAB, and excluded routes", () => {
    const combined = [...MOBILE_BOTTOM_NAV_ITEMS.map((i) => i.href), ...MOBILE_BAR_EXCLUDED_HREFS].sort();
    const all = USER_NAV_ITEMS.map((i) => i.href).sort();
    expect(combined).toEqual(all);
  });
});
