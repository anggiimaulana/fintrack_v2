import type { Metadata } from "next";
import { LegalPage } from "@/components/marketing/LegalPage";
import { PAGE_META } from "@/lib/marketing/copy";
import { TERMS } from "@/lib/marketing/legal";
import { buildPageMetadata } from "@/lib/seo";

/**
 * `/terms` (spec doc 05 §5, §6). Static and indexed. Emits `WebPage` via
 * `LegalPage`.
 */
export const dynamic = "force-static";

export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/terms"].title,
  description: PAGE_META["/terms"].description,
  path: "/terms",
});

export default function TermsPage() {
  return <LegalPage doc={TERMS} />;
}
