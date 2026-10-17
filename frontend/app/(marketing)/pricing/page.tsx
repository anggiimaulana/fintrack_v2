import type { Metadata } from "next";
import { JsonLd } from "@/components/marketing/JsonLd";
import { PricingSections } from "@/components/marketing/PricingSections";
import { PAGE_META } from "@/lib/marketing/copy";
import { fetchPublicPlans } from "@/lib/plans-server";
import { buildPageMetadata, productJsonLd } from "@/lib/seo";

/**
 * `/pricing` (spec doc 04 §A). Server component: the public plan catalog is
 * fetched with ISR so prices, limits, and the tax line always match the
 * endpoint, and `Product`/`Offer` JSON-LD is generated from the same data.
 */
export const revalidate = 3600;

export const metadata: Metadata = buildPageMetadata({
  title: PAGE_META["/pricing"].title,
  description: PAGE_META["/pricing"].description,
  path: "/pricing",
});

export default async function PricingPage() {
  const plans = await fetchPublicPlans();

  return (
    <>
      {/* Only emit an Offer list when there is a real catalog to describe. */}
      {plans.length > 0 ? <JsonLd id="pricing-product-jsonld" data={productJsonLd(plans)} /> : null}
      <PricingSections plans={plans} />
    </>
  );
}
