import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { QuotaCountdown, formatQuotaRemaining } from "./QuotaCountdown"

describe("QuotaCountdown", () => {
  it("formats remaining milliseconds correctly", () => {
    // 2 days 3 hours
    expect(formatQuotaRemaining((2 * 24 + 3) * 3600_000)).toBe("2 hari 3 jam")
    // 4 hours 15 minutes
    expect(formatQuotaRemaining((4 * 60 + 15) * 60_000)).toBe("4 jam 15 menit")
    // 25 minutes
    expect(formatQuotaRemaining(25 * 60_000)).toBe("25 menit")
    // <= 0
    expect(formatQuotaRemaining(-1000)).toBe("mereset...")
  })

  it("renders both weekly and monthly countdowns", () => {
    const weekly = new Date(Date.now() + 3 * 24 * 3600_000).toISOString()
    const monthly = new Date(Date.now() + 15 * 24 * 3600_000).toISOString()

    render(<QuotaCountdown weeklyResetAt={weekly} monthlyResetAt={monthly} />)

    expect(screen.getByText(/Reset mingguan:/i)).toBeInTheDocument()
    expect(screen.getByText(/Reset bulanan:/i)).toBeInTheDocument()
  })
})
