import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { QuotaUsageCard } from "./QuotaUsageCard";
import type { UsageEntry } from "@/hooks/use-entitlements";

function entry(over: Partial<UsageEntry> = {}): UsageEntry {
  return { used: 0, limit: 100, remaining: 100, unlimited: false, ...over };
}

describe("QuotaUsageCard", () => {
  it("renders a bar for the daily voice quota", () => {
    render(
      <QuotaUsageCard
        usage={{
          ai_tokens_per_cycle: entry({ used: 1000, limit: 150000, remaining: 149000 }),
          voice_requests_per_day: entry({ used: 3, limit: 50, remaining: 47, unit: "permintaan" }),
        }}
      />
    );
    expect(screen.getByText("Input suara (per hari)")).toBeInTheDocument();
    expect(screen.getByText("3 / 50")).toBeInTheDocument();
  });

  it("shows every enforced limit the backend reports", () => {
    render(
      <QuotaUsageCard
        usage={{
          ai_tokens_per_cycle: entry(),
          ai_requests_per_cycle: entry(),
          ocr_scans_per_cycle: entry(),
          voice_requests_per_day: entry(),
          max_transactions: entry(),
        }}
      />
    );
    for (const label of [
      "Kuota AI",
      "Permintaan AI",
      "Pindai struk",
      "Input suara (per hari)",
      "Transaksi",
    ]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
  });

  it("renders unlimited limits without a percentage bar", () => {
    render(
      <QuotaUsageCard
        usage={{ voice_requests_per_day: entry({ used: 12, limit: -1, remaining: -1, unlimited: true }) }}
      />
    );
    expect(screen.getByText("12 / ∞")).toBeInTheDocument();
  });

  it("renders nothing when no known limit is present", () => {
    const { container } = render(<QuotaUsageCard usage={{ unknown_limit: entry() }} />);
    expect(container).toBeEmptyDOMElement();
  });
});
