import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { JsonLd } from "@/components/marketing/JsonLd";
import { FeatureDetail } from "@/components/marketing/FeatureDetail";
import { FEATURES, featureHref, getFeature } from "@/lib/marketing/features";
import { buildPageMetadata, featurePageJsonLd } from "@/lib/seo";

/**
 * `/features/{slug}` deep-dive (spec doc 04 §B.4–§B.6). Statically generated for
 * the six known slugs; `dynamicParams = false` makes any unknown slug 404
 * instead of rendering an empty page.
 */
export const dynamicParams = false;

export function generateStaticParams() {
  return FEATURES.map((feature) => ({ slug: feature.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const feature = getFeature(slug);
  if (!feature) return {};

  return buildPageMetadata({
    title: `${feature.title} — FinTrack`,
    description: feature.metaDescription,
    path: featureHref(feature.slug),
    keywords: feature.keywords,
  });
}

export default async function FeaturePage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const feature = getFeature(slug);
  if (!feature) notFound();

  return (
    <>
      <JsonLd
        id="feature-webpage-jsonld"
        data={featurePageJsonLd({
          name: `${feature.title} — FinTrack`,
          path: featureHref(feature.slug),
          description: feature.metaDescription,
          about: feature.navLabel,
        })}
      />
      <FeatureDetail feature={feature} />
    </>
  );
}
