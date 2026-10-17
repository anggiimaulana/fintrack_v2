import { describe, it, expect } from "vitest";
import {
  FEATURES,
  FEATURE_SLUGS,
  getFeature,
  featureHref,
  featureWords,
  featureWordCount,
  featureUniqueWordCount,
  PLAN_LABEL,
} from "./features";

const EXPECTED_SLUGS = [
  "ai-chat",
  "voice-input",
  "receipt-scan",
  "dashboard",
  "financial-health",
  "reports",
];

describe("FEATURES", () => {
  it("ships exactly six features", () => {
    expect(FEATURES).toHaveLength(6);
  });

  it("uses the agreed English slugs", () => {
    expect(FEATURE_SLUGS).toEqual(EXPECTED_SLUGS);
  });

  it("has unique, kebab-case slugs", () => {
    const slugs = FEATURES.map((f) => f.slug);
    expect(new Set(slugs).size).toBe(slugs.length);
    slugs.forEach((slug) => expect(slug).toMatch(/^[a-z0-9]+(-[a-z0-9]+)*$/));
  });

  it("gives every feature the required content", () => {
    FEATURES.forEach((f) => {
      expect(f.title.length).toBeGreaterThan(0);
      expect(f.navLabel.length).toBeGreaterThan(0);
      expect(f.tagline.length).toBeGreaterThan(0);
      expect(f.heroDescription.length).toBeGreaterThan(40);
      expect(f.problem.length).toBeGreaterThan(40);
      expect(f.detail.length).toBeGreaterThanOrEqual(3);
      expect(f.howItWorks.length).toBeGreaterThanOrEqual(3);
      expect(f.benefits.length).toBeGreaterThanOrEqual(3);
      expect(f.useCases.length).toBeGreaterThanOrEqual(3);
      expect(f.notes.length).toBeGreaterThanOrEqual(3);
      expect(f.keywords.length).toBeGreaterThanOrEqual(2);
      expect(f.example.prompt.length).toBeGreaterThan(0);
      expect(f.example.result.length).toBeGreaterThan(0);
      expect(f.enabledInPlans.length).toBeGreaterThan(0);
    });
  });

  it("keeps each sub-page long enough to avoid thin content (spec doc 04 §B.1)", () => {
    FEATURES.forEach((f) => {
      expect(featureWordCount(f), `${f.slug} total words`).toBeGreaterThanOrEqual(300);
      expect(featureUniqueWordCount(f), `${f.slug} distinct words`).toBeGreaterThanOrEqual(200);
    });
  });

  it("keeps every sub-page materially distinct (no near-duplicate copy)", () => {
    const sets = FEATURES.map((f) => new Set(featureWords(f)));
    for (let i = 0; i < sets.length; i += 1) {
      for (let j = i + 1; j < sets.length; j += 1) {
        const intersection = [...sets[i]].filter((word) => sets[j].has(word)).length;
        const union = sets[i].size + sets[j].size - intersection;
        // Shared function words are expected; whole sentences are not.
        expect(intersection / union, `${FEATURES[i].slug} vs ${FEATURES[j].slug}`).toBeLessThan(0.5);
      }
    }
  });

  it("gives every feature an SEO meta description with a target keyword", () => {
    FEATURES.forEach((f) => {
      expect(f.metaDescription.length, `${f.slug} meta length`).toBeGreaterThanOrEqual(70);
      expect(f.metaDescription.length).toBeLessThanOrEqual(180);
      const lower = f.metaDescription.toLowerCase();
      expect(f.keywords.some((keyword) => lower.includes(keyword.toLowerCase()))).toBe(true);
    });
  });

  it("points every relatedSlug at a real feature", () => {
    FEATURES.forEach((f) => {
      expect(f.relatedSlugs.length).toBeGreaterThanOrEqual(2);
      f.relatedSlugs.forEach((slug) => {
        expect(FEATURE_SLUGS).toContain(slug);
        expect(slug).not.toBe(f.slug);
      });
    });
  });

  it("matches the backend feature gating (receipt-scan and reports are paid)", () => {
    expect(getFeature("receipt-scan")?.enabledInPlans).toEqual(["pro", "business"]);
    expect(getFeature("voice-input")?.enabledInPlans).toEqual(["pro", "business"]);
    expect(getFeature("reports")?.enabledInPlans).toEqual(["pro", "business"]);
    expect(getFeature("ai-chat")?.enabledInPlans).toEqual(["free", "pro", "business"]);
  });

  it("never advertises the unavailable bank_sync feature", () => {
    const haystack = JSON.stringify(FEATURES).toLowerCase();
    expect(haystack).not.toContain("bank_sync");
    expect(haystack).not.toContain("sinkronisasi bank");
  });
});

describe("feature helpers", () => {
  it("looks up a feature by slug", () => {
    expect(getFeature("ai-chat")?.title).toBe("Catat & Tanya Lewat Chat");
    expect(getFeature("nope")).toBeUndefined();
  });

  it("builds canonical hrefs", () => {
    expect(featureHref("receipt-scan")).toBe("/features/receipt-scan");
  });

  it("labels plans", () => {
    expect(PLAN_LABEL.pro).toBe("Pro");
  });
});
