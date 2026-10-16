"use client"

import * as React from "react"
import { Search, RotateCcw } from "lucide-react"
import { type ColumnDef } from "@tanstack/react-table"
import { DataTable } from "@/components/data-table/data-table"
import { DataTablePagination } from "@/components/data-table/data-table-pagination"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import type { UseDataTableUrlState } from "@/hooks/use-data-table-url-state"

export interface DataTableLayoutProps<TData> {
  /** Optional KPI stats cards rendered above the table toolbar. */
  stats?: React.ReactNode
  /** Column definitions for TanStack Table. */
  columns: ColumnDef<TData, unknown>[]
  /** Row data for the current page. */
  data: TData[]
  /** Loading state indicator. */
  isLoading?: boolean
  /** Error message if query fails. */
  error?: string | null
  /** Empty state message. */
  emptyMessage?: string
  /** URL state controller from useDataTableUrlState. */
  tableState: UseDataTableUrlState
  /** Total matching records across all pages. */
  total: number
  /** Total pages derived from total and perPage. */
  totalPages: number
  /** Search input placeholder text. */
  searchPlaceholder?: string
  /** Extra filter components rendered in the toolbar. */
  filters?: React.ReactNode
  /** Optional click handler per row. */
  onRowClick?: (row: TData) => void
}

export function DataTableLayout<TData>({
  stats,
  columns,
  data,
  isLoading = false,
  error = null,
  emptyMessage = "Tidak ada data.",
  tableState,
  total,
  totalPages,
  searchPlaceholder = "Cari data...",
  filters,
  onRowClick,
}: DataTableLayoutProps<TData>) {
  const hasActiveFilters = Boolean(
    tableState.immediate.q ||
      Object.values(tableState.immediate.filters).some((v) => Boolean(v))
  )

  return (
    <div className="space-y-6 w-full">
      {/* 1. Statistics / KPI cards */}
      {stats ? <div className="w-full">{stats}</div> : null}

      {/* 2. Search & Filters Toolbar */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-1 flex-wrap items-center gap-2">
          <div className="relative w-full max-w-xs">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground select-none" />
            <Input
              type="text"
              value={tableState.immediate.q}
              onChange={(e) => tableState.setSearch(e.target.value)}
              placeholder={searchPlaceholder}
              className="pl-9 h-9 font-medium"
            />
          </div>

          {filters}

          {hasActiveFilters && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={tableState.reset}
              className="h-9 px-2.5 text-xs font-bold text-muted-foreground hover:text-foreground cursor-pointer"
            >
              <RotateCcw className="w-3.5 h-3.5 mr-1.5" />
              Reset Filter
            </Button>
          )}
        </div>
      </div>

      {/* 3. Table Container with Horizontal Scroll */}
      <div className="rounded-lg border-2 border-foreground bg-card overflow-hidden shadow-[4px_4px_0px_0px] shadow-foreground">
        <div className="overflow-x-auto min-w-full">
          <DataTable
            columns={columns}
            data={data}
            isLoading={isLoading}
            error={error}
            emptyMessage={emptyMessage}
            sort={tableState.immediate.sort}
            order={tableState.immediate.order}
            sorts={tableState.immediate.sorts}
            onSort={tableState.setSort}
            onRowClick={onRowClick}
          />
        </div>
      </div>

      {/* 4. Numbered Pagination Footer */}
      <div className="pt-2">
        <DataTablePagination
          page={tableState.page}
          perPage={tableState.perPage}
          total={total}
          totalPages={totalPages}
          onPageChange={tableState.setPage}
          onPerPageChange={tableState.setPerPage}
          showNumbering={true}
        />
      </div>
    </div>
  )
}
