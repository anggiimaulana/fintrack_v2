"use client";

import * as React from "react";
import { Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

interface DataTableToolbarProps {
  /** Current search text (from URL state). */
  search: string;
  /** Called after the debounce interval with the new search text. */
  onSearch: (value: string) => void;
  placeholder?: string;
  /** Active filters, rendered as removable chips. */
  filters?: Record<string, string>;
  /** Called when a filter chip is removed. */
  onRemoveFilter?: (key: string) => void;
  /** Optional extra controls (e.g. filter dropdowns) rendered on the right. */
  children?: React.ReactNode;
  /** Debounce delay in ms (default 300). */
  debounceMs?: number;
}

/**
 * Table toolbar with a debounced search box and removable filter chips.
 * Search changes are debounced so the URL/query is not updated on every
 * keystroke.
 */
export function DataTableToolbar({
  search,
  onSearch,
  placeholder = "Cari...",
  filters = {},
  onRemoveFilter,
  children,
  debounceMs = 300,
}: DataTableToolbarProps) {
  const [value, setValue] = React.useState(search);
  const [prevSearch, setPrevSearch] = React.useState(search);

  // Sync local input when the URL search changes externally (back/forward).
  // Adjusting state during render is the React-recommended alternative to a
  // setState-in-effect and avoids cascading renders.
  if (search !== prevSearch) {
    setPrevSearch(search);
    setValue(search);
  }

  // Debounce: fire onSearch only after the user stops typing.
  React.useEffect(() => {
    if (value === search) return;
    const t = setTimeout(() => onSearch(value), debounceMs);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value, debounceMs]);

  const filterEntries = Object.entries(filters).filter(([, v]) => v !== "");

  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <div className="flex flex-1 flex-wrap items-center gap-2">
        <div className="relative w-full max-w-xs">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder={placeholder}
            className="pl-8"
            aria-label="Cari"
          />
        </div>

        {filterEntries.map(([key, v]) => (
          <span
            key={key}
            className="inline-flex items-center gap-1 border-2 border-foreground bg-muted px-2 py-1 text-xs font-bold"
          >
            {key}: {v}
            {onRemoveFilter && (
              <button
                type="button"
                onClick={() => onRemoveFilter(key)}
                className="cursor-pointer hover:opacity-70"
                aria-label={`Hapus filter ${key}`}
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </span>
        ))}
      </div>

      <div className="flex items-center gap-2">{children}</div>
    </div>
  );
}

/** Convenience: a "reset" button for toolbars. */
export function DataTableResetButton({ onClick }: { onClick: () => void }) {
  return (
    <Button type="button" variant="outline" size="sm" onClick={onClick} className="cursor-pointer">
      Reset
    </Button>
  );
}
