import type { Metadata } from "next";
import type { PublicPlan } from "@/lib/plans-types";
import { site } from "@/lib/marketing/site";

/**
 * SEO helpers: per-page metadata and JSON-LD builders (spec doc 07).
 *
 * Deliberately NOT implemented: `FAQPage`, `Review`, `AggregateRating`, and
 * `SearchAction` — Google retired the FAQ rich result, we have no genuine
 * reviews, and the site has no internal search.
 */

export interface BreadcrumbItem {
  name: string;
  path: string;
}

/** JSON-LD node (loosely typed; consumed by the `JsonLd` component). */
export type JsonLdNode = Record<string, unknown>;

/** Join a site-relative path onto the canonical origin. */
export function absoluteUrl(path: string): string {
  if (/^https?:\/\//.test(path)) return path;
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return `${site.url}${normalized}`;
}

export interface PageMetadataInput {
  title: string;
  description: string;
  /** Site-relative path, e.g. `/pricing`. */
  path: string;
  keywords?: string[];
}

/**
 * Build per-page metadata with a canonical link and OG/Twitter defaults.
 *
 * `title` is emitted as `{ absolute }` on purpose: the canonical titles in
 * spec doc 13 §11 already contain the brand, so the root template
 * ("%s · FinTrack") must not append it again.
 */
export function buildPageMetadata({ title, description, path, keywords }: PageMetadataInput): Metadata {
  return {
    title: { absolute: title },
    description,
    keywords,
    alternates: { canonical: path },
    openGraph: {
      type: "website",
      siteName: site.name,
      locale: "id_ID",
      url: absoluteUrl(path),
      title,
      description,
    },
    twitter: {
      card: "summary_large_image",
      title,
      description,
    },
  };
}

/** Organization + WebSite graph, emitted once by the marketing layout. */
export function organizationGraph(): JsonLdNode {
  const orgId = `${site.url}/#organization`;
  return {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "Organization",
        "@id": orgId,
        name: site.name,
        url: `${site.url}/`,
        logo: `${site.url}/logo.png`,
        contactPoint: [
          {
            "@type": "ContactPoint",
            contactType: "customer support",
            email: site.email,
            availableLanguage: ["id"],
          },
        ],
      },
      {
        "@type": "WebSite",
        "@id": `${site.url}/#website`,
        url: `${site.url}/`,
        name: site.name,
        inLanguage: "id-ID",
        publisher: { "@id": orgId },
      },
    ],
  };
}

/**
 * Product + Offer markup for `/pricing`, generated from the live catalog so
 * prices always match the endpoint. `price` is the monthly subtotal (tax, when
 * any, is added at checkout and stated in the description).
 */
export function productJsonLd(plans: PublicPlan[]): JsonLdNode {
  const ordered = [...plans].sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0));
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: site.name,
    description: "Aplikasi pencatat keuangan otomatis dengan AI berbahasa Indonesia.",
    brand: { "@type": "Brand", name: site.name },
    offers: ordered.map((plan) => ({
      "@type": "Offer",
      name: plan.name,
      price: String(plan.price_monthly_idr),
      priceCurrency: "IDR",
      url: absoluteUrl("/pricing"),
      availability: "https://schema.org/InStock",
    })),
  };
}

/** BreadcrumbList for any child page. */
export function breadcrumbJsonLd(items: BreadcrumbItem[]): JsonLdNode {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: items.map((item, index) => ({
      "@type": "ListItem",
      position: index + 1,
      name: item.name,
      item: absoluteUrl(item.path),
    })),
  };
}

/**
 * AboutPage plus a self-contained Organization (name, url, logo, contact) as
 * required by spec doc 05 §1.3. Kept inline so the node is valid even before the
 * shared `organizationGraph()` is wired into the marketing layout.
 */
export function aboutPageJsonLd(): JsonLdNode {
  return {
    "@context": "https://schema.org",
    "@type": "AboutPage",
    name: `Tentang ${site.name}`,
    inLanguage: "id-ID",
    url: absoluteUrl("/about"),
    mainEntity: {
      "@type": "Organization",
      name: site.name,
      url: `${site.url}/`,
      logo: `${site.url}/logo.png`,
      contactPoint: {
        "@type": "ContactPoint",
        contactType: "customer support",
        email: site.email,
        availableLanguage: ["id"],
      },
    },
  };
}

/**
 * WebPage for a feature deep-dive, with `about` naming the capability and
 * `isPartOf` pointing at the WebSite node (spec doc 04 §B.6). No `FAQPage` here
 * either — the visible example/prompt text is plain content.
 */
export function featurePageJsonLd(input: {
  name: string;
  path: string;
  description: string;
  about: string;
}): JsonLdNode {
  return {
    "@context": "https://schema.org",
    "@type": "WebPage",
    name: input.name,
    description: input.description,
    inLanguage: "id-ID",
    url: absoluteUrl(input.path),
    isPartOf: { "@id": `${site.url}/#website` },
    about: { "@type": "Thing", name: input.about },
  };
}

/** ContactPage with the support contact point. */
export function contactPageJsonLd(): JsonLdNode {
  return {
    "@context": "https://schema.org",
    "@type": "ContactPage",
    name: `Kontak ${site.name}`,
    inLanguage: "id-ID",
    url: absoluteUrl("/contact"),
    about: {
      "@type": "Organization",
      name: site.name,
      url: `${site.url}/`,
      contactPoint: {
        "@type": "ContactPoint",
        contactType: "customer support",
        email: site.email,
        availableLanguage: ["id"],
      },
    },
  };
}

/** Simple WebPage node for FAQ and legal pages. */
export function webPageJsonLd(name: string, path: string, description: string): JsonLdNode {
  return {
    "@context": "https://schema.org",
    "@type": "WebPage",
    name,
    description,
    inLanguage: "id-ID",
    url: absoluteUrl(path),
  };
}
