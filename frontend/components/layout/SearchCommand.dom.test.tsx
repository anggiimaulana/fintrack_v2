import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { axe } from "@/test/setup";
import { SearchCommand } from "./SearchCommand";

const push = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
}));

beforeEach(() => {
  push.mockClear();
});

describe("SearchCommand", () => {
  it("lists modules when opened with no query", () => {
    render(<SearchCommand open onOpenChange={() => {}} />);
    expect(screen.getByText("Transaksi")).toBeInTheDocument();
    expect(screen.getByText("AI Chat")).toBeInTheDocument();
  });

  it("filters modules/features by query", () => {
    render(<SearchCommand open onOpenChange={() => {}} />);
    fireEvent.change(screen.getByLabelText("Kata kunci pencarian"), { target: { value: "laporan" } });
    expect(screen.getByText("Laporan")).toBeInTheDocument();
    expect(screen.queryByText("AI Chat")).not.toBeInTheDocument();
  });

  it("navigates to the module route on click", () => {
    render(<SearchCommand open onOpenChange={() => {}} />);
    fireEvent.change(screen.getByLabelText("Kata kunci pencarian"), { target: { value: "profil" } });
    fireEvent.click(screen.getByText("Profil & Pengaturan"));
    expect(push).toHaveBeenCalledWith("/user/profile");
  });

  it("navigates to the highlighted result on Enter", () => {
    render(<SearchCommand open onOpenChange={() => {}} />);
    const input = screen.getByLabelText("Kata kunci pencarian");
    fireEvent.change(input, { target: { value: "transaksi" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(push).toHaveBeenCalledWith("/user/transactions");
  });

  it("shows an empty state for a query with no matches", () => {
    render(<SearchCommand open onOpenChange={() => {}} />);
    fireEvent.change(screen.getByLabelText("Kata kunci pencarian"), { target: { value: "zzzznope" } });
    expect(screen.getByText(/Tidak ada hasil/)).toBeInTheDocument();
  });

  it("has no accessibility violations", async () => {
    const { container } = render(<SearchCommand open onOpenChange={() => {}} />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
