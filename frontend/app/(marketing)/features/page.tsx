import type { Metadata } from "next";
import { FeaturesHub } from "@/components/marketing/FeaturesHub";
import { PAGE_META } from "@/lib/marketing/copy";
import { buildPageMetadata } from "@/lib/seo";

/**
 * `/features` hub (spec doc 04 §B.7). Fully static — the six cards come from the
 * shared feature model, so there is nothing to fetch at request time.
 */
export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/features"].title,
  description: PAGE_META["/features"].description,
  path: "/features",
});

export default function FeaturesPage() {
  return <FeaturesHub />;
}
