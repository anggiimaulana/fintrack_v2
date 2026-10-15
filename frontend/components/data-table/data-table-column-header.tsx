"use client";

import { ChevronsUpDown, ChevronUp, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";
import type { SortOrder, SortSpec } from "@/lib/table-params";

interface DataTableColumnHeaderProps {
  title: string;
  /** Column id used as the `sort` URL parameter. */
  columnId: string;
  /** Whether this column is sortable (server-side). */
  sortable?: boolean;
  /** Current active sort column (primary). */
  currentSort?: string;
  /** Current sort direction (primary). */
  currentOrder?: SortOrder;
  /** Full multi-sort list, used to show secondary-sort indicators. */
  sorts?: SortSpec[];
  /** Called when the user clicks to sort by this column. */
  onSort?: (columnId: string, order: SortOrder, opts?: { additive?: boolean }) => void;
  className?: string;
}

/**
 * The sort state TanStack injects into a column's `header` render context (see
 * DataTable, which spreads `sort`/`order`/`sorts`/`onSort` into `flexRender`).
 * Declared as a named type and narrowed at runtime so header renderers never
 * need an unchecked inline cast to read it.
 */
export interface SortContext {
  sort?: string;
  order?: SortOrder;
  sorts?: SortSpec[];
  onSort?: (columnId: string, order: SortOrder, opts?: { additive?: boolean }) => void;
}

/** Narrow an arbitrary header-render context to the sort fields it may carry. */
export function asSortContext(ctx: unknown): SortContext {
  if (typeof ctx !== "object" || ctx === null) return {};
  const out: SortContext = {};
  if ("sort" in ctx && typeof ctx.sort === "string") out.sort = ctx.sort;
  if ("order" in ctx && (ctx.order === "asc" || ctx.order === "desc")) out.order = ctx.order;
  if ("sorts" in ctx && Array.isArray(ctx.sorts)) out.sorts = ctx.sorts as SortSpec[];
  if ("onSort" in ctx && typeof ctx.onSort === "function") {
    out.onSort = ctx.onSort as SortContext["onSort"];
  }
  return out;
}

/**
 * Convenience: a fully-wired column header driven by the render context.
 * Header renderers pass their raw `ctx`; this narrows it once so callers never
 * repeat the cast dance.
 */
export function SortableHeader({
  title,
  columnId,
  ctx,
  className,
}: {
  title: string;
  columnId: string;
  ctx: unknown;
  className?: string;
}) {
  const { sort, order, sorts, onSort } = asSortContext(ctx);
  return (
    <DataTableColumnHeader
      title={title}
      columnId={columnId}
      sortable
      currentSort={sort}
      currentOrder={order}
      sorts={sorts}
      onSort={onSort}
      className={className}
    />
  );
}

/**
 * Sortable table header. Sorting is server-side: clicking cycles asc → desc →
 * (and back). Shift-clicking adds the column as a secondary sort (multi-sort),
 * with the order shown as a small index badge.
 */
export function DataTableColumnHeader({
  title,
  columnId,
  sortable = false,
  currentSort,
  currentOrder,
  sorts,
  onSort,
  className,
}: DataTableColumnHeaderProps) {
  if (!sortable) {
    return <span className={cn("font-black uppercase tracking-wide text-xs", className)}>{title}</span>;
  }

  const sortIndex = sorts?.findIndex((s) => s.column === columnId) ?? -1;
  const inList = sortIndex >= 0;
  const active = inList || currentSort === columnId;
  const activeOrder = inList ? sorts![sortIndex].order : currentOrder;
  const nextOrder: SortOrder = active && activeOrder === "asc" ? "desc" : "asc";

  return (
    <button
      type="button"
      onClick={(e) => onSort?.(columnId, nextOrder, { additive: e.shiftKey })}
      className={cn(
        "inline-flex cursor-pointer items-center gap-1.5 font-black uppercase tracking-wide text-xs hover:opacity-70",
        active && "text-foreground",
        className
      )}
      aria-label={`Sortir berdasarkan ${title} (${nextOrder === "asc" ? "naik" : "turun"})`}
      title="Klik untuk urut, Shift+klik untuk urut multi-kolom"
    >
      {title}
      {active ? (
        activeOrder === "asc" ? (
          <ChevronUp className="h-3.5 w-3.5" />
        ) : (
          <ChevronDown className="h-3.5 w-3.5" />
        )
      ) : (
        <ChevronsUpDown className="h-3.5 w-3.5 opacity-50" />
      )}
      {(sorts?.length ?? 0) > 1 && inList ? (
        <span className="ml-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full border border-foreground bg-foreground px-1 text-[9px] font-black text-background">
          {sortIndex + 1}
        </span>
      ) : null}
    </button>
  );
}
