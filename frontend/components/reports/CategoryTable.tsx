"use client";

import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { applyTableQuery, type TableQuery } from "@/lib/client-table";
import type { CategoryReport } from "@/hooks/useReports";
import type { ReportTypeId } from "@/lib/report-tabs";
import { trendIsGood } from "@/lib/report-tabs";
import { formatDisplay } from "@/lib/currency";

/** Report totals are already converted server-side to the report's currency. */
function formatReport(value: number, currencyCode?: string | null): string {
  return formatDisplay(value, currencyCode);
}

/** Row shape includes a 1-based number for display, independent of sort. */
type Row = CategoryReport & { __no: number; trend_good: string };

interface CategoryTableProps {
  categories: CategoryReport[];
  type: ReportTypeId;
  /** Currency the report totals are expressed in (server-converted). */
  currencyCode?: string | null;
  query: TableQuery;
  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
  onSort: (columnId: string, order: "asc" | "desc") => void;
  onSearch: (q: string) => void;
  onFilter: (key: string, value: string) => void;
  onReset: () => void;
}

/**
 * Category detail table: numbered rows with client-side sorting, filtering,
 * debounced search, and pagination. All state lives in the URL (owned by the
 * page) so a view is shareable and survives refresh.
 */
export function CategoryTable({
  categories,
  type,
  currencyCode,
  query,
  onPageChange,
  onPerPageChange,
  onSort,
  onSearch,
  onFilter,
  onReset,
}: CategoryTableProps) {
  // Number rows by their *position in the filtered+sorted list*, so the number
  // stays meaningful when the user sorts.
  const { page, total } = applyTableQuery(
    categories as unknown as Record<string, unknown>[],
    query,
    ["category_name"]
  );

  const startIndex = (query.page - 1) * query.perPage;
  const numbered: Row[] = page.map((r, i) => {
    const c = r as unknown as CategoryReport;
    return {
      ...c,
      __no: startIndex + i + 1,
      trend_good: c.trend === 0 ? "flat" : trendIsGood(type, c.trend) ? "good" : "bad",
    };
  });

  const columns: ColumnDef<Row, unknown>[] = [
    {
      id: "__no",
      header: () => <span className="font-black uppercase tracking-wide text-xs">No</span>,
      cell: ({ row }) => <span className="font-mono text-xs font-bold text-muted-foreground">{row.original.__no}</span>,
    },
    {
      id: "category_name",
      header: () => (
        <DataTableColumnHeader
          title="Kategori"
          columnId="category_name"
          sortable
          currentSort={query.sort}
          currentOrder={query.order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => <span className="font-bold">{row.original.category_name}</span>,
    },
    {
      id: "total",
      header: () => (
        <DataTableColumnHeader
          title="Total"
          columnId="total"
          sortable
          currentSort={query.sort}
          currentOrder={query.order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => <span className="font-mono font-bold">{formatReport(row.original.total, currencyCode)}</span>,
    },
    {
      id: "percentage",
      header: () => (
        <DataTableColumnHeader
          title="%"
          columnId="percentage"
          sortable
          currentSort={query.sort}
          currentOrder={query.order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="h-2 w-20 border rounded-md border-foreground bg-muted">
            <div className="h-full bg-primary" style={{ width: `${Math.min(100, row.original.percentage)}%` }} />
          </div>
          <span className="font-mono text-xs font-bold">{row.original.percentage.toFixed(1)}%</span>
        </div>
      ),
    },
    {
      id: "count",
      header: () => (
        <DataTableColumnHeader
          title="Transaksi"
          columnId="count"
          sortable
          currentSort={query.sort}
          currentOrder={query.order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.count}</span>,
    },
    {
      id: "trend",
      header: () => (
        <DataTableColumnHeader
          title="Trend vs Periode Lalu"
          columnId="trend"
          sortable
          currentSort={query.sort}
          currentOrder={query.order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => {
        const t = row.original;
        if (t.trend === 0) return <span className="font-mono text-xs text-muted-foreground">—</span>;
        // Colour by whether the move is *good* for this report type, not its sign.
        const cls = t.trend_good === "good" ? "text-green-700" : "text-red-600";
        return (
          <span className={`font-mono text-xs font-bold ${cls}`}>
            {t.trend > 0 ? "+" : ""}{t.trend.toFixed(1)}%
          </span>
        );
      },
    },
  ];

  return (
    <div className="space-y-4">
      <DataTableToolbar
        search={query.q}
        onSearch={onSearch}
        placeholder="Cari kategori…"
        filters={query.filters}
        onRemoveFilter={(key) => onFilter(key, "")}
        debounceMs={300}
      >
        <select
          value={query.filters.min_pct || ""}
          onChange={(e) => onFilter("min_pct", e.target.value)}
          className="h-9 cursor-pointer border-2 rounded-md border-foreground bg-background px-2 text-xs font-bold"
          aria-label="Filter porsi minimum"
        >
          <option value="">Semua porsi</option>
          <option value="5">&ge; 5%</option>
          <option value="10">&ge; 10%</option>
          <option value="25">&ge; 25%</option>
        </select>
        <button
          type="button"
          onClick={onReset}
          className="h-9 cursor-pointer border-2 rounded-md border-foreground px-3 text-xs font-bold hover:bg-muted"
        >
          Reset
        </button>
      </DataTableToolbar>

      <DataTable
        columns={columns}
        data={numbered}
        emptyMessage="Belum ada data kategori untuk periode ini"
        sort={query.sort}
        order={query.order}
        onSort={onSort}
      />

      <DataTablePagination
        page={query.page}
        perPage={query.perPage}
        total={total}
        totalPages={Math.max(1, Math.ceil(total / query.perPage))}
        onPageChange={onPageChange}
        onPerPageChange={onPerPageChange}
      />
    </div>
  );
}
