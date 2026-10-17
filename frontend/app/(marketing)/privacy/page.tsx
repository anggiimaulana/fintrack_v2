import type { Metadata } from "next";
import { LegalPage } from "@/components/marketing/LegalPage";
import { PAGE_META } from "@/lib/marketing/copy";
import { PRIVACY } from "@/lib/marketing/legal";
import { buildPageMetadata } from "@/lib/seo";

/**
 * `/privacy` (spec doc 05 §4, §6). Long, stable copy → rendered statically and
 * indexed (trust surface). Emits `WebPage` via `LegalPage`.
 */
export const dynamic = "force-static";

export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/privacy"].title,
  description: PAGE_META["/privacy"].description,
  path: "/privacy",
});

export default function PrivacyPage() {
  return <LegalPage doc={PRIVACY} />;
}
