"use client";

import * as React from "react";
import { Suspense, useState, useMemo, useCallback } from "react";
import { useSearchParams, usePathname } from "next/navigation";
import { useDebounce } from "@/hooks/use-debounce";
import type { ColumnDef } from "@tanstack/react-table";
import type { RowSelectionState } from "@tanstack/react-table";
import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/data-table/data-table";
import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { useDataTableUrlState } from "@/hooks/use-data-table-url-state";
import { useEntitlements } from "@/hooks/use-entitlements";
import { useUsageHistory, type UsageEvent } from "@/hooks/use-usage";
import { DualQuotaCard } from "@/components/billing/DualQuotaCard";
import { UsageChart, USAGE_CHART_TYPES, type UsageRange, type UsageChartType } from "@/components/usage/UsageChart";
import { usageHistoryColumns } from "@/components/usage/UsageHistoryColumns";
import { downloadCsv, toCsv } from "@/lib/csv";

const KIND_FILTERS: { key: string; label: string }[] = [
  { key: "", label: "Semua" },
  { key: "transaction", label: "Transaksi" },
  { key: "ai_token", label: "Token AI" },
  { key: "ai_request", label: "Permintaan AI" },
  { key: "ocr", label: "Pindai Struk" },
  { key: "voice", label: "Input Suara" },
];

const RANGE_KEYS: UsageRange[] = ["24h", "7d", "30d", "3m", "1y", "custom"];

export default function UsagePage() {
  return (
    <Suspense fallback={<div className="h-40 animate-pulse rounded-md border-2 border-foreground bg-muted" />}>
      <UsageContent />
    </Suspense>
  );
}

function UsageContent() {
  const searchParams = useSearchParams();
  const pathname = usePathname();

  // All chart + history-date controls live in the URL, so the page is fully
  // shareable and back/forward works. The URL is the single source of truth;
  // there is no mirrored React state to drift out of sync.

  // Range preset (range=24h|7d|30d|3m|1y|custom).
  const rawRange = searchParams.get("range") ?? "";
  const initialRange: UsageRange = (RANGE_KEYS as string[]).includes(rawRange)
    ? (rawRange as UsageRange)
    : "7d";
  const [range, setRange] = useState<UsageRange>(initialRange);
  const [chartStart, setChartStart] = useState(searchParams.get("start") ?? "");
  const [chartEnd, setChartEnd] = useState(searchParams.get("end") ?? "");

  const rawChart = searchParams.get("chart") ?? "";
  const initialChartType: UsageChartType = (USAGE_CHART_TYPES as string[]).includes(rawChart)
    ? (rawChart as UsageChartType)
    : "bar";
  const [chartType, setChartType] = useState<UsageChartType>(initialChartType);

  const initialHidden = useMemo(
    () => (searchParams.get("hide") ?? "").split(",").filter(Boolean),
    [searchParams]
  );
  const [hiddenSeries, setHiddenSeries] = useState<string[]>(initialHidden);

  const [histStart, setHistStart] = useState(searchParams.get("start_date") ?? "");
  const [histEnd, setHistEnd] = useState(searchParams.get("end_date") ?? "");
  // Rewrite the URL with the given overrides, preserving every other param
  // (including the history table's page/sort/filter namespace). Uses
  // router.replace so Next keeps its own state in sync, with no scroll jump.
  const syncUrl = useCallback(
    (over: Record<string, string | undefined>) => {
      if (typeof window === "undefined") return;
      const next = new URLSearchParams(window.location.search);
      for (const [k, v] of Object.entries(over)) {
        if (v) next.set(k, v);
        else next.delete(k);
      }
      const qs = next.toString();
      window.history.replaceState(null, "", qs ? `${pathname}?${qs}` : pathname);
    },
    [pathname]
  );

  const handleRangeChange = (r: UsageRange) => {
    setRange(r);
    syncUrl({ range: r, start: undefined, end: undefined });
  };

  const handleCustomChange = (start: string, end: string) => {
    setRange("custom");
    setChartStart(start);
    setChartEnd(end);
    syncUrl({ range: "custom", start, end });
  };

  const handleChartTypeChange = (t: UsageChartType) => {
    setChartType(t);
    syncUrl({ chart: t });
  };

  const handleToggleSeries = (key: string) => {
    const nextHidden = hiddenSeries.includes(key)
      ? hiddenSeries.filter((k) => k !== key)
      : [...hiddenSeries, key];
    setHiddenSeries(nextHidden);
    syncUrl({ hide: nextHidden.length > 0 ? nextHidden.join(",") : undefined });
  };

  const setHistoryDateParams = (start: string, end: string) => {
    setHistStart(start);
    setHistEnd(end);
    syncUrl({ start_date: start || undefined, end_date: end || undefined });
  };

  const table = useDataTableUrlState({ perPage: 20 });
  const filterKind = table.filters.kind || "";

  const { usage, period } = useEntitlements();

  const debouncedHistStart = useDebounce(histStart, 300);
  const debouncedHistEnd = useDebounce(histEnd, 300);

  const { events, total, isLoading, isError, refetch } = useUsageHistory({
    page: table.debounced.page,
    perPage: table.debounced.perPage,
    sorts: table.debounced.sorts,
    q: table.debounced.q,
    filters: table.debounced.filters,
    startDate: debouncedHistStart || undefined,
    endDate: debouncedHistEnd || undefined,
  });

  const [rowSelection, setRowSelection] = React.useState<RowSelectionState>({});
  // Clear selection when the result set changes (page/sort/filter), so the
  // "export selected" action never exports a stale cross-page selection.
  const [selectionKey, setSelectionKey] = React.useState("");
  const currentKey = `${table.page}|${table.sort}|${table.order}|${table.q}|${JSON.stringify(table.filters)}`;
  if (currentKey !== selectionKey) {
    setSelectionKey(currentKey);
    setRowSelection({});
  }

  const columns = React.useMemo<ColumnDef<UsageEvent, unknown>[]>(() => {
    const selectColumn: ColumnDef<UsageEvent, unknown> = {
      id: "select",
      header: ({ table: t }) => (
        <input
          type="checkbox"
          checked={t.getIsAllPageRowsSelected()}
          ref={(el) => {
            if (el) el.indeterminate = t.getIsSomePageRowsSelected();
          }}
          onChange={t.getToggleAllPageRowsSelectedHandler()}
          className="h-4 w-4 cursor-pointer accent-primary"
          aria-label="Pilih semua baris di halaman ini"
        />
      ),
      enableSorting: false,
      cell: ({ row }) => (
        <input
          type="checkbox"
          checked={row.getIsSelected()}
          onChange={row.getToggleSelectedHandler()}
          className="h-4 w-4 cursor-pointer accent-primary"
          aria-label="Pilih baris"
        />
      ),
    };
    return [selectColumn, ...usageHistoryColumns({ page: table.page, perPage: table.perPage })];
  }, [table.page, table.perPage]);

  const selectedRows = React.useMemo(
    () => events.filter((e) => rowSelection[e.id]),
    [events, rowSelection]
  );

  const handleExport = () => {
    const rows = (selectedRows.length > 0 ? selectedRows : events).map((e) => [
      e.kind_label,
      e.description,
      e.amount,
      e.unit,
      e.cost_idr,
      new Date(e.created_at).toISOString(),
    ]);
    const csv = toCsv(["Jenis", "Keterangan", "Jumlah", "Satuan", "Biaya (IDR)", "Waktu"], rows);
    downloadCsv(`penggunaan-${new Date().toISOString().slice(0, 10)}.csv`, csv);
  };

  return (
    <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-black uppercase tracking-wide text-foreground">Penggunaan</h1>
        <p className="text-xs font-bold text-muted-foreground">
          Pantau pemakaian kuota, tren pemakaian, dan riwayat konsumsi Anda.
        </p>
      </div>

      {/* Detail penggunaan saat ini (dual weekly + monthly quota engine). */}
      <DualQuotaCard />

      <UsageChart
        range={range}
        start={chartStart || undefined}
        end={chartEnd || undefined}
        chartType={chartType}
        hiddenSeries={hiddenSeries}
        onRangeChange={handleRangeChange}
        onCustomChange={handleCustomChange}
        onChartTypeChange={handleChartTypeChange}
        onToggleSeries={handleToggleSeries}
      />

      {/* Full usage history ledger. */}
      <div className="flex flex-col overflow-hidden rounded-md border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
        <div className="flex flex-col justify-between gap-4 border-b-2 border-foreground bg-muted p-4 md:flex-row md:items-center">
          <div className="w-full overflow-x-auto pb-1 -mb-1">
            <div className="flex min-w-max rounded-md border-2 border-foreground bg-background p-1 md:w-auto">
              {KIND_FILTERS.map((f) => (
                <button
                  key={f.key || "all"}
                  type="button"
                  onClick={() => table.setFilter("kind", f.key)}
                  className={`flex-1 cursor-pointer whitespace-nowrap px-3 py-1.5 text-xs font-bold transition-colors md:flex-none ${
                    filterKind === f.key
                      ? "border-x-2 border-foreground bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {f.label}
                </button>
              ))}
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <div className="flex items-center gap-1.5">
              <label htmlFor="hist-from" className="text-[11px] font-bold uppercase text-muted-foreground">Dari</label>
              <input
                id="hist-from"
                type="date"
                value={histStart}
                onChange={(e) => setHistoryDateParams(e.target.value, histEnd)}
                className="rounded-md border-2 border-foreground bg-background px-2 py-1 text-xs font-bold focus:outline-none"
              />
            </div>
            <div className="flex items-center gap-1.5">
              <label htmlFor="hist-to" className="text-[11px] font-bold uppercase text-muted-foreground">Ke</label>
              <input
                id="hist-to"
                type="date"
                value={histEnd}
                onChange={(e) => setHistoryDateParams(histStart, e.target.value)}
                className="rounded-md border-2 border-foreground bg-background px-2 py-1 text-xs font-bold focus:outline-none"
              />
            </div>
            {(histStart || histEnd) && (
              <button
                type="button"
                onClick={() => setHistoryDateParams("", "")}
                className="cursor-pointer rounded-md border-2 border-transparent px-2 py-1 text-xs font-bold text-muted-foreground hover:border-foreground hover:text-foreground"
              >
                Reset
              </button>
            )}
          </div>

          <DataTableToolbar
            search={table.q}
            onSearch={(q) => table.setSearch(q)}
            placeholder="Cari keterangan..."
          >
            <Button
              type="button"
              variant="outline"
              className="cursor-pointer"
              onClick={handleExport}
              disabled={events.length === 0}
            >
              <Download className="mr-2 h-4 w-4" />
              {selectedRows.length > 0 ? `Export (${selectedRows.length})` : "Export CSV"}
            </Button>
          </DataTableToolbar>
        </div>

        <DataTable
          columns={columns}
          data={events}
          isLoading={isLoading}
          error={isError ? "Gagal memuat riwayat penggunaan." : null}
          emptyMessage="Tidak ada riwayat penggunaan ditemukan."
          sort={table.sort}
          order={table.order}
          sorts={table.sorts}
          onSort={table.setSort}
          enableRowSelection
          rowSelection={rowSelection}
          onRowSelectionChange={(updater) => {
            const next = typeof updater === "function" ? updater(rowSelection) : updater;
            setRowSelection(next);
          }}
          getRowId={(e) => e.id}
        />

        <div className="border-t-2 border-foreground bg-muted/40 p-4">
          <DataTablePagination
            page={table.page}
            perPage={table.perPage}
            total={total}
            totalPages={Math.max(1, Math.ceil(total / table.perPage))}
            onPageChange={table.setPage}
            onPerPageChange={table.setPerPage}
          />
        </div>

        {isError && (
          <div className="border-t-2 border-foreground p-4">
            <Button variant="outline" className="cursor-pointer" onClick={() => refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
