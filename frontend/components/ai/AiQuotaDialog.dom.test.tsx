import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import * as React from "react";
import { AiQuotaDialog } from "./AiQuotaDialog";

const mockUseEntitlements = vi.fn();

vi.mock("@/hooks/use-entitlements", () => ({
  useEntitlements: () => mockUseEntitlements(),
}));

describe("AiQuotaDialog", () => {
  const onOpenChange = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseEntitlements.mockReturnValue({
      planName: "Pro",
      period: {
        start: "2026-10-15T00:00:00Z",
        end: "2026-11-14T00:00:00Z",
      },
      usage: {
        ai_tokens_per_cycle: {
          used: 43544,
          limit: 150000,
          remaining: 106456,
          unlimited: false,
          unit: "kuota",
        },
        ai_requests_per_cycle: {
          used: 5,
          limit: 2000,
          remaining: 1995,
          unlimited: false,
          unit: "permintaan",
        },
      },
    });
  });

  it("renders quota information and usage breakdown when open", () => {
    render(<AiQuotaDialog open={true} onOpenChange={onOpenChange} />);

    expect(screen.getByText("Informasi Kuota AI")).toBeInTheDocument();
    expect(screen.getByText("Paket Pro")).toBeInTheDocument();
    expect(screen.getByText(/106\.456 tersisa/)).toBeInTheDocument();
    expect(screen.getByText(/1\.995 tersisa/)).toBeInTheDocument();
  });

  it("provides a link button to /user/usage", () => {
    render(<AiQuotaDialog open={true} onOpenChange={onOpenChange} />);

    const usageLink = screen.getByRole("link", { name: /Lihat Detail Penggunaan/i });
    expect(usageLink).toHaveAttribute("href", "/user/usage");
  });

  it("calls onOpenChange(false) when Tutup is clicked", () => {
    render(<AiQuotaDialog open={true} onOpenChange={onOpenChange} />);

    const closeBtn = screen.getByRole("button", { name: "Tutup" });
    fireEvent.click(closeBtn);

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
