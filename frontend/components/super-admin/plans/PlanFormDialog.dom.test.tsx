import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { PlanFormDialog } from "./PlanFormDialog"
import type { PlanDTO } from "@/hooks/useSuperAdminPlans"
import type { PlanLimitCycle } from "@/hooks/usePlanCycles"

const EMPTY_CYCLES: PlanLimitCycle[] = []
vi.mock("@/hooks/useEntitlementCatalog", () => ({
  useEntitlementCatalog: () => ({
    catalog: {
      features: [
        { key: "ai_chat", label: "Asisten AI (Chat)", description: "Chat AI" },
        { key: "ocr_receipt", label: "Pindai Struk (OCR)", description: "OCR" },
      ],
      limits: [
        { key: "ai_tokens_per_cycle", label: "Token AI per Siklus", unit: "token", enforced: true },
        { key: "max_transactions", label: "Maksimal Transaksi", unit: "transaksi", enforced: true },
      ],
    },
    isLoading: false,
  }),
}))

vi.mock("@/hooks/usePlanCycles", () => ({
  usePlanCycles: () => ({
    cycles: EMPTY_CYCLES,
    saveCycles: vi.fn(),
  }),
}))

vi.mock("./CycleConfigMatrix", () => ({
  CycleConfigMatrix: () => <div data-testid="cycle-config-matrix" />,
}))

const mockPlan: PlanDTO = {
  id: "plan-123",
  code: "pro",
  name: "Pro Plan",
  description: "Paket untuk pengguna aktif",
  price_monthly_idr: 50000,
  price_yearly_idr: 500000,
  billing_period_days: 30,
  is_active: true,
  is_public: true,
  fallback_gateway: "midtrans",
  sort_order: 1,
  token_quota_ratio: 5,
  entitlements: {
    ai_chat: true,
    ocr_receipt: false,
  },
  limits: {
    ai_tokens_per_cycle: 10000,
    max_transactions: -1,
  },
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
}

describe("PlanFormDialog", () => {
  it("renders textarea for description and clean labels for settings", () => {
    render(
      <PlanFormDialog
        open={true}
        onOpenChange={vi.fn()}
        plan={mockPlan}
        onSubmit={vi.fn()}
      />
    )

    // Textarea for description
    const descTextarea = screen.getByLabelText(/Deskripsi Paket/i)
    expect(descTextarea.tagName.toLowerCase()).toBe("textarea")
    expect(descTextarea).toHaveValue("Paket untuk pengguna aktif")

    // Clean labels (no snake_case like sort_order or is_active)
    expect(screen.getByText("Urutan Tampilan")).toBeInTheDocument()
    expect(screen.getByText("Rasio Kuota Token")).toBeInTheDocument()
    expect(screen.getByText("Status Aktif")).toBeInTheDocument()
    expect(screen.getByText("Tampilkan ke Publik")).toBeInTheDocument()
    expect(screen.queryByText(/is_active/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/is_public/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/sort_order/i)).not.toBeInTheDocument()
  })

  it("renders ready-to-use checklist switches for entitlements and clean limits", () => {
    render(
      <PlanFormDialog
        open={true}
        onOpenChange={vi.fn()}
        plan={mockPlan}
        onSubmit={vi.fn()}
      />
    )

    // Ready-to-use checklist for features
    expect(screen.getByLabelText(/Aktifkan fitur Asisten AI/i)).toBeChecked()
    expect(screen.getByLabelText(/Aktifkan fitur Pindai Struk/i)).not.toBeChecked()

    // Configured limits rendered with clean Indonesian labels
    expect(screen.getByText("Token AI per Siklus")).toBeInTheDocument()
    expect(screen.getByText("Maksimal Transaksi")).toBeInTheDocument()
    expect(screen.getByText("Unlimited")).toBeInTheDocument()
  })

  it("submits the updated payload when save button is clicked", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(
      <PlanFormDialog
        open={true}
        onOpenChange={vi.fn()}
        plan={mockPlan}
        onSubmit={onSubmit}
      />
    )

    const submitBtn = screen.getByRole("button", { name: /Simpan Perubahan/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledTimes(1)
    })

    const payload = onSubmit.mock.calls[0][0]
    expect(payload.code).toBe("pro")
    expect(payload.name).toBe("Pro Plan")
    expect(payload.description).toBe("Paket untuk pengguna aktif")
    expect(payload.fallback_gateway).toBe("midtrans")
    expect(payload.sort_order).toBe(1)
    expect(payload.token_quota_ratio).toBe(5)
    expect(payload.entitlements.ai_chat).toBe(true)
    expect(payload.limits.ai_tokens_per_cycle).toBe(10000)
    expect(payload.limits.max_transactions).toBe(-1)
  })
})
