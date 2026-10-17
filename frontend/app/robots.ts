import type { MetadataRoute } from "next";

/**
 * Public robots.txt (spec doc 07 §4).
 *
 * Rules:
 * - Allow all public marketing pages and `/register`.
 * - Disallow private API endpoints, authenticated user dashboard, admin, and login.
 * - AI search crawlers (Googlebot, Bingbot, GPTBot, PerplexityBot) are NOT blocked,
 *   preserving citability and GEO signals.
 */
export default function robots(): MetadataRoute.Robots {
  const siteUrl = (process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3001").replace(/\/+$/, "");

  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: ["/api/", "/user", "/admin", "/super-admin", "/login"],
      },
    ],
    sitemap: `${siteUrl}/sitemap.xml`,
    host: siteUrl,
  };
}
