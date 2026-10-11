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
