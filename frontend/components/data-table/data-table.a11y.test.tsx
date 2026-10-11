import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { axe } from "vitest-axe";
import type { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "./data-table";
import { DataTableToolbar } from "./data-table-toolbar";

interface Row {
  id: string;
  name: string;
  amount: number;
}

const columns: ColumnDef<Row, unknown>[] = [
  { accessorKey: "name", header: "Nama" },
  { accessorKey: "amount", header: "Jumlah" },
];

const rows: Row[] = [
  { id: "1", name: "Transaksi A", amount: 1000 },
  { id: "2", name: "Transaksi B", amount: 2000 },
];

describe("DataTable a11y", () => {
  it("has no axe violations with data", async () => {
    const { container } = render(<DataTable columns={columns} data={rows} />);
    expect(await axe(container)).toHaveNoViolations();
  });

  it("has no axe violations when empty", async () => {
    const { container } = render(<DataTable columns={columns} data={[]} emptyMessage="Belum ada data." />);
    expect(await axe(container)).toHaveNoViolations();
  });

  it("renders a real table with header cells", () => {
    render(<DataTable columns={columns} data={rows} />);
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getAllByRole("columnheader").length).toBeGreaterThanOrEqual(2);
  });
});

describe("DataTableToolbar a11y", () => {
  it("has no axe violations", async () => {
    const { container } = render(
      <DataTableToolbar search="" onSearch={() => {}} placeholder="Cari transaksi..." />,
    );
    expect(await axe(container)).toHaveNoViolations();
  });

  it("gives the search input an accessible name", () => {
    render(<DataTableToolbar search="" onSearch={() => {}} placeholder="Cari transaksi..." />);
    // An input must be reachable by its accessible name, not just a placeholder.
    const input = screen.getByRole("textbox");
    expect(input).toHaveAccessibleName();
  });
});
