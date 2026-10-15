import type { LucideIcon } from "lucide-react";
import { Home, ArrowLeftRight, FileText, MessageSquare, CreditCard, User, BarChart3 } from "lucide-react";

/** One entry in the user dashboard navigation. */
export interface NavItem {
  href: string;
  icon: LucideIcon;
  label: string;
}

/** The user dashboard sidebar / header navigation, in display order. */
export const USER_NAV_ITEMS: NavItem[] = [
  { href: "/user", icon: Home, label: "Dashboard" },
  { href: "/user/transactions", icon: ArrowLeftRight, label: "Transaksi" },
  { href: "/user/ai", icon: MessageSquare, label: "AI Chat" },
  { href: "/user/reports", icon: FileText, label: "Laporan" },
  { href: "/user/usage", icon: BarChart3, label: "Penggunaan" },
  { href: "/user/subscription", icon: CreditCard, label: "Langganan" },
  // Profile and settings are one page now (accordion groups); the old
  // /user/settings route redirects here, so no separate nav entry.
  { href: "/user/profile", icon: User, label: "Profil" },
];

/**
 * Returns the single active navigation href for a pathname, using longest-prefix
 * matching so a nested route highlights only its most specific item.
 *
 * Without this, `/user/settings/subscription` would match both
 * `/user/settings/subscription` (Langganan) and its parent, lighting up two menu
 * entries at once. A route matches an item when it equals the item's href or is
 * a child of it (`href + "/"`); when several match, the longest href wins.
 */
export function activeNavHref(pathname: string, items: NavItem[] = USER_NAV_ITEMS): string | null {
  let best: string | null = null;
  for (const item of items) {
    const matches = pathname === item.href || pathname.startsWith(`${item.href}/`);
    if (matches && (best === null || item.href.length > best.length)) {
      best = item.href;
    }
  }
  return best;
}

/**
 * The route promoted to the floating action button on mobile. It is pulled out
 * of the bottom bar so the remaining items split the width evenly.
 */
export const MOBILE_FAB_HREF = "/user/ai";

/**
 * Routes deliberately kept out of the mobile bottom bar because they already
 * have a dedicated entry point there:
 * - `/user/ai` is lifted into the floating action button.
 * - `/user/profile` is reached by tapping the header avatar (ProfileMenu), which
 *   links to the same page, so a bar slot would be redundant.
 */
export const MOBILE_BAR_EXCLUDED_HREFS: readonly string[] = [MOBILE_FAB_HREF, "/user/profile"];

/**
 * Items rendered in the mobile bottom bar: every user nav item that is not
 * excluded above. Order follows `USER_NAV_ITEMS`, so Dashboard sits first.
 */
export const MOBILE_BOTTOM_NAV_ITEMS: NavItem[] = USER_NAV_ITEMS.filter(
  (item) => !MOBILE_BAR_EXCLUDED_HREFS.includes(item.href),
);

/** Reports whether a nav item is the active one for the current pathname. */
export function isNavItemActive(pathname: string, href: string, items: NavItem[] = USER_NAV_ITEMS): boolean {
  return activeNavHref(pathname, items) === href;
}
