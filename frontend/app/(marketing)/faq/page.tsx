import type { Metadata } from "next";
import { FaqSections } from "@/components/marketing/FaqSections";
import { JsonLd } from "@/components/marketing/JsonLd";
import { PAGE_META } from "@/lib/marketing/copy";
import { buildPageMetadata, webPageJsonLd } from "@/lib/seo";

/**
 * `/faq` (spec doc 05 §3). Renders grouped Q&A as plain text/`<details>` and
 * emits only `WebPage` + `BreadcrumbList` — never `FAQPage`.
 */
export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/faq"].title,
  description: PAGE_META["/faq"].description,
  path: "/faq",
});

export default function FaqPage() {
  return (
    <>
      <JsonLd
        id="faq-webpage-jsonld"
        data={webPageJsonLd(PAGE_META["/faq"].title, "/faq", PAGE_META["/faq"].description)}
      />
      <FaqSections />
    </>
  );
}
