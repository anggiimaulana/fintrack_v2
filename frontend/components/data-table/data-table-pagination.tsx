"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const PAGE_SIZES = [10, 20, 50, 100];

/** Maximum number of page buttons rendered between the prev/next arrows. */
const MAX_PAGE_BUTTONS = 5;

interface DataTablePaginationProps {
  page: number;
  perPage: number;
  total: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
  /** Optional row-numbering: index of the first row on this page (1-based). */
  showNumbering?: boolean;
}

/**
 * Builds the visible page-button list: always the first page, the last page,
 * and a window of up to `MAX_PAGE_BUTTONS` numbers around the current one,
 * separated by `"gap"` placeholders (rendered as an ellipsis).
 *
 * e.g. page 1 of 13 → [1, 2, 3, 4, "gap", 13]
 *      page 6 of 13 → [1, "gap", 5, 6, 7, "gap", 13]
 *      page 12 of 13 → [1, "gap", 10, 11, 12, 13]
 */
function getPageItems(page: number, totalPages: number): (number | "gap")[] {
  const last = Math.max(totalPages, 1);
  const current = Math.min(Math.max(page, 1), last);

  if (last <= MAX_PAGE_BUTTONS) {
    return Array.from({ length: last }, (_, i) => i + 1);
  }

  // Keep the window anchored to the first/last page when the current page is
  // near either end, so the total page count is always one button away.
  const edge = Math.ceil(MAX_PAGE_BUTTONS / 2);
  let start: number;
  let end: number;
  if (current <= edge) {
    start = 2;
    end = Math.min(last - 1, MAX_PAGE_BUTTONS - 1);
  } else if (current >= last - edge + 1) {
    start = Math.max(2, last - MAX_PAGE_BUTTONS + 2);
    end = last - 1;
  } else {
    start = current - 1;
    end = current + 1;
  }

  const items: (number | "gap")[] = [1];
  if (start > 2) items.push("gap");
  for (let i = start; i <= end; i++) items.push(i);
  if (end < last - 1) items.push("gap");
  items.push(last);
  return items;
}

/**
 * Server-side pagination controls. Shows the record range, a per-page selector
 * (10/20/50/100), and a centered numbered page list with prev/next arrows.
 * All changes flow back to the URL state hook via callbacks.
 */
export function DataTablePagination({
  page,
  perPage,
  total,
  totalPages,
  onPageChange,
  onPerPageChange,
  showNumbering = true,
}: DataTablePaginationProps) {
  const from = total === 0 ? 0 : (page - 1) * perPage + 1;
  const to = Math.min(page * perPage, total);
  const last = Math.max(totalPages, 1);
  const current = Math.min(Math.max(page, 1), last);
  const canPrev = current > 1;
  const canNext = current < last;
  const pageItems = getPageItems(current, last);

  return (
    <div className="flex flex-col items-center gap-3">
      {showNumbering && (
        <p className="text-xs font-bold text-muted-foreground text-center">
          Menampilkan {from}–{to} dari {total} data
        </p>
      )}

      <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-3">
        <div className="flex items-center gap-2">
          <span className="text-xs font-bold text-muted-foreground">Baris</span>
          <Select value={String(perPage)} onValueChange={(v) => onPerPageChange(Number(v))}>
            <SelectTrigger className="h-8 w-[72px] cursor-pointer" aria-label="Baris per halaman">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PAGE_SIZES.map((n) => (
                <SelectItem key={n} value={String(n)} className="cursor-pointer">
                  {n}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <nav className="flex items-center gap-1" aria-label="Navigasi halaman">
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="h-8 w-8 cursor-pointer"
            onClick={() => onPageChange(current - 1)}
            disabled={!canPrev}
            aria-label="Halaman sebelumnya"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>

          {pageItems.map((item, i) =>
            item === "gap" ? (
              <span
                key={`gap-${i}`}
                className="px-1 text-xs font-bold text-muted-foreground select-none"
                aria-hidden="true"
              >
                …
              </span>
            ) : (
              <Button
                key={item}
                type="button"
                variant="outline"
                size="icon"
                aria-current={item === current ? "page" : undefined}
                aria-label={`Halaman ${item}`}
                onClick={() => onPageChange(item)}
                className={`h-8 w-8 cursor-pointer text-xs ${
                  item === current
                    ? "bg-primary text-primary-foreground hover:bg-primary/90"
                    : "hover:bg-muted"
                }`}
              >
                {item}
              </Button>
            )
          )}

          <Button
            type="button"
            variant="outline"
            size="icon"
            className="h-8 w-8 cursor-pointer"
            onClick={() => onPageChange(current + 1)}
            disabled={!canNext}
            aria-label="Halaman berikutnya"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </nav>
      </div>
    </div>
  );
}
