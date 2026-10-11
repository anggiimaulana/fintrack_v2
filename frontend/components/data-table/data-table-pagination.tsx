"use client";

import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const PAGE_SIZES = [10, 20, 50, 100];

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
 * Server-side pagination controls. Shows the record range, a per-page selector
 * (10/20/50/100), and first/prev/next/last navigation. All changes flow back to
 * the URL state hook via callbacks.
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
  const canPrev = page > 1;
  const canNext = totalPages > 0 && page < totalPages;

  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      {showNumbering && (
        <p className="text-xs font-bold text-muted-foreground">
          Menampilkan {from}–{to} dari {total} data
        </p>
      )}

      <div className="flex items-center gap-4">
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

        <span className="text-xs font-bold text-muted-foreground">
          Halaman {page} / {Math.max(totalPages, 1)}
        </span>

        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="h-8 w-8 cursor-pointer"
            onClick={() => onPageChange(1)}
            disabled={!canPrev}
            aria-label="Halaman pertama"
          >
            <ChevronsLeft className="h-4 w-4" />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="h-8 w-8 cursor-pointer"
            onClick={() => onPageChange(page - 1)}
            disabled={!canPrev}
            aria-label="Halaman sebelumnya"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="h-8 w-8 cursor-pointer"
            onClick={() => onPageChange(page + 1)}
            disabled={!canNext}
            aria-label="Halaman berikutnya"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="h-8 w-8 cursor-pointer"
            onClick={() => onPageChange(totalPages)}
            disabled={!canNext}
            aria-label="Halaman terakhir"
          >
            <ChevronsRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
