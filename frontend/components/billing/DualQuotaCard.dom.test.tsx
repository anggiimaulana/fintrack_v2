import { render, screen } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { DualQuotaCard } from "./DualQuotaCard"
import * as entitlementsHook from "@/hooks/use-entitlements"
import * as quotaHook from "@/hooks/use-quota"
import type { UseEntitlementsResult } from "@/hooks/use-entitlements"
import type { UserQuotaSummary } from "@/hooks/use-quota"
import type { UseQueryResult } from "@tanstack/react-query"

vi.mock("@/hooks/use-entitlements")
vi.mock("@/hooks/use-quota")

describe("DualQuotaCard", () => {
  it("renders dual progress bars for weekly + monthly metric", () => {
    vi.mocked(entitlementsHook.useEntitlements).mockReturnValue({
      entitlements: {
        user_id: "1",
        plan_code: "pro",
        plan_name: "Pro Plan",
        status: "active",
        version: 1,
        features: {},
        limits: {
          ai_tokens_per_cycle: 150000,
        },
        usage: {
          ai_tokens_per_cycle: {
            used: 45000,
            limit: 150000,
            remaining: 105000,
            unlimited: false,
            unit: "kuota",
          },
        },
        period: {
          start: new Date().toISOString(),
          end: new Date(Date.now() + 15 * 24 * 3600_000).toISOString(),
        },
      },
      isLoading: false,
      isError: false,
    } as unknown as UseEntitlementsResult)

    vi.mocked(quotaHook.useQuota).mockReturnValue({
      data: {
        warning_threshold: 80,
        weekly_anchor: new Date().toISOString(),
        counters: [
          {
            id: "1",
            user_id: "1",
            limit_key: "ai_tokens_per_cycle",
            cycle_type: "weekly",
            cycle_start: new Date().toISOString(),
            cycle_end: new Date(Date.now() + 3 * 24 * 3600_000).toISOString(),
            used: 15000,
            degraded_used: 0,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        ],
        cycles: {
          ai_tokens_per_cycle: {
            limit_key: "ai_tokens_per_cycle",
            cycle_type: "weekly_monthly",
            weekly_limit: 50000,
            degraded_mode: "model_downgrade",
            fallback_model: "gpt-4o-mini",
          },
        },
      },
      isLoading: false,
    } as unknown as UseQueryResult<UserQuotaSummary, Error>)

    render(<DualQuotaCard />)

    expect(screen.getByText("Penggunaan Kuota Anda")).toBeInTheDocument()
    expect(screen.getByText("Kuota AI")).toBeInTheDocument()
    expect(screen.getByText("Siklus Bulanan")).toBeInTheDocument()
    expect(screen.getByText("Siklus Mingguan")).toBeInTheDocument()
    expect(screen.getByText("Normal")).toBeInTheDocument()
  })

  it("renders status badge as Dibatasi when weekly exhausted and throttle mode configured", () => {
    vi.mocked(entitlementsHook.useEntitlements).mockReturnValue({
      entitlements: {
        user_id: "1",
        plan_code: "pro",
        plan_name: "Pro Plan",
        status: "active",
        version: 1,
        features: {},
        limits: {
          ai_requests_per_cycle: 1000,
        },
        usage: {
          ai_requests_per_cycle: {
            used: 300,
            limit: 1000,
            remaining: 700,
            unlimited: false,
            unit: "permintaan",
          },
        },
      },
      isLoading: false,
      isError: false,
    } as unknown as UseEntitlementsResult)

    vi.mocked(quotaHook.useQuota).mockReturnValue({
      data: {
        warning_threshold: 80,
        counters: [
          {
            id: "1",
            user_id: "1",
            limit_key: "ai_requests_per_cycle",
            cycle_type: "weekly",
            cycle_start: new Date().toISOString(),
            cycle_end: new Date(Date.now() + 2 * 24 * 3600_000).toISOString(),
            used: 200, // Exhausted weekly limit
            degraded_used: 10,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        ],
        cycles: {
          ai_requests_per_cycle: {
            limit_key: "ai_requests_per_cycle",
            cycle_type: "weekly_monthly",
            weekly_limit: 200,
            degraded_mode: "throttle",
          },
        },
      },
      isLoading: false,
    } as unknown as UseQueryResult<UserQuotaSummary, Error>)

    render(<DualQuotaCard />)

    expect(screen.getByText("Dibatasi")).toBeInTheDocument()
  })
  it("inherits monthly usage for weekly cycle in week 1 when weekly counter is zero", () => {
    vi.mocked(entitlementsHook.useEntitlements).mockReturnValue({
      entitlements: {
        user_id: "1",
        plan_code: "pro",
        plan_name: "Pro Plan",
        status: "active",
        version: 1,
        features: {},
        limits: {
          ai_tokens_per_cycle: 500000,
        },
        usage: {
          ai_tokens_per_cycle: {
            used: 59394,
            limit: 500000,
            remaining: 440606,
            unlimited: false,
            unit: "kuota",
          },
        },
        period: {
          start: new Date(Date.now() - 2 * 24 * 3600_000).toISOString(), // 2 days ago
          end: new Date(Date.now() + 28 * 24 * 3600_000).toISOString(),
        },
      },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    } as unknown as UseEntitlementsResult)

    vi.mocked(quotaHook.useQuota).mockReturnValue({
      data: {
        warning_threshold: 80,
        weekly_anchor: new Date(Date.now() - 2 * 24 * 3600_000).toISOString(),
        counters: [
          {
            id: "1",
            user_id: "1",
            limit_key: "ai_tokens_per_cycle",
            cycle_type: "weekly",
            cycle_start: new Date(Date.now() - 2 * 24 * 3600_000).toISOString(),
            cycle_end: new Date(Date.now() + 5 * 24 * 3600_000).toISOString(),
            used: 0,
            degraded_used: 0,
            created_at: "",
            updated_at: "",
          },
        ],
        cycles: {
          ai_tokens_per_cycle: {
            limit_key: "ai_tokens_per_cycle",
            cycle_type: "weekly_monthly",
            weekly_limit: 100000,
            degraded_mode: "model_downgrade",
          },
        },
      },
      isLoading: false,
    } as unknown as UseQueryResult<UserQuotaSummary, Error>)

    render(<DualQuotaCard />)

    // Weekly progress text displays 59.394 instead of 0
    expect(screen.getByText(/59\.394 \/ 100\.000/)).toBeInTheDocument()
  })

  it("does not render unenforced/untracked limits such as max_categories or max_wallets", () => {
    vi.mocked(entitlementsHook.useEntitlements).mockReturnValue({
      entitlements: {
        user_id: "1",
        plan_code: "pro",
        plan_name: "Pro Plan",
        status: "active",
        version: 1,
        features: {},
        limits: {
          ai_tokens_per_cycle: 150000,
          max_categories: 30,
          max_wallets: 5,
        },
        usage: {
          ai_tokens_per_cycle: {
            used: 45000,
            limit: 150000,
            remaining: 105000,
            unlimited: false,
            unit: "kuota",
          },
        },
        period: {
          start: new Date().toISOString(),
          end: new Date(Date.now() + 15 * 24 * 3600_000).toISOString(),
        },
      },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    } as unknown as UseEntitlementsResult)

    vi.mocked(quotaHook.useQuota).mockReturnValue({
      data: {
        warning_threshold: 80,
        counters: [],
        cycles: {},
      },
      isLoading: false,
    } as unknown as UseQueryResult<UserQuotaSummary, Error>)

    render(<DualQuotaCard />)

    expect(screen.getByText("Kuota AI")).toBeInTheDocument()
    expect(screen.queryByText("max_categories")).not.toBeInTheDocument()
    expect(screen.queryByText("max_wallets")).not.toBeInTheDocument()
  })
})

