import { describe, it, expect } from "vitest";
import { LANDING, NAV, FOOTER_COLUMNS, MICROCOPY, PAGE_META, PRIMARY_CTA } from "./copy";

const ALL_COPY = JSON.stringify({ LANDING, NAV, FOOTER_COLUMNS, MICROCOPY, PAGE_META });

describe("copy voice & claims", () => {
  it("avoids unverifiable superlatives and social proof", () => {
    ["terbaik", "nomor 1", "tercepat", "ribuan pengguna", "dipercaya jutaan", "dijamin"].forEach((bad) => {
      expect(ALL_COPY.toLowerCase()).not.toContain(bad);
    });
  });

  it("never promises unavailable features", () => {
    ["sinkron bank", "sinkronisasi bank", "bank_sync", "byok", "api key milik"].forEach((bad) => {
      expect(ALL_COPY.toLowerCase()).not.toContain(bad);
    });
  });

  it("addresses the reader as 'Anda', never 'kamu'", () => {
    expect(ALL_COPY).not.toMatch(/\bkamu\b/i);
  });

  it("does not shout (no doubled exclamation marks)", () => {
    expect(ALL_COPY).not.toContain("!!");
  });
});

describe("landing copy structure", () => {
  it("puts the primary keyword in the hero H1", () => {
    expect(LANDING.hero.h1.toLowerCase()).toContain("catat keuangan otomatis");
  });

  it("keeps the highlighted word present in the H1", () => {
    expect(LANDING.hero.h1).toContain(LANDING.hero.h1HighlightWord);
  });

  it("has four factual trust points", () => {
    expect(LANDING.trustBar).toHaveLength(4);
  });

  it("uses only true stat numbers", () => {
    expect(LANDING.stats.map((s) => s.value)).toEqual([3, 5, 0]);
  });

  it("has three how-it-works steps", () => {
    expect(LANDING.howItWorks.steps).toHaveLength(3);
  });

  it("lists the five financial-health statuses", () => {
    expect(LANDING.spk.statuses).toEqual(["Sangat Baik", "Baik", "Cukup", "Boros", "Kritis"]);
  });

  it("has four personas and never uses a fake name", () => {
    expect(LANDING.personas.items).toHaveLength(4);
    ["Budi", "Siti", "Andi"].forEach((name) => expect(ALL_COPY).not.toContain(name));
  });

  it("keeps the ticker words", () => {
    expect(LANDING.ticker.words).toEqual(["CATAT", "PAHAMI", "KELOLA", "TENANG"]);
  });
});

describe("navigation & footer links", () => {
  it("links navigation to real routes", () => {
    expect(NAV.map((n) => n.href)).toEqual(["/features", "/#how-it-works", "/pricing", "/explore"]);
  });

  it("never uses an empty '#' link anywhere", () => {
    [...NAV, ...FOOTER_COLUMNS.flatMap((c) => c.links)].forEach((link) => {
      expect(link.href).not.toBe("#");
      expect(link.href.startsWith("/")).toBe(true);
    });
  });

  it("declares the primary CTA target", () => {
    expect(PRIMARY_CTA).toEqual({ label: "Mulai Gratis", href: "/register" });
  });
});

describe("PAGE_META", () => {
  const expectedRoutes = [
    "/",
    "/pricing",
    "/features",
    "/explore",
    "/about",
    "/contact",
    "/faq",
    "/privacy",
    "/terms",
  ];

  it("covers every public route", () => {
    expectedRoutes.forEach((route) => expect(PAGE_META[route]).toBeDefined());
  });

  it("keeps titles and descriptions within SEO bounds", () => {
    Object.values(PAGE_META).forEach((meta) => {
      expect(meta.title.length).toBeLessThanOrEqual(70);
      expect(meta.description.length).toBeGreaterThanOrEqual(80);
      expect(meta.description.length).toBeLessThanOrEqual(200);
    });
  });
});
