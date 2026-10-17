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

      {/* Backdrop overlay */}
      {open ? (
        <div
          aria-hidden="true"
          onClick={() => setOpenedPath(null)}
          className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs md:hidden"
        />
      ) : null}

      {/* Right-side mobile sidebar drawer */}
      <div
        id="marketing-mobile-menu"
        hidden={!open}
        className={cn(
          "fixed inset-y-0 right-0 z-50 flex w-[280px] sm:w-[320px] flex-col justify-between border-l-[3px] border-foreground bg-background p-6 shadow-[-6px_0px_0px_0px] shadow-foreground md:hidden",
        )}
      >
        <div className="flex flex-col gap-6">
          <div className="flex items-center justify-between border-b-2 border-foreground pb-4">
            <div className="flex items-center gap-2">
              <span className="grid size-7 place-items-center rounded-md border-2 border-foreground bg-accent-acid shadow-[2px_2px_0px_0px] shadow-foreground">
                <Wallet aria-hidden className="size-3.5 text-accent-acid-foreground" />
              </span>
              <span className="font-heading text-lg font-black tracking-tight">FinTrack</span>
            </div>
            <button
              type="button"
              className="grid size-8 place-items-center rounded-md border-2 border-foreground bg-background shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[2px] active:translate-y-[2px] active:shadow-none"
              aria-label="Tutup navigasi"
              onClick={() => setOpenedPath(null)}
            >
              <X aria-hidden className="size-4" />
            </button>
          </div>

          <nav aria-label="Navigasi seluler" className="flex flex-col gap-2">
            {NAV.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                aria-current={isActive(item.href) ? "page" : undefined}
                onClick={() => setOpenedPath(null)}
                className={cn(
                  "rounded-md border-2 px-4 py-2.5 text-sm font-bold uppercase tracking-wide transition-all",
                  isActive(item.href)
                    ? "border-foreground bg-accent-acid text-accent-acid-foreground shadow-[3px_3px_0px_0px] shadow-foreground"
                    : "border-transparent text-foreground hover:border-foreground hover:bg-muted",
                )}
              >
                {item.label}
              </Link>
            ))}
          </nav>
        </div>

        <div className="flex flex-col gap-3 border-t-2 border-foreground pt-4">
          <Button variant="outline" asChild className="w-full font-bold">
            <Link href={SIGN_IN_CTA.href}>{SIGN_IN_CTA.label}</Link>
          </Button>
          <Button asChild className="w-full font-bold">
            <Link href={PRIMARY_CTA.href}>{PRIMARY_CTA.label}</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}
