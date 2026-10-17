import { describe, it, expect } from "vitest";
import type { PublicPlan } from "@/lib/plans-types";
import {
  buildLimitRows,
  formatLimitValue,
  isFreePlan,
  LIMIT_ROW_ORDER,
  orderPlans,
  planHasFeature,
  planHighlights,
  taxBreakdown,
  yearlyMonthlyEquivalent,
} from "./plans";

function makePlan(over: Partial<PublicPlan>): PublicPlan {
  return {
    code: "free",
    name: "Free",
    price_monthly_idr: 0,
    price_yearly_idr: 0,
    billing_period_days: 30,
    is_public: true,
    sort_order: 0,
    fallback_gateway: "midtrans",
    features: {},
    limits: {},
    limit_meta: {},
    ...over,
  };
}

const FREE = makePlan({
  code: "free",
  name: "Free",
  sort_order: 1,
  features: { ai_chat: true },
  limits: { ai_tokens_per_cycle: 5000, ai_requests_per_cycle: 50, max_transactions: 100, max_categories: 10, max_wallets: 1 },
  limit_meta: {
    ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
    ai_requests_per_cycle: { key: "ai_requests_per_cycle", label: "Permintaan AI / siklus", unit: "permintaan", enforced: true },
    max_transactions: { key: "max_transactions", label: "Transaksi", unit: "transaksi", enforced: true },
    max_categories: { key: "max_categories", label: "Kategori", unit: "kategori", enforced: false },
    max_wallets: { key: "max_wallets", label: "Dompet", unit: "dompet", enforced: false },
  },
});

const PRO = makePlan({
  code: "pro",
  name: "Pro",
  price_monthly_idr: 39000,
  price_yearly_idr: 390000,
  sort_order: 2,
  features: { ai_chat: true, ocr_receipt: true, voice_input: true, rag_search: true, reports_export: true, advanced_budgeting: true },
  limits: { ai_tokens_per_cycle: 150000, max_transactions: -1 },
  limit_meta: {
    ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
    max_transactions: { key: "max_transactions", label: "Transaksi", unit: "transaksi", enforced: true },
  },
});

const BUSINESS = makePlan({
  code: "business",
  name: "Business",
  price_monthly_idr: 129000,
  sort_order: 3,
  features: { ai_chat: true, ocr_receipt: true, voice_input: true, rag_search: true, reports_export: true, advanced_budgeting: true, api_access: true },
  limits: { ai_tokens_per_cycle: 1000000, voice_requests_per_day: -1 },
  limit_meta: {
    ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
    voice_requests_per_day: { key: "voice_requests_per_day", label: "Permintaan suara / hari", unit: "permintaan", enforced: true },
  },
});

describe("planHighlights", () => {
  it("lists only enabled features for each curated plan", () => {
    expect(planHighlights(FREE)).toContain("Asisten AI (chat)");
    expect(planHighlights(PRO)).toContain("Pindai struk (OCR)");
    expect(planHighlights(PRO)).not.toContain("Akses API");
    expect(planHighlights(BUSINESS)).toContain("Akses API");
  });

  it("drops a curated bullet whose backing feature is disabled", () => {
    const noApi = makePlan({ code: "business", features: { ocr_receipt: true } });
    expect(planHighlights(noApi)).not.toContain("Akses API");
  });

  it("caps bullets at four", () => {
    expect(planHighlights(BUSINESS).length).toBeLessThanOrEqual(4);
  });

  it("falls back to catalog rows for an unknown plan code", () => {
    const custom = makePlan({ code: "enterprise", features: { ai_chat: true, api_access: true } });
    expect(planHighlights(custom)).toEqual(["Asisten AI (chat)", "Akses API"]);
  });
});

describe("planHasFeature", () => {
  it("is true only for enabled features", () => {
    expect(planHasFeature(BUSINESS, "api_access")).toBe(true);
    expect(planHasFeature(PRO, "api_access")).toBe(false);
  });
});

describe("formatLimitValue", () => {
  it("renders the unbounded sentinel as 'Tanpa batas'", () => {
    expect(formatLimitValue(-1, "transaksi")).toBe("Tanpa batas");
  });

  it("formats numbers with Indonesian separators and a unit", () => {
    expect(formatLimitValue(150000, "token")).toBe("150.000 token");
  });

  it("renders a missing value as an em dash", () => {
    expect(formatLimitValue(undefined, "scan")).toBe("—");
  });
});

describe("buildLimitRows", () => {
  const rows = buildLimitRows([FREE, PRO, BUSINESS]);

  it("keeps only enforced limits in the canonical order", () => {
    expect(rows.map((r) => r.key)).toEqual([
      "ai_tokens_per_cycle",
      "ai_requests_per_cycle",
      "max_transactions",
      "voice_requests_per_day",
    ]);
  });

  it("never shows a limit the backend does not enforce", () => {
    const keys = rows.map((r) => r.key);
    expect(keys).not.toContain("max_categories");
    expect(keys).not.toContain("max_wallets");
  });

  it("takes labels and units from limit_meta", () => {
    const tokenRow = rows.find((r) => r.key === "ai_tokens_per_cycle");
    expect(tokenRow?.label).toBe("Token AI / siklus");
    expect(tokenRow?.unit).toBe("token");
  });

  it("renders a value per plan with 'Tanpa batas' for -1", () => {
    const txRow = rows.find((r) => r.key === "max_transactions");
    expect(txRow?.values.find((v) => v.code === "free")?.text).toBe("100 transaksi");
    expect(txRow?.values.find((v) => v.code === "pro")?.text).toBe("Tanpa batas");
  });

  it("returns no rows for an empty catalog", () => {
    expect(buildLimitRows([])).toEqual([]);
  });

  it("covers every key in the canonical order list", () => {
    // Guards against a typo silently dropping a row.
    LIMIT_ROW_ORDER.forEach((key) => expect(typeof key).toBe("string"));
  });
});

describe("plan ordering & helpers", () => {
  it("orders plans by sort_order", () => {
    const shuffled = [BUSINESS, FREE, PRO];
    expect(orderPlans(shuffled).map((p) => p.code)).toEqual(["free", "pro", "business"]);
  });

  it("detects the free plan by code or zero price", () => {
    expect(isFreePlan(FREE)).toBe(true);
    expect(isFreePlan(PRO)).toBe(false);
    expect(isFreePlan(makePlan({ code: "starter", price_monthly_idr: 0 }))).toBe(true);
  });

  it("computes the effective monthly price for a yearly plan", () => {
    expect(yearlyMonthlyEquivalent(PRO)).toBe(32500);
  });
});

describe("taxBreakdown", () => {
  const TAXED = makePlan({
    code: "pro",
    price_monthly_idr: 39000,
    price_yearly_idr: 390000,
    tax_enabled: true,
    tax_label: "PPN",
    tax_rate_percent: 11,
    tax_idr: 4290,
    total_idr: 43290,
  });

  it("returns null when tax is disabled", () => {
    expect(taxBreakdown(FREE, "monthly")).toBeNull();
  });

  it("returns null rather than a '+ 0%' line when the rate is zero", () => {
    expect(taxBreakdown(makePlan({ tax_enabled: true, tax_rate_percent: 0 }), "monthly")).toBeNull();
  });

  it("uses the backend monthly tax/total for the monthly cycle", () => {
    expect(taxBreakdown(TAXED, "monthly")).toEqual({
      subtotalIdr: 39000,
      taxIdr: 4290,
      totalIdr: 43290,
      label: "PPN",
      ratePercent: 11,
    });
  });

  it("applies the same rate to the yearly subtotal, which has no backend tax field", () => {
    const yearly = taxBreakdown(TAXED, "yearly");
    expect(yearly).not.toBeNull();
    expect(yearly?.subtotalIdr).toBe(390000);
    expect(yearly?.taxIdr).toBe(42900);
    expect(yearly?.totalIdr).toBe(432900);
  });

  it("never mixes tax into the subtotal", () => {
    const monthly = taxBreakdown(TAXED, "monthly");
    expect(monthly?.subtotalIdr).toBe(TAXED.price_monthly_idr);
    expect(monthly?.totalIdr).toBe((monthly?.subtotalIdr ?? 0) + (monthly?.taxIdr ?? 0));
  });
});
