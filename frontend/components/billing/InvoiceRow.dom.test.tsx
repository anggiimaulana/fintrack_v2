import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { axe } from "@/test/setup";
import { InvoiceRow } from "./InvoiceRow";
import { useCancelInvoice } from "@/hooks/use-billing";
import type { Invoice } from "@/hooks/use-billing";

vi.mock("@/hooks/use-billing", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/hooks/use-billing")>();
  return { ...actual, useCancelInvoice: vi.fn() };
});

const mocked = vi.mocked(useCancelInvoice);

function makeInvoice(over: Partial<Invoice> = {}): Invoice {
  return {
    id: "inv-1",
    invoice_number: "INV/20261014/00001",
    status: "issued",
    subtotal_idr: 39000,
    tax_idr: 4290,
    total_idr: 43290,
    issued_at: new Date(Date.now() - 3600_000).toISOString(),
    due_at: new Date(Date.now() + 2 * 3600_000).toISOString(),
    ...over,
  };
}

beforeEach(() => {
  mocked.mockReset();
  mocked.mockReturnValue({ mutateAsync: vi.fn().mockResolvedValue({}), isPending: false } as never);
});

describe("InvoiceRow", () => {
  it("shows a live countdown for an open invoice", () => {
    render(
      <table><tbody><InvoiceRow invoice={makeInvoice()} /></tbody></table>
    );
    expect(screen.getByText("INV/20261014/00001")).toBeInTheDocument();
    // ~2h remaining -> "2j 0m lagi"
    expect(screen.getByText(/lagi/)).toBeInTheDocument();
  });

  it("offers a cancel action only for a cancellable invoice", () => {
    const { rerender } = render(
      <table><tbody><InvoiceRow invoice={makeInvoice()} /></tbody></table>
    );
    expect(screen.getByText("Batalkan")).toBeInTheDocument();

    rerender(<table><tbody><InvoiceRow invoice={makeInvoice({ status: "paid" })} /></tbody></table>);
    expect(screen.queryByText("Batalkan")).not.toBeInTheDocument();
  });

  it("confirms before cancelling, then calls the API", async () => {
    const mutateAsync = vi.fn().mockResolvedValue({});
    mocked.mockReturnValue({ mutateAsync, isPending: false } as never);

    render(<table><tbody><InvoiceRow invoice={makeInvoice()} /></tbody></table>);
    fireEvent.click(screen.getByText("Batalkan"));

    // A confirmation must appear first — no request yet.
    expect(mutateAsync).not.toHaveBeenCalled();
    expect(screen.getByText("Batalkan Tagihan?")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Ya, Batalkan"));
    expect(mutateAsync).toHaveBeenCalledWith("inv-1");
  });

  it("explains why a voided invoice was cancelled", () => {
    render(
      <table><tbody><InvoiceRow invoice={makeInvoice({ status: "voided", closed_reason: "cancelled_by_user" })} /></tbody></table>
    );
    expect(screen.getByText("Dibatalkan oleh Anda")).toBeInTheDocument();
  });

  it("labels an auto-expired invoice", () => {
    render(
      <table><tbody><InvoiceRow invoice={makeInvoice({ status: "expired", closed_reason: "expired" })} /></tbody></table>
    );
    expect(screen.getByText("Kedaluwarsa")).toBeInTheDocument();
    expect(screen.getByText("Kedaluwarsa otomatis")).toBeInTheDocument();
  });

  it("shows the tax portion when present", () => {
    render(<table><tbody><InvoiceRow invoice={makeInvoice()} /></tbody></table>);
    expect(screen.getByText(/incl\. pajak/)).toBeInTheDocument();
  });

  it("has no accessibility violations", async () => {
    const { container } = render(
      <table><tbody><InvoiceRow invoice={makeInvoice()} /></tbody></table>
    );
    expect(await axe(container)).toHaveNoViolations();
  });
});
