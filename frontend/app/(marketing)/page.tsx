import type { Metadata } from "next";
import { LandingSections } from "@/components/marketing/LandingSections";
import { PAGE_META } from "@/lib/marketing/copy";
import { fetchPublicPlans } from "@/lib/plans-server";
import { buildPageMetadata } from "@/lib/seo";

/**
 * Landing page (`/`). Server component: the plan catalog is fetched on the
 * server with ISR and handed to the pricing teaser, so prices always match the
 * public endpoint and no client JavaScript is needed to show them.
 */
export const revalidate = 3600;

export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/"].title,
  description: PAGE_META["/"].description,
  path: "/",
});

export default async function HomePage() {
  const plans = await fetchPublicPlans();
  return <LandingSections plans={plans} />;
}
