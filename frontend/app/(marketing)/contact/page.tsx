import type { Metadata } from "next";
import { ContactSections } from "@/components/marketing/ContactSections";
import { JsonLd } from "@/components/marketing/JsonLd";
import { PAGE_META } from "@/lib/marketing/copy";
import { buildPageMetadata, contactPageJsonLd } from "@/lib/seo";

/**
 * `/contact` (spec doc 05 §2). Backend-free: the page only exposes a `mailto:`
 * link and a `wa.me` link, both derived from `site.ts`.
 */
export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/contact"].title,
  description: PAGE_META["/contact"].description,
  path: "/contact",
});

export default function ContactPage() {
  return (
    <>
      <JsonLd id="contact-page-jsonld" data={contactPageJsonLd()} />
      <ContactSections />
    </>
  );
}
