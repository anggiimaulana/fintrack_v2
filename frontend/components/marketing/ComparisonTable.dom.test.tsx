import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ComparisonTable } from "./ComparisonTable";
import { PRICING } from "@/lib/marketing/pricing";
import { PLAN_FEATURE_ROWS, buildLimitRows } from "@/lib/marketing/plans";
import { SAMPLE_PLANS } from "@/test/fixtures/public-plans";

describe("ComparisonTable", () => {
  it("renders a feature row for every catalog feature", () => {
    render(<ComparisonTable plans={SAMPLE_PLANS} />);
    PLAN_FEATURE_ROWS.forEach((row) =>
      expect(screen.getByRole("rowheader", { name: row.label })).toBeInTheDocument(),
    );
  });

  it("deep-links the feature rows that have a page", () => {
    render(<ComparisonTable plans={SAMPLE_PLANS} />);
    expect(screen.getByRole("link", { name: "Asisten AI (chat)" })).toHaveAttribute(
      "href",
      "/features/ai-chat",
    );
    expect(screen.getByRole("link", { name: "Pindai struk (OCR)" })).toHaveAttribute(
      "href",
      "/features/receipt-scan",
    );
  });

  it("marks availability with descriptive accessible names", () => {
    render(<ComparisonTable plans={SAMPLE_PLANS} />);
    // ai_chat is on all three plans; every other paid feature is not on Free.
    expect(screen.getAllByRole("img", { name: PRICING.comparison.available }).length).toBe(14);
    expect(screen.getAllByRole("img", { name: PRICING.comparison.unavailable }).length).toBe(7);
  });

  it("shows only the limits the backend enforces", () => {
    render(<ComparisonTable plans={SAMPLE_PLANS} />);
    buildLimitRows(SAMPLE_PLANS).forEach((row) =>
      expect(screen.getByRole("rowheader", { name: row.label })).toBeInTheDocument(),
    );
    // Not enforced in the catalog -> must never be advertised.
    expect(screen.queryByRole("rowheader", { name: "Kategori" })).not.toBeInTheDocument();
    expect(screen.queryByRole("rowheader", { name: "Dompet" })).not.toBeInTheDocument();
    expect(screen.queryByRole("rowheader", { name: "bank_sync" })).not.toBeInTheDocument();
  });

  it("renders '-1' as 'Tanpa batas' and never as a raw number", () => {
    const { container } = render(<ComparisonTable plans={SAMPLE_PLANS} />);
    // Four unbounded cells on the Business plan.
    expect(screen.getAllByText("Tanpa batas")).toHaveLength(4);
    expect(container.textContent).not.toContain("-1");
  });

  it("formats bounded limits with the catalog unit", () => {
    render(<ComparisonTable plans={SAMPLE_PLANS} />);
    expect(screen.getByText("5.000 token")).toBeInTheDocument();
    expect(screen.getByText("150.000 token")).toBeInTheDocument();
  });

  it("never mentions a bank-sync feature", () => {
    const { container } = render(<ComparisonTable plans={SAMPLE_PLANS} />);
    const text = container.textContent ?? "";
    expect(text).not.toMatch(/\bbank\b/i);
    expect(text).not.toMatch(/bank_sync/i);
  });
});
