import { describe, it, expect } from "vitest";
import sitemap from "./sitemap";
import { FEATURE_SLUGS } from "@/lib/marketing/features";

describe("app/sitemap.ts", () => {
  it("includes all 9 static marketing routes plus 6 feature sub-pages", () => {
    const entries = sitemap();
    expect(entries).toHaveLength(9 + 6);
  });

  it("lists all feature slugs in the sitemap", () => {
    const urls = sitemap().map((e) => e.url);
    FEATURE_SLUGS.forEach((slug) => {
      expect(urls.some((u) => u.endsWith(`/features/${slug}`))).toBe(true);
    });
  });

  it("uses valid ISO dates, priorities, and change frequencies", () => {
    const entries = sitemap();
    entries.forEach((e) => {
      expect(e.url).toMatch(/^https?:\/\//);
      expect(e.priority).toBeGreaterThanOrEqual(0.1);
      expect(e.priority).toBeLessThanOrEqual(1.0);
      expect(["weekly", "monthly", "yearly"]).toContain(e.changeFrequency);
      expect(e.lastModified).toBeInstanceOf(Date);
    });
  });

  it("never includes private or dashboard routes in the sitemap", () => {
    const blob = JSON.stringify(sitemap());
    ["/api", "/user", "/admin", "/super-admin", "/login"].forEach((priv) => {
      expect(blob).not.toContain(priv);
    });
  });
});
