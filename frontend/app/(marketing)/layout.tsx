import type { Metadata } from "next";
import { MarketingHeader } from "@/components/marketing/MarketingHeader";
import { MarketingFooter } from "@/components/marketing/MarketingFooter";
import { PAGE_META } from "@/lib/marketing/copy";

/**
 * Route group for every public marketing page (`/`, `/pricing`, `/features/*`,
 * `/about`, `/contact`, `/faq`, `/privacy`, `/terms`).
 *
 * Provides the shared header/footer chrome and the landing default metadata.
 * Individual pages override the title/description via `buildPageMetadata`
 * (see `lib/seo.ts`).
 */
export const metadata: Metadata = {
  title: { absolute: PAGE_META["/"].title },
  description: PAGE_META["/"].description,
};

export default function MarketingLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[60] focus:rounded-md focus:border-2 focus:border-foreground focus:bg-accent-acid focus:px-4 focus:py-2 focus:text-sm focus:font-bold focus:text-accent-acid-foreground"
      >
        Lewati ke konten
      </a>
      <MarketingHeader />
      <main id="main" className="flex-1">
        {children}
      </main>
      <MarketingFooter />
    </div>
  );
}
