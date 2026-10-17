import type { MetadataRoute } from "next";
import { FEATURES } from "@/lib/marketing/features";

/**
 * Public sitemap.xml (spec doc 07 §5).
 *
 * Covers all 8 canonical marketing pages + 6 deep-dive feature pages.
 * Never includes private routes (`/user`, `/admin`, `/login`, etc.).
 */
export default function sitemap(): MetadataRoute.Sitemap {
  const siteUrl = (process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3001").replace(/\/+$/, "");
  const now = new Date();

  const staticRoutes: MetadataRoute.Sitemap = [
    { url: `${siteUrl}/`, changeFrequency: "weekly", priority: 1.0, lastModified: now },
    { url: `${siteUrl}/pricing`, changeFrequency: "weekly", priority: 0.9, lastModified: now },
    { url: `${siteUrl}/features`, changeFrequency: "monthly", priority: 0.8, lastModified: now },
    { url: `${siteUrl}/explore`, changeFrequency: "monthly", priority: 0.8, lastModified: now },
    { url: `${siteUrl}/faq`, changeFrequency: "monthly", priority: 0.7, lastModified: now },
    { url: `${siteUrl}/about`, changeFrequency: "yearly", priority: 0.5, lastModified: now },
    { url: `${siteUrl}/contact`, changeFrequency: "yearly", priority: 0.5, lastModified: now },
    { url: `${siteUrl}/privacy`, changeFrequency: "yearly", priority: 0.3, lastModified: now },
    { url: `${siteUrl}/terms`, changeFrequency: "yearly", priority: 0.3, lastModified: now },
  ];

  const featureRoutes: MetadataRoute.Sitemap = FEATURES.map((f) => ({
    url: `${siteUrl}/features/${f.slug}`,
    changeFrequency: "monthly",
    priority: 0.7,
    lastModified: now,
  }));

  return [...staticRoutes, ...featureRoutes];
}
