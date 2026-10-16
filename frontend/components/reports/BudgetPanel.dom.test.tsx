import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import * as React from "react";
import { BudgetPanel } from "./BudgetPanel";

const mockUseBudgets = vi.fn();
const mockUseAdminCategories = vi.fn();
const mockUpsertMutateAsync = vi.fn();
const mockDeleteMutateAsync = vi.fn();

vi.mock("@/hooks/useBudgets", () => ({
  useBudgets: () => mockUseBudgets(),
  useUpsertBudget: () => ({
    mutateAsync: mockUpsertMutateAsync,
    isPending: false,
  }),
  useDeleteBudget: () => ({
    mutateAsync: mockDeleteMutateAsync,
    isPending: false,
  }),
}));

vi.mock("@/hooks/useAdminCategories", () => ({
  useAdminCategories: () => mockUseAdminCategories(),
}));

describe("BudgetPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseBudgets.mockReturnValue({
      data: [],
      isLoading: false,
    });
    mockUseAdminCategories.mockReturnValue({
      categories: [{ id: "cat-1", name: "Makanan" }],
      isLoading: false,
    });
  });

  it("formats input digits into rupiah thousands separator 50000 -> 50.000", () => {
    render(<BudgetPanel period={undefined} enabled={true} />);

    const amountInput = screen.getByLabelText(/Nominal/i);
    fireEvent.change(amountInput, { target: { value: "50000" } });

    expect(amountInput).toHaveValue("50.000");
  });

  it("submits budget with numeric amount", async () => {
    mockUpsertMutateAsync.mockResolvedValue({});
    render(<BudgetPanel period={undefined} enabled={true} />);

    const categorySelect = screen.getByLabelText(/Kategori/i);
    fireEvent.change(categorySelect, { target: { value: "cat-1" } });

    const amountInput = screen.getByLabelText(/Nominal/i);
    fireEvent.change(amountInput, { target: { value: "50000" } });

    const submitBtn = screen.getByRole("button", { name: /Simpan/i });
    fireEvent.click(submitBtn);

    expect(mockUpsertMutateAsync).toHaveBeenCalledWith({
      category_id: "cat-1",
      amount_idr: 50000,
    });
  });
});
