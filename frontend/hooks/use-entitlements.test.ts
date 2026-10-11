import { describe, it, expect } from "vitest";
import {
  EMPTY_ENTITLEMENTS,
  hasFeature,
  withinLimit,
  limitValue,
  type Entitlements,
} from "./use-entitlements";

function ent(partial: Partial<Entitlements>): Entitlements {
  return { ...EMPTY_ENTITLEMENTS, ...partial };
}

describe("hasFeature", () => {
  it("returns true only when the feature is explicitly enabled", () => {
    const e = ent({ features: { ai_chat: true, ocr_receipt: false } });
    expect(hasFeature(e, "ai_chat")).toBe(true);
    expect(hasFeature(e, "ocr_receipt")).toBe(false);
  });

  it("fails closed for an unknown feature", () => {
    expect(hasFeature(ent({ features: { ai_chat: true } }), "nope")).toBe(false);
  });

  it("fails closed when entitlements are absent", () => {
    expect(hasFeature(null, "ai_chat")).toBe(false);
    expect(hasFeature(ent({}), "ai_chat")).toBe(false);
  });
});

describe("withinLimit", () => {
  it("allows usage below the limit", () => {
    const e = ent({ limits: { max_transactions: 100 } });
    expect(withinLimit(e, "max_transactions", 99)).toBe(true);
  });

  it("blocks usage at the limit", () => {
    const e = ent({ limits: { max_transactions: 100 } });
    expect(withinLimit(e, "max_transactions", 100)).toBe(false);
  });

  it("treats -1 as unlimited", () => {
    const e = ent({ limits: { max_transactions: -1 } });
    expect(withinLimit(e, "max_transactions", 1_000_000)).toBe(true);
  });

  it("fails closed for an absent limit key", () => {
    expect(withinLimit(ent({ limits: {} }), "max_transactions", 0)).toBe(false);
    expect(withinLimit(null, "max_transactions", 0)).toBe(false);
  });
});

describe("limitValue", () => {
  it("returns the raw limit including the unlimited sentinel", () => {
    const e = ent({ limits: { ai_tokens_per_cycle: 5000, max_transactions: -1 } });
    expect(limitValue(e, "ai_tokens_per_cycle")).toBe(5000);
    expect(limitValue(e, "max_transactions")).toBe(-1);
  });

  it("returns undefined for an absent key", () => {
    expect(limitValue(ent({ limits: {} }), "nope")).toBeUndefined();
    expect(limitValue(null, "nope")).toBeUndefined();
  });
});
