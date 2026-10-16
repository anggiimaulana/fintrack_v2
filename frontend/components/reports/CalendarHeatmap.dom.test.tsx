import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import * as React from "react";
import { CalendarHeatmap } from "./CalendarHeatmap";
import type { DailyPoint } from "@/hooks/useReportAnalytics";

const mockUseTransactions = vi.fn();
const mockUseAdminCategories = vi.fn();

vi.mock("@/hooks/useTransactions", () => ({
  useTransactions: (...args: unknown[]) => mockUseTransactions(...args),
}));

vi.mock("@/hooks/useAdminCategories", () => ({
  useAdminCategories: (...args: unknown[]) => mockUseAdminCategories(...args),
}));

describe("CalendarHeatmap", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseTransactions.mockReturnValue({
      data: {
        data: [
          {
            id: "tx-1",
            type: "expense",
            amount: 50000,
            category: "Makanan",
            description: "Makan siang",
            source: "manual",
          },
        ],
        total: 1,
      },
      isLoading: false,
    });
    mockUseAdminCategories.mockReturnValue({
      categories: [{ id: "c1", name: "Makanan" }],
      isLoading: false,
    });
  });

  it("renders full month even if daily data is partial", () => {
    // Only 3 days provided in October
    const partialDaily: DailyPoint[] = [
      { date: "2026-10-01", income: 100000, expense: 50000, savings: 0, net: 50000 },
      { date: "2026-10-02", income: 0, expense: 20000, savings: 0, net: -20000 },
      { date: "2026-10-03", income: 0, expense: 0, savings: 0, net: 0 },
    ];

    render(<CalendarHeatmap daily={partialDaily} />);

    // October has 31 days, so day 31 should be rendered
    expect(screen.getByRole("button", { name: /2026-10-31/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /2026-10-01/i })).toBeInTheDocument();
  });

  it("opens popup modal when a date cell is clicked", () => {
    const daily: DailyPoint[] = [
      { date: "2026-10-15", income: 500000, expense: 50000, savings: 100000, net: 450000 },
    ];

    render(<CalendarHeatmap daily={daily} />);

    const cell = screen.getByRole("button", { name: /2026-10-15/i });
    fireEvent.click(cell);

    expect(screen.getByText("Detail Harian")).toBeInTheDocument();
    expect(screen.getByText(/15 Oktober 2026/i)).toBeInTheDocument();
    expect(screen.getByText("Daftar Transaksi")).toBeInTheDocument();
    expect(screen.getByText("Makan siang")).toBeInTheDocument();
  });
});
