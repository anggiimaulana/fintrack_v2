import { describe, it, expect } from "vitest";
import { FAQ_CATEGORIES, LANDING_FAQ, PRICING_FAQ } from "./faq";

describe("FAQ_CATEGORIES", () => {
  it("has at least four categories with unique anchors", () => {
    expect(FAQ_CATEGORIES.length).toBeGreaterThanOrEqual(4);
    const ids = FAQ_CATEGORIES.map((c) => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("gives every item a non-empty question and answer", () => {
    FAQ_CATEGORIES.forEach((category) => {
      expect(category.title.length).toBeGreaterThan(0);
      expect(category.items.length).toBeGreaterThanOrEqual(2);
      category.items.forEach((item) => {
        expect(item.q.trim().length).toBeGreaterThan(8);
        expect(item.a.trim().length).toBeGreaterThan(40);
      });
    });
  });

  it("uses internal links that point to real public routes", () => {
    const hrefs = FAQ_CATEGORIES.flatMap((c) => c.items.flatMap((i) => i.links ?? []).map((l) => l.href));
    hrefs.forEach((href) => {
      expect(href.startsWith("/")).toBe(true);
      // No dead "#" links.
      expect(href).not.toBe("#");
    });
  });

  it("respects the calm tone (no 'kamu', no '!' runs)", () => {
    const text = JSON.stringify(FAQ_CATEGORIES);
    expect(text).not.toMatch(/\bkamu\b/i);
    expect(text).not.toContain("!!");
  });
});

describe("LANDING_FAQ", () => {
  it("has six answer-first items", () => {
    expect(LANDING_FAQ).toHaveLength(6);
    LANDING_FAQ.forEach((item) => {
      expect(item.q.length).toBeGreaterThan(8);
      expect(item.a.length).toBeGreaterThan(30);
    });
  });

  it("never promises an unavailable feature", () => {
    const text = JSON.stringify(LANDING_FAQ).toLowerCase();
    expect(text).not.toContain("sinkron bank");
    expect(text).not.toContain("ribuan pengguna");
    expect(text).not.toContain("byok");
  });

  it("answers the API-key question with a clear 'tidak'", () => {
    const item = LANDING_FAQ.find((f) => /api key/i.test(f.q));
    expect(item).toBeDefined();
    expect(item?.a.startsWith("Tidak")).toBe(true);
  });
});

describe("PRICING_FAQ", () => {
  it("has eight answer-first items", () => {
    expect(PRICING_FAQ).toHaveLength(8);
    PRICING_FAQ.forEach((item) => {
      expect(item.q.trim().length).toBeGreaterThan(8);
      expect(item.a.trim().length).toBeGreaterThan(30);
    });
  });

  it("never advertises an unavailable feature or false claim", () => {
    const text = JSON.stringify(PRICING_FAQ).toLowerCase();
    ["sinkron bank", "bank_sync", "byok", "api key milik", "ribuan pengguna", "tak terbatas"].forEach((bad) => {
      expect(text).not.toContain(bad);
    });
  });

  it("keeps internal links real (no dead anchors)", () => {
    const hrefs = PRICING_FAQ.flatMap((i) => (i.links ?? []).map((l) => l.href));
    expect(hrefs.length).toBeGreaterThan(0);
    hrefs.forEach((href) => {
      expect(href.startsWith("/")).toBe(true);
      expect(href).not.toBe("#");
    });
  });
});
