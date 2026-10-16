import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { SmartInsightCard } from "./SmartInsightCard"

describe("SmartInsightCard", () => {
  it("renders summary text and delta badge", () => {
    render(
      <SmartInsightCard
        summaryText="Konsumsi token meningkat 18.4% minggu ini."
        deltaPercent={18.4}
        deltaDirection="up"
        isPositiveDelta={true}
      />
    )

    expect(screen.getByText("Konsumsi token meningkat 18.4% minggu ini.")).toBeInTheDocument()
    expect(screen.getByText("+18.4%")).toBeInTheDocument()
  })

  it("renders metric summary items", () => {
    render(
      <SmartInsightCard
        summaryText="Ringkasan metrik kuota."
        metricsSummary={[
          { label: "Total Token", value: "150.000" },
          { label: "Biaya IDR", value: "Rp 250.000" },
        ]}
      />
    )

    expect(screen.getByText("Total Token")).toBeInTheDocument()
    expect(screen.getByText("150.000")).toBeInTheDocument()
    expect(screen.getByText("Rp 250.000")).toBeInTheDocument()
  })

  it("renders recommendation text and link", () => {
    render(
      <SmartInsightCard
        summaryText="Ringkasan"
        actionRecommendation={{
          text: "Periksa bukti transfer.",
          actionLabel: "Review Pembayaran",
          href: "/super-admin/payments/review",
        }}
      />
    )

    expect(screen.getByText(/Periksa bukti transfer/i)).toBeInTheDocument()
    const link = screen.getByRole("link", { name: /Review Pembayaran/i })
    expect(link).toHaveAttribute("href", "/super-admin/payments/review")
  })
})
