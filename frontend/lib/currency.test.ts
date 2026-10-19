import { describe, it, expect, afterEach } from "vitest";
import {
  BASE_CURRENCY,
  formatMoney,
  formatOriginal,
  fromBase,
  toBase,
  setActiveCurrency,
  type CurrencyMeta,
} from "./currency";

const JPY: CurrencyMeta = {
  code: "JPY",
  name: "Yen Jepang",
  symbol: "¥",
  locale: "ja-JP",
  decimal_digits: 0,
  is_active: true,
  is_base: false,
  sort_order: 20,
  rate_source: "manual",
  rate_to_base: 110,
};

// Strip locale grouping spaces so assertions are stable across ICU versions.
const norm = (s: string) => s.replace(/\u00a0/g, "").replace(/\s/g, "");

afterEach(() => setActiveCurrency(BASE_CURRENCY));

describe("currency conversion", () => {
  it("keeps base IDR unchanged", () => {
    setActiveCurrency(BASE_CURRENCY);
    expect(norm(formatMoney(1_000))).toBe("Rp1.000");
    expect(fromBase(500)).toBe(500);
    expect(toBase(500)).toBe(500);
  });

  it("converts base IDR into the display currency", () => {
    // 110 IDR == 1 JPY (rate_to_base = 110).
    setActiveCurrency(JPY);
    expect(fromBase(1100)).toBe(10);
    expect(norm(formatMoney(1100))).toBe("¥10");
    // Round-trip: a display value returns to the same base.
    expect(toBase(10)).toBe(1100);
  });

  it("formats (does not convert) an already-display value via formatOriginal", () => {
    setActiveCurrency(BASE_CURRENCY);
    expect(norm(formatOriginal(1500, "JPY"))).toBe("¥1.500");
    expect(norm(formatOriginal(1500, "IDR"))).toBe("Rp1.500");
  });

  it("falls back to base when the active currency has no rate", () => {
    setActiveCurrency({ ...JPY, rate_to_base: null });
    expect(fromBase(500)).toBe(500);
  });
});
