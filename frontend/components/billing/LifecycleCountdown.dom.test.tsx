import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LifecycleCountdown, lifecycleCountdown, formatRemaining } from "./LifecycleCountdown";

describe("formatRemaining", () => {
  it("formats days, hours and minutes", () => {
    expect(formatRemaining(2 * 86_400_000 + 3 * 3_600_000)).toBe("2 hari 3 jam");
    expect(formatRemaining(5 * 3_600_000 + 30 * 60_000)).toBe("5 jam 30 menit");
    expect(formatRemaining(45 * 60_000)).toBe("45 menit");
  });

  it("reports elapsed time as already past", () => {
    expect(formatRemaining(0)).toBe("sudah lewat");
    expect(formatRemaining(-1000)).toBe("sudah lewat");
  });
});

describe("lifecycleCountdown", () => {
  const now = Date.parse("2026-06-01T00:00:00Z");

  it("returns null without an end date", () => {
    expect(lifecycleCountdown("active", undefined, now)).toBeNull();
    expect(lifecycleCountdown("active", "not-a-date", now)).toBeNull();
  });

  it("returns null for terminal subscriptions", () => {
    expect(lifecycleCountdown("expired", "2026-07-01T00:00:00Z", now)).toBeNull();
    expect(lifecycleCountdown("canceled", "2026-07-01T00:00:00Z", now)).toBeNull();
  });

  it("shows an ok countdown while the cycle is running", () => {
    const info = lifecycleCountdown("active", "2026-06-20T00:00:00Z", now);
    expect(info?.tone).toBe("ok");
    expect(info?.label).toBe("Siklus berjalan");
  });

  it("warns when the cycle ends within 3 days", () => {
    const info = lifecycleCountdown("active", "2026-06-02T00:00:00Z", now);
    expect(info?.tone).toBe("warn");
    expect(info?.label).toContain("Berakhir dalam");
  });

  it("shows a danger countdown inside the grace window", () => {
    const info = lifecycleCountdown("active", "2026-05-31T00:00:00Z", now);
    expect(info?.tone).toBe("danger");
    expect(info?.label).toContain("Masa tenggang");
    expect(info?.detail).toContain("Bayar tagihan");
  });

  it("tells a past_due user the plan will lapse after grace", () => {
    const info = lifecycleCountdown("past_due", "2026-05-31T00:00:00Z", now);
    expect(info?.tone).toBe("danger");
    expect(info?.detail).toContain("turun otomatis");
  });
});

describe("LifecycleCountdown", () => {
  it("renders a live countdown for a running cycle", () => {
    const soon = new Date(Date.now() + 10 * 86_400_000).toISOString();
    render(<LifecycleCountdown status="active" periodEnd={soon} />);
    expect(screen.getByText(/Siklus berjalan|Berakhir dalam/i)).toBeInTheDocument();
  });

  it("warns when the cycle ends within 3 days", () => {
    const soon = new Date(Date.now() + 2 * 86_400_000).toISOString();
    render(<LifecycleCountdown status="active" periodEnd={soon} />);
    expect(screen.getByText(/Berakhir dalam/i)).toBeInTheDocument();
  });

  it("shows a grace countdown once the cycle has ended", () => {
    const past = new Date(Date.now() - 3_600_000).toISOString();
    render(<LifecycleCountdown status="active" periodEnd={past} />);
    expect(screen.getByText(/Masa tenggang/i)).toBeInTheDocument();
  });

  it("shows a reactivation hint for an expired subscription", () => {
    render(<LifecycleCountdown status="expired" periodEnd="2026-01-01T00:00:00Z" />);
    expect(screen.getByText(/berakhir\. Pilih paket/i)).toBeInTheDocument();
  });

  it("marks a non-renewing subscription", () => {
    const soon = new Date(Date.now() + 10 * 86_400_000).toISOString();
    render(<LifecycleCountdown status="active" periodEnd={soon} cancelAtPeriodEnd />);
    expect(screen.getByText(/tidak diperpanjang/i)).toBeInTheDocument();
  });
});
