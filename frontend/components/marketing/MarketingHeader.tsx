"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Menu, Wallet, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { NAV, PRIMARY_CTA, SIGN_IN_CTA } from "@/lib/marketing/copy";
import { cn } from "@/lib/utils";

/**
 * Public marketing header: sticky, neobrutalist, with a collapsible mobile menu.
 *
 * Client component because it owns the menu open state and reads the current
 * path to mark the active link.
 */
export function MarketingHeader() {
  const pathname = usePathname();
  // The panel is open only while the current path matches the path it was
  // opened on. Navigating changes `pathname`, so the panel closes on its own
  // — no effect (which would be a synchronous setState-in-effect).
  const [openedPath, setOpenedPath] = React.useState<string | null>(null);
  const open = openedPath === pathname;

  // Lock body scroll while the mobile panel is open.
  React.useEffect(() => {
    if (!open) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [open]);

  // Close the mobile panel on Escape (keyboard users).
  React.useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpenedPath(null);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [open]);

  const isActive = (href: string) =>
    !href.includes("#") && (pathname === href || pathname.startsWith(`${href}/`));

  return (
    <header className="sticky top-0 z-50 w-full border-b-[3px] border-foreground bg-background">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4">
        <Link href="/" className="flex shrink-0 items-center gap-2" aria-label="FinTrack — beranda">
          <span className="grid size-8 place-items-center rounded-md border-2 border-foreground bg-accent-acid shadow-[3px_3px_0px_0px] shadow-foreground">
            <Wallet aria-hidden className="size-4 text-accent-acid-foreground" />
          </span>
          <span className="font-heading text-xl font-black tracking-tight">FinTrack</span>
        </Link>

        <nav aria-label="Navigasi utama" className="hidden items-center gap-6 md:flex">
          {NAV.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              aria-current={isActive(item.href) ? "page" : undefined}
              className={cn(
                "text-sm font-bold uppercase tracking-wide transition-colors",
                isActive(item.href)
                  ? "text-foreground underline decoration-accent-acid decoration-[3px] underline-offset-4"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="hidden items-center gap-3 md:flex">
          <Button variant="outline" asChild>
            <Link href={SIGN_IN_CTA.href}>{SIGN_IN_CTA.label}</Link>
          </Button>
          <Button asChild>
            <Link href={PRIMARY_CTA.href}>{PRIMARY_CTA.label}</Link>
          </Button>
        </div>

        <button
          type="button"
          className="grid size-10 place-items-center rounded-md border-2 border-foreground bg-background shadow-[3px_3px_0px_0px] shadow-foreground active:translate-x-[3px] active:translate-y-[3px] active:shadow-none md:hidden"
          aria-label={open ? "Tutup menu" : "Buka menu"}
          aria-expanded={open}
          aria-controls="marketing-mobile-menu"
          onClick={() => setOpenedPath(open ? null : pathname)}
        >
          {open ? <X aria-hidden className="size-5" /> : <Menu aria-hidden className="size-5" />}
        </button>
      </div>

      <div
        id="marketing-mobile-menu"
        hidden={!open}
        className="border-t-[3px] border-foreground bg-background md:hidden"
      >
        <nav aria-label="Navigasi seluler" className="flex flex-col gap-1 p-4">
          {NAV.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              aria-current={isActive(item.href) ? "page" : undefined}
              className={cn(
                "rounded-md px-3 py-2 text-sm font-bold uppercase tracking-wide",
                isActive(item.href) ? "bg-muted text-foreground" : "text-muted-foreground",
              )}
            >
              {item.label}
            </Link>
          ))}
          <div className="mt-2 flex flex-col gap-2">
            <Button variant="outline" asChild>
              <Link href={SIGN_IN_CTA.href}>{SIGN_IN_CTA.label}</Link>
            </Button>
            <Button asChild>
              <Link href={PRIMARY_CTA.href}>{PRIMARY_CTA.label}</Link>
            </Button>
          </div>
        </nav>
      </div>
    </header>
  );
}
