import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { CycleConfigMatrix } from "./CycleConfigMatrix"
import type { CatalogLimit } from "@/hooks/useEntitlementCatalog"
import type { PlanLimitCycle } from "@/hooks/usePlanCycles"

const mockLimits: CatalogLimit[] = [
  {
    key: "ai_tokens_per_cycle",
    label: "Kuota AI",
    unit: "kuota",
    description: "Kuota token AI per siklus",
    enforced: true,
    per_cycle: true,
  },
  {
    key: "max_transactions",
    label: "Transaksi",
    unit: "transaksi",
    description: "Transaksi aktif",
    enforced: true,
    per_cycle: false,
  },
]

describe("CycleConfigMatrix", () => {
  it("renders all catalog limits with monthly_only by default", () => {
    const onChange = vi.fn()
    render(<CycleConfigMatrix limits={mockLimits} cycles={[]} onChange={onChange} />)

    expect(screen.getByText("Kuota AI")).toBeInTheDocument()
    expect(screen.getByText("Transaksi")).toBeInTheDocument()
    expect(screen.getAllByText("Bulanan Saja").length).toBe(2)
  })

  it("renders weekly limit input and degraded form when weekly_monthly is active", () => {
    const cycles: PlanLimitCycle[] = [
      {
        limit_key: "ai_tokens_per_cycle",
        cycle_type: "weekly_monthly",
        weekly_limit: 50000,
        degraded_mode: "model_downgrade",
        fallback_model: "gpt-4o-mini",
      },
    ]

    const onChange = vi.fn()
    render(<CycleConfigMatrix limits={mockLimits} cycles={cycles} onChange={onChange} />)

    expect(screen.getByDisplayValue("50.000")).toBeInTheDocument()
    expect(screen.getByText("Perilaku Saat Mingguan Habis")).toBeInTheDocument()
    expect(screen.getByText("Model AI Fallback")).toBeInTheDocument()
  })

  it("updates weekly limit on change", () => {
    const cycles: PlanLimitCycle[] = [
      {
        limit_key: "ai_tokens_per_cycle",
        cycle_type: "weekly_monthly",
        weekly_limit: 50000,
        degraded_mode: "hard_block",
      },
    ]

    const onChange = vi.fn()
    render(<CycleConfigMatrix limits={mockLimits} cycles={cycles} onChange={onChange} />)

    const input = screen.getByDisplayValue("50.000")
    fireEvent.change(input, { target: { value: "60000" } })

    expect(onChange).toHaveBeenCalledWith([
      expect.objectContaining({
        limit_key: "ai_tokens_per_cycle",
        weekly_limit: 60000,
      }),
    ])
  })
})
