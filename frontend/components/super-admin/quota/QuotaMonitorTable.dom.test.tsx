import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { axe } from "@/test/setup";
import { QuotaMonitorTable, QuotaMonitorSummary } from "./QuotaMonitorTable";
import type { QuotaMonitorRow } from "@/hooks/useSuperAdminQuotaMonitor";

function row(partial: Partial<QuotaMonitorRow>): QuotaMonitorRow {
  return {
    user_id: "u1",
    email: "budi@example.com",
    full_name: "Budi",
    plan_code: "pro",
    plan_name: "Pro",
    limit_key: "ai_tokens_per_cycle",
    used: 900,
    limit: 1000,
    used_percent: 90,
    remaining: 100,
    ...partial,
  };
}

const noop = () => {};

describe("QuotaMonitorTable", () => {
  it("renders one row per (user, limit) with usage and remaining", () => {
    render(
      <QuotaMonitorTable
        rows={[
          row({ limit_key: "ai_tokens_per_cycle", used: 900, limit: 1000, used_percent: 90, remaining: 100 }),
          row({
            user_id: "u2",
            full_name: "Siti",
            email: "siti@example.com",
            limit_key: "ai_requests_per_cycle",
            used: 40,
            limit: 50,
            used_percent: 80,
            remaining: 10,
          }),
        ]}
        isLoading={false}
        isError={false}
        threshold={80}
        onRetry={noop}
      />
    );
    expect(screen.getByText("Budi")).toBeInTheDocument();
    expect(screen.getByText("Siti")).toBeInTheDocument();
    expect(screen.getByText("Token AI / siklus")).toBeInTheDocument();
    expect(screen.getByText("Permintaan AI / siklus")).toBeInTheDocument();
    expect(screen.getByText("900 / 1.000")).toBeInTheDocument();
    // Percentages are shown as a band label.
    expect(screen.getByText("90%")).toBeInTheDocument();
    expect(screen.getByText("80%")).toBeInTheDocument();
  });

  it("shows a friendly empty state when nothing crosses the threshold", () => {
    render(
      <QuotaMonitorTable rows={[]} isLoading={false} isError={false} threshold={80} onRetry={noop} />
    );
    expect(screen.getByText("Semua kuota aman")).toBeInTheDocument();
    expect(screen.getByText(/melewati 80%/)).toBeInTheDocument();
  });

  it("shows a retry action on error", () => {
    const onRetry = vi.fn();
    render(
      <QuotaMonitorTable rows={[]} isLoading={false} isError threshold={80} onRetry={onRetry} />
    );
    expect(screen.getByText(/Gagal memuat pemantauan kuota/)).toBeInTheDocument();
  });

  it("exposes the usage as an accessible progressbar", () => {
    render(
      <QuotaMonitorTable
        rows={[row({ used_percent: 95, used: 950, limit: 1000, remaining: 50 })]}
        isLoading={false}
        isError={false}
        threshold={80}
        onRetry={noop}
      />
    );
    expect(screen.getByLabelText(/95% terpakai/)).toBeInTheDocument();
  });

  it("has no accessibility violations", async () => {
    const { container } = render(
      <QuotaMonitorTable
        rows={[row({}), row({ user_id: "u2", email: "siti@example.com", full_name: "Siti", used_percent: 82 })]}
        isLoading={false}
        isError={false}
        threshold={80}
        onRetry={noop}
      />
    );
    expect(await axe(container)).toHaveNoViolations();
  });
});

describe("QuotaMonitorSummary", () => {
  it("counts critical (>=90%) vs near-limit rows", () => {
    render(
      <QuotaMonitorSummary
        rows={[
          row({ used_percent: 95 }),
          row({ used_percent: 90 }),
          row({ used_percent: 85 }),
        ]}
      />
    );
    expect(screen.getByText("2 kritis (≥90%)")).toBeInTheDocument();
    expect(screen.getByText("1 mendekati batas")).toBeInTheDocument();
    expect(screen.getByText(/3 peringatan/)).toBeInTheDocument();
  });
});
