"use client";

import { ChevronsUpDown, ChevronUp, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";
import type { SortOrder } from "@/lib/table-params";

interface DataTableColumnHeaderProps {
  title: string;
  /** Column id used as the `sort` URL parameter. */
  columnId: string;
  /** Whether this column is sortable (server-side). */
  sortable?: boolean;
  /** Current active sort column. */
  currentSort?: string;
  /** Current sort direction. */
  currentOrder?: SortOrder;
  /** Called when the user clicks to sort by this column. */
  onSort?: (columnId: string, order: SortOrder) => void;
  className?: string;
}

/**
 * The sort state TanStack injects into a column's `header` render context (see
 * DataTable, which spreads `sort`/`order`/`onSort` into `flexRender`). Declared
 * as a named type and narrowed at runtime so header renderers never need an
 * unchecked inline cast to read it.
 */
export interface SortContext {
  sort?: string;
  order?: SortOrder;
  onSort?: (columnId: string, order: SortOrder) => void;
}

/** Narrow an arbitrary header-render context to the sort fields it may carry. */
export function asSortContext(ctx: unknown): SortContext {
  if (typeof ctx !== "object" || ctx === null) return {};
  const out: SortContext = {};
  if ("sort" in ctx && typeof ctx.sort === "string") out.sort = ctx.sort;
  if ("order" in ctx && (ctx.order === "asc" || ctx.order === "desc")) out.order = ctx.order;
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
  const { sort, order, onSort } = asSortContext(ctx);
  return (
    <DataTableColumnHeader
      title={title}
      columnId={columnId}
      sortable
      currentSort={sort}
      currentOrder={order}
      onSort={onSort}
      className={className}
    />
  );
}

/**
 * Sortable table header. Sorting is server-side: clicking cycles
 * asc → desc → (and back), delegating to `onSort` which updates the URL.
 */
export function DataTableColumnHeader({
  title,
  columnId,
  sortable = false,
  currentSort,
  currentOrder,
  onSort,
  className,
}: DataTableColumnHeaderProps) {
  if (!sortable) {
    return <span className={cn("font-black uppercase tracking-wide text-xs", className)}>{title}</span>;
  }

  const active = currentSort === columnId;
  const nextOrder: SortOrder = active && currentOrder === "asc" ? "desc" : "asc";

  return (
    <button
      type="button"
      onClick={() => onSort?.(columnId, nextOrder)}
      className={cn(
        "inline-flex cursor-pointer items-center gap-1.5 font-black uppercase tracking-wide text-xs hover:opacity-70",
        active && "text-foreground",
        className
      )}
      aria-label={`Sortir berdasarkan ${title} (${nextOrder === "asc" ? "naik" : "turun"})`}
    >
      {title}
      {active ? (
        currentOrder === "asc" ? (
          <ChevronUp className="h-3.5 w-3.5" />
        ) : (
          <ChevronDown className="h-3.5 w-3.5" />
        )
      ) : (
        <ChevronsUpDown className="h-3.5 w-3.5 opacity-50" />
      )}
    </button>
  );
}
