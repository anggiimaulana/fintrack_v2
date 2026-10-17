"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Compass, Home, LayoutGrid, Tag, Workflow } from "lucide-react";
import { cn } from "@/lib/utils";

const MOBILE_NAV_ITEMS = [
  { label: "Beranda", href: "/", icon: Home },
  { label: "Fitur", href: "/features", icon: LayoutGrid },
  { label: "Cara Kerja", href: "/#how-it-works", icon: Workflow },
  { label: "Harga", href: "/pricing", icon: Tag },
  { label: "Jelajahi", href: "/explore", icon: Compass },
];

export function MarketingMobileBottomNav() {
  const pathname = usePathname();

  const isItemActive = (href: string) => {
    if (href === "/") return pathname === "/";
    if (href.startsWith("/#")) return false;
    return pathname === href || pathname.startsWith(`${href}/`);
  };

  return (
    <nav
      aria-label="Navigasi bawah seluler"
      className="fixed bottom-0 left-0 right-0 z-40 border-t-[3px] border-foreground bg-background pb-[env(safe-area-inset-bottom)] md:hidden"
    >
      <ul className="flex items-stretch justify-around">
        {MOBILE_NAV_ITEMS.map((item) => {
          const active = isItemActive(item.href);
          const Icon = item.icon;
          return (
            <li key={item.label} className="flex-1 min-w-0">
              <Link
                href={item.href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex h-16 min-w-0 flex-col items-center justify-center gap-1 px-1 transition-colors",
                  active
                    ? "font-extrabold text-primary"
                    : "font-bold text-muted-foreground hover:text-foreground",
                )}
              >
                <Icon className={cn("size-5 shrink-0", active && "stroke-[2.5]")} aria-hidden="true" />
                <span className="w-full truncate text-center text-[10px] tracking-tight">{item.label}</span>
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
