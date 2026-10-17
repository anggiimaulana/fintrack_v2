import type { Metadata } from "next";
import { ExploreHub } from "@/components/marketing/ExploreHub";
import { PAGE_META } from "@/lib/marketing/copy";
import { buildPageMetadata } from "@/lib/seo";

export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/explore"].title,
  description: PAGE_META["/explore"].description,
  path: "/explore",
});

export default function ExplorePage() {
  return <ExploreHub />;
}
