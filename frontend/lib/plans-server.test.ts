import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { fetchPublicPlans, enforcedLimits } from "./plans-server";
import type { PublicPlan } from "./plans-types";

/** Build a minimal fetch Response-like object. */
function jsonResponse(status: number, payload: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => payload,
  } as unknown as Response;
}

function makePlan(overrides: Partial<PublicPlan> = {}): PublicPlan {
  return {
    code: "free",
    name: "Free",
    price_monthly_idr: 0,
    price_yearly_idr: 0,
    billing_period_days: 30,
    is_public: true,
    sort_order: 1,
    fallback_gateway: "manual",
    features: {},
    limits: {},
    ...overrides,
  };
}

describe("fetchPublicPlans", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns the parsed array on a 200 response", async () => {
    const plans = [makePlan({ code: "free" }), makePlan({ code: "pro" })];
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, { success: true, data: plans }));

    await expect(fetchPublicPlans()).resolves.toEqual(plans);
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("returns [] on a non-OK response", async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse(500, { success: false }));

    await expect(fetchPublicPlans()).resolves.toEqual([]);
  });

  it("returns [] when data is not an array", async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, { data: { nope: true } }));

    await expect(fetchPublicPlans()).resolves.toEqual([]);
  });

  it("returns [] when the network call throws", async () => {
    vi.mocked(fetch).mockRejectedValue(new Error("network down"));

    await expect(fetchPublicPlans()).resolves.toEqual([]);
  });
});

describe("enforcedLimits", () => {
  it("hides max_categories and max_wallets (not enforced)", () => {
    const plan = makePlan({
      limits: {
        ai_tokens_per_cycle: 5000,
        max_transactions: 100,
        max_categories: 30,
        max_wallets: 5,
      },
      limit_meta: {
        ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
        max_transactions: { key: "max_transactions", label: "Transaksi", unit: "transaksi", enforced: true },
        max_categories: { key: "max_categories", label: "Kategori", unit: "kategori", enforced: false },
        max_wallets: { key: "max_wallets", label: "Dompet", unit: "dompet", enforced: false },
      },
    });

    const keys = enforcedLimits(plan).map(([key]) => key);
    expect(keys).toEqual(["ai_tokens_per_cycle", "max_transactions"]);
    expect(keys).not.toContain("max_categories");
    expect(keys).not.toContain("max_wallets");
  });

  it("treats a limit missing from limit_meta as enforced (backwards compatible)", () => {
    const plan = makePlan({
      limits: { max_transactions: 100 },
      limit_meta: {},
    });

    expect(enforcedLimits(plan)).toEqual([["max_transactions", 100]]);
  });
});
