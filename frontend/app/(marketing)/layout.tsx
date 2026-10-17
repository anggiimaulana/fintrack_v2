import type { Metadata } from "next";
import { JsonLd } from "@/components/marketing/JsonLd";
import { MarketingHeader } from "@/components/marketing/MarketingHeader";
import { MarketingFooter } from "@/components/marketing/MarketingFooter";
import { PAGE_META } from "@/lib/marketing/copy";
import { organizationGraph } from "@/lib/seo";

/**
 * Route group for every public marketing page (`/`, `/pricing`, `/features/*`,
 * `/about`, `/contact`, `/faq`, `/privacy`, `/terms`).
 *
 * Emits the top-level Organization + WebSite JSON-LD graph once for the entire
 * marketing section (spec doc 07 §6.1). Provides the shared header/footer
 * chrome and the landing default metadata.
 */
export const metadata: Metadata = {
  title: { absolute: PAGE_META["/"].title },
  description: PAGE_META["/"].description,
};

export default function MarketingLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <JsonLd id="marketing-org-website-jsonld" data={organizationGraph()} />
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
