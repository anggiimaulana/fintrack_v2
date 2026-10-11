"use client";

import * as React from "react";
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from "@tanstack/react-table";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { EmptyState } from "@/components/shared/EmptyState";
import { DataTableSkeleton } from "./data-table-skeleton";
import type { SortOrder } from "@/lib/table-params";

export interface DataTableProps<TData> {
  columns: ColumnDef<TData, unknown>[];
  data: TData[];
  /** Loading state — renders a skeleton. */
  isLoading?: boolean;
  /** Error state — renders an error message row. */
  error?: string | null;
  /** Message shown when there is no data. */
  emptyMessage?: string;
  /**
   * Server-side sort props. When `onSort` is provided, headers built with
   * `DataTableColumnHeader` become interactive.
   */
  sort?: string;
  order?: SortOrder;
  onSort?: (columnId: string, order: SortOrder) => void;
  /** Optional per-row click handler. */
  onRowClick?: (row: TData) => void;
  /** Column count used for the skeleton (defaults to columns.length). */
  skeletonColumns?: number;
}

/**
 * Generic, server-driven data table built on TanStack Table v8.
 *
 * This component is intentionally "dumb": it renders the rows it is given and
 * delegates sorting/pagination/filtering to the server via props (so the URL
 * state hook stays the single source of truth). Pagination and toolbar live in
 * sibling components.
 */
export function DataTable<TData>({
  columns,
  data,
  isLoading = false,
  error = null,
  emptyMessage = "Tidak ada data.",
  sort,
  order,
  onSort,
  onRowClick,
  skeletonColumns,
}: DataTableProps<TData>) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualSorting: true,
    manualPagination: true,
    manualFiltering: true,
  });

  if (isLoading) {
    return <DataTableSkeleton columns={skeletonColumns ?? Math.max(columns.length, 1)} />;
  }

  if (error) {
    return (
      <div className="border-2 border-danger-500 bg-danger-50 p-6 text-sm font-bold text-danger-700">
        Gagal memuat data: {error}
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="border-2 border-foreground bg-background">
        <EmptyState title={emptyMessage} />
      </div>
    );
  }

  return (
    <div className="border-2 border-foreground bg-background">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id} className="hover:bg-transparent bg-neutral-50">
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, {
                        ...header.getContext(),
                        sort,
                        order,
                        onSort,
                      })}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.map((row) => (
            <TableRow
              key={row.id}
              onClick={onRowClick ? () => onRowClick(row.original) : undefined}
              className={onRowClick ? "cursor-pointer" : undefined}
            >
              {row.getVisibleCells().map((cell) => (
                <TableCell key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
