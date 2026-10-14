import { describe, it, expect } from "vitest";
import { enforcedLimitEntries, limitLabel } from "./plan-limits";
import type { PublicPlan } from "@/hooks/use-billing";

function plan(partial: Partial<PublicPlan>): PublicPlan {
  return {
    code: "pro",
    name: "Pro",
    price_monthly_idr: 39000,
    price_yearly_idr: 0,
    billing_period_days: 30,
    is_public: true,
    sort_order: 2,
    fallback_gateway: "manual",
    features: {},
    limits: {},
    ...partial,
  };
}

describe("enforcedLimitEntries", () => {
  it("drops limits the backend marks as not enforced", () => {
    // Regression for F4: max_categories/max_wallets are not enforced (categories
    // are admin-managed, no per-user wallet entity), so the pricing UI must not
    // advertise them as plan benefits.
    const p = plan({
      limits: {
        ai_tokens_per_cycle: 150000,
        max_transactions: 5000,
        ocr_scans_per_cycle: 100,
        max_categories: 30,
        max_wallets: 5,
      },
      limit_meta: {
        ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
        max_transactions: { key: "max_transactions", label: "Transaksi", unit: "transaksi", enforced: true },
        ocr_scans_per_cycle: { key: "ocr_scans_per_cycle", label: "Pindai struk / siklus", unit: "scan", enforced: true },
        max_categories: { key: "max_categories", label: "Kategori", unit: "kategori", enforced: false },
        max_wallets: { key: "max_wallets", label: "Dompet", unit: "dompet", enforced: false },
      },
    });

    const keys = enforcedLimitEntries(p).map(([k]) => k);
    expect(keys).toEqual(["ai_tokens_per_cycle", "max_transactions", "ocr_scans_per_cycle"]);
    expect(keys).not.toContain("max_categories");
    expect(keys).not.toContain("max_wallets");
  });

  it("treats a limit with no metadata as enforced (backwards compatible)", () => {
    const p = plan({ limits: { max_transactions: 100 } });
    expect(enforcedLimitEntries(p).map(([k]) => k)).toEqual(["max_transactions"]);
  });

  it("keeps an explicitly enforced limit", () => {
    const p = plan({
      limits: { ai_tokens_per_cycle: 5000 },
      limit_meta: {
        ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
      },
    });
    expect(enforcedLimitEntries(p)).toEqual([["ai_tokens_per_cycle", 5000]]);
  });

  it("sorts keys deterministically", () => {
    const p = plan({ limits: { ocr_scans_per_cycle: 1, ai_tokens_per_cycle: 2, max_transactions: 3 } });
    expect(enforcedLimitEntries(p).map(([k]) => k)).toEqual([
      "ai_tokens_per_cycle",
      "max_transactions",
      "ocr_scans_per_cycle",
    ]);
  });

  it("returns an empty list when every limit is unenforced", () => {
    const p = plan({
      limits: { max_categories: 5, max_wallets: 1 },
      limit_meta: {
        max_categories: { key: "max_categories", label: "Kategori", unit: "kategori", enforced: false },
        max_wallets: { key: "max_wallets", label: "Dompet", unit: "dompet", enforced: false },
      },
    });
    expect(enforcedLimitEntries(p)).toEqual([]);
  });
});

describe("limitLabel", () => {
  it("prefers the server-provided label", () => {
    const p = plan({
      limits: { max_transactions: 100 },
      limit_meta: {
        max_transactions: { key: "max_transactions", label: "Transaksi (server)", unit: "transaksi", enforced: true },
      },
    });
    expect(limitLabel(p, "max_transactions", { max_transactions: "fallback" })).toBe("Transaksi (server)");
  });

  it("falls back to the provided label map, then the raw key", () => {
    const p = plan({ limits: { max_transactions: 100, unknown_key: 1 } });
    expect(limitLabel(p, "max_transactions", { max_transactions: "Transaksi" })).toBe("Transaksi");
    expect(limitLabel(p, "unknown_key")).toBe("unknown_key");
  });
});
