import { describe, it, expect } from "vitest";
import { PRICING } from "./pricing";

/** Flatten every string in the copy tree so guards can scan the whole module. */
function allStrings(value: unknown, out: string[] = []): string[] {
  if (typeof value === "string") out.push(value);
  else if (Array.isArray(value)) value.forEach((v) => allStrings(v, out));
  else if (value && typeof value === "object") Object.values(value).forEach((v) => allStrings(v, out));
  return out;
}

describe("PRICING copy", () => {
  const strings = allStrings(PRICING);
  const text = strings.join("\n");

  it("names the page for the primary keyword", () => {
    expect(PRICING.hero.h1).toBe("Harga FinTrack");
    expect(PRICING.hero.subhead.length).toBeGreaterThan(0);
  });

  it("never advertises the unenforced bank sync feature", () => {
    expect(text).not.toMatch(/\bbank_sync\b/i);
    expect(text).not.toMatch(/\bbank\b/i);
  });

  it("never re-introduces the removed BYOK claim", () => {
    expect(text).not.toMatch(/api key|byok/i);
  });

  it("describes the real payment flow without over-promising", () => {
    expect(PRICING.payment.methods).toHaveLength(2);
    expect(PRICING.payment.taxNote).toMatch(/pajak/i);
  });

  it("keeps every CTA on-site", () => {
    expect(PRICING.cta.cta.href.startsWith("/")).toBe(true);
  });
});
