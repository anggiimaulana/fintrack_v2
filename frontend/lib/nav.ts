import type { LucideIcon } from "lucide-react";
import { Home, ArrowLeftRight, FileText, MessageSquare, CreditCard, User } from "lucide-react";

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
  { href: "/user/settings/subscription", icon: CreditCard, label: "Langganan" },
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

/** Reports whether a nav item is the active one for the current pathname. */
export function isNavItemActive(pathname: string, href: string, items: NavItem[] = USER_NAV_ITEMS): boolean {
  return activeNavHref(pathname, items) === href;
}
