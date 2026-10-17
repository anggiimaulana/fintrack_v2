import type { Metadata } from "next";
import { AboutSections } from "@/components/marketing/AboutSections";
import { JsonLd } from "@/components/marketing/JsonLd";
import { PAGE_META } from "@/lib/marketing/copy";
import { aboutPageJsonLd, buildPageMetadata } from "@/lib/seo";

/**
 * `/about` (spec doc 05 §1). Static trust page; emits `AboutPage` +
 * `Organization` (the visible trail supplies the `BreadcrumbList`).
 */
export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/about"].title,
  description: PAGE_META["/about"].description,
  path: "/about",
});

export default function AboutPage() {
  return (
    <>
      <JsonLd id="about-page-jsonld" data={aboutPageJsonLd()} />
      <AboutSections />
    </>
  );
}
