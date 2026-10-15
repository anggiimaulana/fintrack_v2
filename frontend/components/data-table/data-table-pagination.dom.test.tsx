import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { DataTablePagination } from "./data-table-pagination";

/**
 * Regression: a backend that omits `total_pages` made the page count NaN, which
 * React surfaced as "Received NaN for the `children` attribute" and rendered a
 * literal "NaN" page button. The component must coerce unsafe inputs.
 */
describe("DataTablePagination NaN hardening", () => {
  const noop = () => {};

  it("does not render NaN when totalPages is undefined", () => {
    render(
      <DataTablePagination
        page={1}
        perPage={20}
        total={13}
        totalPages={undefined as unknown as number}
        onPageChange={noop}
        onPerPageChange={noop}
      />,
    );
    expect(screen.queryByText("NaN")).not.toBeInTheDocument();
    // Falls back to a derived page count (13 / 20 -> 1 page).
    expect(screen.getByRole("button", { name: "Halaman 1" })).toBeInTheDocument();
  });

  it("does not render NaN when totalPages is NaN", () => {
    render(
      <DataTablePagination
        page={NaN}
        perPage={NaN}
        total={NaN}
        totalPages={NaN}
        onPageChange={noop}
        onPerPageChange={noop}
      />,
    );
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument();
    expect(screen.getByText(/Menampilkan 0–0 dari 0 data/)).toBeInTheDocument();
  });

  it("renders the correct page range and buttons for valid input", () => {
    render(
      <DataTablePagination
        page={2}
        perPage={10}
        total={35}
        totalPages={4}
        onPageChange={noop}
        onPerPageChange={noop}
      />,
    );
    expect(screen.getByText(/Menampilkan 11–20 dari 35 data/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Halaman 4" })).toBeInTheDocument();
  });

  it("invokes onPageChange with a finite number", () => {
    const onPageChange = vi.fn();
    render(
      <DataTablePagination
        page={1}
        perPage={10}
        total={35}
        totalPages={4}
        onPageChange={onPageChange}
        onPerPageChange={noop}
      />,
    );
    screen.getByRole("button", { name: "Halaman berikutnya" }).click();
    expect(onPageChange).toHaveBeenCalledWith(2);
  });
});
