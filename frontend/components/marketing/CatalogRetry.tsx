"use client";

import { useRouter } from "next/navigation";
import { RotateCw } from "lucide-react";
import { MICROCOPY } from "@/lib/marketing/copy";

/**
 * Graceful fallback for a failed catalog fetch (spec doc 04 §A.9). Client
 * component because the retry re-renders the Server Component tree via
 * `router.refresh()`. The page still renders the FAQ and CTA around this block,
 * so a backend hiccup never leaves an empty page.
 */
export function CatalogRetry() {
  const router = useRouter();

  return (
    <div className="mx-auto flex max-w-xl flex-col items-center gap-4 rounded-md border-[3px] border-foreground bg-muted p-8 text-center shadow-[6px_6px_0px_0px] shadow-foreground">
      <p className="text-sm font-semibold">{MICROCOPY.catalogError}</p>
      <button
        type="button"
        onClick={() => router.refresh()}
        className="inline-flex h-11 items-center justify-center gap-2 rounded-md border-[3px] border-foreground bg-accent-acid px-6 text-sm font-bold text-accent-acid-foreground shadow-[4px_4px_0px_0px] shadow-foreground transition-transform hover:bg-accent-acid-strong active:translate-x-[3px] active:translate-y-[3px] active:shadow-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
      >
        <RotateCw aria-hidden className="size-4" />
        {MICROCOPY.retry}
      </button>
    </div>
  );
}
