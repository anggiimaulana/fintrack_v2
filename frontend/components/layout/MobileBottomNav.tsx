"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { MessageSquare } from "lucide-react";
import { MOBILE_BOTTOM_NAV_ITEMS, MOBILE_FAB_HREF, isNavItemActive } from "@/lib/nav";
import { cn } from "@/lib/utils";

/**
 * Mobile-only primary navigation: a fixed bar of five destinations plus a
 * floating action button for AI Chat.
 *
 * The bar is capped at five slots so each keeps a full-width, ~44px tap target
 * on a 360px phone. Two routes are therefore surfaced elsewhere: AI Chat — the
 * one action that is a verb rather than a place — becomes the FAB, and Profil is
 * reached by tapping the header avatar (ProfileMenu links to the same page).
 *
 * The FAB hides on its own route: once you are already in the chat, a shortcut
 * to the chat is dead weight that covers content. The bar itself is hidden from
 * `md` up, where the docked sidebar takes over.
 */
export function MobileBottomNav() {
  const pathname = usePathname();
  const onAiRoute = isNavItemActive(pathname, MOBILE_FAB_HREF);

  return (
    <>
      <nav
        aria-label="Navigasi utama"
        className="md:hidden fixed bottom-0 left-0 right-0 z-40 bg-card border-t-2 border-foreground pb-[env(safe-area-inset-bottom)]"
      >
        <ul className="flex items-stretch">
          {MOBILE_BOTTOM_NAV_ITEMS.map((item) => {
            const isActive = isNavItemActive(pathname, item.href);
            return (
              <li key={item.href} className="flex-1 min-w-0">
                <Link
                  href={item.href}
                  aria-current={isActive ? "page" : undefined}
                  className={cn(
                    "flex flex-col items-center justify-center gap-1 h-16 min-w-0 px-1 transition-colors cursor-pointer",
                    isActive ? "text-primary" : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  <item.icon className="w-5 h-5 shrink-0" aria-hidden="true" />
                  <span className="text-[10px] font-bold truncate w-full text-center">{item.label}</span>
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {!onAiRoute && (
        <Link
          href={MOBILE_FAB_HREF}
          aria-label="Buka AI Chat"
          className="md:hidden fixed right-4 bottom-[calc(4.5rem+env(safe-area-inset-bottom))] z-40 flex items-center justify-center w-14 h-14 rounded-full bg-primary text-primary-foreground border-2 border-foreground shadow-[3px_3px_0px_0px] shadow-foreground active:translate-x-[2px] active:translate-y-[2px] active:shadow-none transition-transform cursor-pointer"
        >
          <MessageSquare className="w-6 h-6" aria-hidden="true" />
        </Link>
      )}
    </>
  );
}
