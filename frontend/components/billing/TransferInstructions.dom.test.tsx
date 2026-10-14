import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { axe } from "@/test/setup";
import { TransferInstructions } from "./TransferInstructions";
import { usePaymentInstructions } from "@/hooks/use-billing";

vi.mock("@/hooks/use-billing", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/hooks/use-billing")>();
  return { ...actual, usePaymentInstructions: vi.fn() };
});

const mocked = vi.mocked(usePaymentInstructions);

function setInstructions(value: Partial<ReturnType<typeof usePaymentInstructions>>) {
  mocked.mockReturnValue({ instructions: null, isLoading: false, ...value } as ReturnType<typeof usePaymentInstructions>);
}

beforeEach(() => {
  mocked.mockReset();
});

describe("TransferInstructions", () => {
  it("renders every configured destination", () => {
    setInstructions({
      instructions: {
        amount_idr: 43290,
        order_id: "INV/1",
        accounts: [
          { kind: "bank", name: "BCA", account_number: "1234567890", account_holder: "PT FinTrack" },
          { kind: "ewallet", name: "DANA", account_number: "081234567890", account_holder: "FinTrack" },
          { kind: "qris", name: "QRIS", qris_url: "https://example.com/q.png" },
        ],
      },
    });
    render(<TransferInstructions paymentId="p1" amountIdr={43290} />);

    expect(screen.getByText("BCA")).toBeInTheDocument();
    expect(screen.getByText("DANA")).toBeInTheDocument();
    // "QRIS" is both the account name and its kind badge, so assert the image.
    expect(screen.getByAltText("Kode QRIS QRIS")).toBeInTheDocument();
    // Amount + order id are shown once, not per account.
    expect(screen.getByText(/43\.290/)).toBeInTheDocument();
    expect(screen.getByText("INV/1")).toBeInTheDocument();
    // QRIS renders its image.
    expect(screen.getByAltText("Kode QRIS QRIS")).toBeInTheDocument();
  });

  it("falls back to the legacy single account when no accounts exist", () => {
    setInstructions({
      instructions: { amount_idr: 1000, order_id: "INV/2", bank_name: "BCA", account_number: "999", account_holder: "PT X" },
    });
    render(<TransferInstructions paymentId="p2" amountIdr={1000} />);
    expect(screen.getByText("BCA")).toBeInTheDocument();
    expect(screen.getByText("999")).toBeInTheDocument();
    expect(screen.queryByText(/Pilih Salah Satu/)).not.toBeInTheDocument();
  });

  it("shows a contact-admin hint when nothing is configured", () => {
    setInstructions({ instructions: { amount_idr: 5000, order_id: "INV/3" } });
    render(<TransferInstructions paymentId="p3" amountIdr={5000} />);
    expect(screen.getByText(/Hubungi admin/)).toBeInTheDocument();
  });

  it("shows a loading state", () => {
    setInstructions({ isLoading: true });
    render(<TransferInstructions paymentId="p4" amountIdr={1} />);
    expect(screen.getByText(/Memuat instruksi/)).toBeInTheDocument();
  });

  it("has no accessibility violations with multiple accounts", async () => {
    setInstructions({
      instructions: {
        amount_idr: 43290,
        order_id: "INV/9",
        accounts: [
          { kind: "bank", name: "BCA", account_number: "123", account_holder: "PT F" },
          { kind: "qris", name: "QRIS", qris_url: "https://example.com/q.png" },
        ],
      },
    });
    const { container } = render(<TransferInstructions paymentId="p5" amountIdr={43290} />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
