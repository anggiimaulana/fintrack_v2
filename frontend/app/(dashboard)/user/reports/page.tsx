"use client";

import * as React from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
import { usePathname, useSearchParams, useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { useDebounce } from "@/hooks/use-debounce";
import { Download, Sparkles, FileSpreadsheet, FileText, FileType } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { EmptyState } from "@/components/shared/EmptyState";
import { DateRangeFilter, type DateRangeValue, type PresetKey, computePresetDates } from "@/components/shared/DateRangeFilter";
import { useReports, useReportsRealtime, normalizeReport, type CategoryReport, type MonthlyComparison } from "@/hooks/useReports";
import { useCategoryChart } from "@/hooks/useDashboard";
import { mapSpk, mapSpkHistory } from "@/hooks/useSpk";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";
import api from "@/lib/api";
import { useEntitlements } from "@/hooks/use-entitlements";
import { useReportAnalytics, useExportReport } from "@/hooks/useReportAnalytics";
import { ReportsSummaryTab } from "@/components/reports/ReportsSummaryTab";
import { ReportsSpkTab } from "@/components/reports/ReportsSpkTab";
import { CategoryTable } from "@/components/reports/CategoryTable";
import { TopMoversPanel, AnomalyPanel } from "@/components/reports/TopMoversPanel";
import { CalendarHeatmap } from "@/components/reports/CalendarHeatmap";
import { BudgetPanel } from "@/components/reports/BudgetPanel";
import {
  REPORT_TABS,
  REPORT_TYPES,
  parseTab,
  parseReportType,
  parseReportFilterParams,
  reportFilterToParams,
  typeLabel,
  type ReportTypeId,
} from "@/lib/report-tabs";
import type { TableQuery } from "@/lib/client-table";
import { toast } from "sonner";
import { formatDisplay } from "@/lib/currency";

// Charts pull in recharts (~500 KB); keep them code-split and below the fold.
const chartFallback = <Skeleton className="h-full w-full border-2 rounded-md border-foreground" />;
const PieChartView = dynamic(() => import("@/components/charts/PieChartView").then((m) => m.PieChartView), { ssr: false, loading: () => chartFallback });
const BarChartView = dynamic(() => import("@/components/charts/BarChartView").then((m) => m.BarChartView), { ssr: false, loading: () => chartFallback });
const MultiSeriesChart = dynamic(() => import("@/components/charts/MultiSeriesChart").then((m) => m.MultiSeriesChart), { ssr: false, loading: () => chartFallback });

function formatIDR(value: number, currencyCode?: string | null): string {
  // Report totals already arrive converted to the report's currency.
  return formatDisplay(value, currencyCode);
}

const DEFAULT_PER_PAGE = 20;

export default function ReportsPage() {
  return (
    <React.Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 rounded-md border-foreground" />}>
      <ReportsContent />
    </React.Suspense>
  );
}

function ReportsContent() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const now = React.useMemo(() => new Date(), []);

  // ── URL-backed state ─────────────────────────────────────────
  const [tab, setTabState] = React.useState(() => parseTab(searchParams.get("tab")));
  const [reportType, setReportTypeState] = React.useState<ReportTypeId>(() => parseReportType(searchParams.get("type")));
  const [dateFilter, setDateFilterState] = React.useState<DateRangeValue>(() => parseReportFilterParams(searchParams, now));

  const query: TableQuery = {
    page: Math.max(1, Number(searchParams.get("page")) || 1),
    perPage: Number(searchParams.get("per_page")) || DEFAULT_PER_PAGE,
    sort: searchParams.get("sort") ?? "",
    order: searchParams.get("order") === "asc" ? "asc" : "desc",
    q: searchParams.get("q") ?? "",
    filters: {
      min_pct: searchParams.get("filter_min_pct") ?? "",
    },
  };

  /** Applies a shallow URL update without RSC navigation overhead. */
  const updateParams = React.useCallback(
    (patch: Record<string, string | null>) => {
      if (typeof window === "undefined") return;
      const p = new URLSearchParams(window.location.search);
      for (const [k, v] of Object.entries(patch)) {
        if (v === null || v === "") p.delete(k);
        else p.set(k, v);
      }
      const qs = p.toString();
      const nextUrl = qs ? `${pathname}?${qs}` : pathname;
      window.history.replaceState(null, "", nextUrl);
      router.replace(nextUrl, { scroll: false });
    },
    [pathname, router]
  );

  const setTab = (t: string) => {
    const parsed = parseTab(t);
    setTabState(parsed);
    updateParams({ tab: parsed });
  };
  const setType = (t: ReportTypeId) => {
    setReportTypeState(t);
    updateParams({ type: t, page: null });
  };
  const setDateFilter = (val: DateRangeValue) => {
    setDateFilterState(val);
    updateParams(reportFilterToParams(val));
  };

  // Sync state if browser back/forward or external URL navigation occurs
  React.useEffect(() => {
    const nextTab = parseTab(searchParams.get("tab"));
    setTabState((curr) => (curr !== nextTab ? nextTab : curr));

    const nextType = parseReportType(searchParams.get("type"));
    setReportTypeState((curr) => (curr !== nextType ? nextType : curr));

    const nextFilter = parseReportFilterParams(searchParams, now);
    setDateFilterState((curr) => {
      if (
        curr.preset === nextFilter.preset &&
        curr.startDate === nextFilter.startDate &&
        curr.endDate === nextFilter.endDate &&
        curr.enableCompare === nextFilter.enableCompare &&
        curr.compareStart === nextFilter.compareStart &&
        curr.compareEnd === nextFilter.compareEnd
      ) {
        return curr;
      }
      return nextFilter;
    });
  }, [searchParams, now]);

  // ── Table state writers (reset page on anything but page/per_page) ──
  const setPage = (page: number) => updateParams({ page: page > 1 ? String(page) : null });
  const setPerPage = (perPage: number) => updateParams({ per_page: perPage !== DEFAULT_PER_PAGE ? String(perPage) : null, page: null });
  const setSort = (sort: string, order: "asc" | "desc") => updateParams({ sort, order, page: null });
  const setSearch = (q: string) => updateParams({ q: q || null, page: null });
  const setFilter = (key: string, value: string) => updateParams({ [`filter_${key}`]: value || null, page: null });
  const resetTable = () => updateParams({ q: null, sort: null, order: null, page: null, filter_min_pct: null });

  const debouncedQuery = useDebounce(query, 300);

  // ── Realtime SSE subscription ───────────────────────────────
  useReportsRealtime();

  // ── Data ─────────────────────────────────────────────────────
  const { hasFeature, isLoading: entLoading } = useEntitlements();
  const canViewReports = hasFeature("reports_export");
  const canBudget = hasFeature("advanced_budgeting");
  const reportsEnabled = !entLoading && canViewReports;

  // Prefetch common periods and deep-dive tabs so switching is instant
  React.useEffect(() => {
    if (!reportsEnabled) return;
    const common: PresetKey[] = ["this_month", "last_month", "3months", "this_year", "today", "this_week"];
    const now2 = new Date();
    for (const preset of common) {
      const d = computePresetDates(preset, now2);
      if (d.start && d.end) {
        const qKey = `${reportType}|${d.start}|${d.end}||`;
        queryClient.prefetchQuery({
          queryKey: [...queryKeys.reports.byPeriod("current_month"), qKey],
          queryFn: async () => {
            const res = await api.get(`/reports?type=${reportType}&start_date=${d.start}&end_date=${d.end}`);
            return normalizeReport(res.data?.data);
          },
          staleTime: staleTimeConfig.reports,
        });

        const qStr = `type=${reportType}&start_date=${d.start}&end_date=${d.end}`;
        queryClient.prefetchQuery({
          queryKey: queryKeys.reports.analytics(qStr),
          queryFn: async () => {
            const res = await api.get(`/reports/analytics?${qStr}`);
            return res.data?.data;
          },
          staleTime: staleTimeConfig.reports,
        });

        if (canBudget) {
          queryClient.prefetchQuery({
            queryKey: [...queryKeys.budgets.all, `?start_date=${d.start}&end_date=${d.end}`],
            queryFn: async () => {
              const res = await api.get(`/budgets/status?start_date=${d.start}&end_date=${d.end}`);
              return res.data?.data ?? [];
            },
            staleTime: staleTimeConfig.dashboard,
          });
        }
      }
    }

    queryClient.prefetchQuery({
      queryKey: queryKeys.spk.latest,
      queryFn: async () => {
        const res = await api.get("/spk/latest");
        return mapSpk(res.data?.data);
      },
      staleTime: staleTimeConfig.spk,
    });
    queryClient.prefetchQuery({
      queryKey: queryKeys.spk.history,
      queryFn: async () => {
        const res = await api.get("/spk/history?limit=12");
        return mapSpkHistory(res.data?.data);
      },
      staleTime: staleTimeConfig.spk,
    });
  }, [reportsEnabled, reportType, canBudget, queryClient]);

  const reportParams = dateFilter.startDate && dateFilter.endDate
    ? { type: reportType, startDate: dateFilter.startDate, endDate: dateFilter.endDate }
    : undefined;

  const { data: report, isLoading } = useReports(reportParams, { enabled: reportsEnabled });
  const { data: catChart } = useCategoryChart(dateFilter.startDate, dateFilter.endDate, { enabled: reportsEnabled });
  const { data: analytics, isLoading: analyticsLoading } = useReportAnalytics(
    { type: reportType, startDate: dateFilter.startDate, endDate: dateFilter.endDate },
    { enabled: reportsEnabled }
  );
  const exportReport = useExportReport();
  const [exporting, setExporting] = React.useState<string | null>(null);

  const handleExport = async (format: "csv" | "xlsx" | "pdf") => {
    setExporting(format);
    try {
      await exportReport(format, { type: reportType, startDate: dateFilter.startDate, endDate: dateFilter.endDate });
      toast.success(`Laporan ${format.toUpperCase()} diunduh`);
    } catch {
      toast.error("Gagal mengunduh laporan");
    } finally {
      setExporting(null);
    }
  };

  // Client-side percentage filter is applied here (the generic filter is exact-match).
  const minPct = Number(query.filters.min_pct) || 0;
  const categories: CategoryReport[] = React.useMemo(() => {
    const list = report?.categories ?? [];
    return minPct > 0 ? list.filter((c) => c.percentage >= minPct) : list;
  }, [report?.categories, minPct]);

  const typeName = typeLabel(reportType);
  const reportCurrency = report?.currency_code ?? "IDR";

  // ── Paywall ──────────────────────────────────────────────────
  if (entLoading) {
    return (
      <div className="mx-auto grid max-w-7xl grid-cols-1 gap-4 md:grid-cols-3">
        {[1, 2, 3].map((i) => <Skeleton key={i} className="h-24 border-2 rounded-md border-foreground" />)}
      </div>
    );
  }
  if (!canViewReports) {
    return (
      <div className="mx-auto max-w-7xl">
        <EmptyState
          title="Laporan tersedia di paket Pro"
          description="Analisis laporan, distribusi kategori, tren, skor SPK, dan ekspor tersedia mulai paket Pro. Tingkatkan paket untuk membuka halaman ini."
          action={
            <Button asChild className="cursor-pointer">
              <Link href="/user/subscription">
                <Sparkles className="w-4 h-4 mr-2" /> Tingkatkan Paket
              </Link>
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <div className="mx-auto flex min-h-full w-full max-w-7xl flex-col gap-6">
      {/* Header: title, type switch, export */}
      <div className="flex flex-col justify-between gap-4 lg:flex-row lg:items-center">
        <div>
          <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Laporan Keuangan</h2>
          <p className="mt-1 text-xs font-bold text-muted-foreground">
            Analisis mendalam, skor SPK, dan ekspor data Anda.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {/* Report type switch (URL-backed) */}
          <div className="flex border-2 rounded-md border-foreground bg-background p-1">
            {REPORT_TYPES.map((t) => (
              <button
                key={t.id}
                type="button"
                onClick={() => setType(t.id)}
                aria-pressed={reportType === t.id}
                className={`cursor-pointer px-3 py-1.5 text-xs font-bold transition-colors ${
                  reportType === t.id
                    ? "border-x-2 border-foreground bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button className="cursor-pointer" disabled={exporting !== null}>
                <Download className="w-4 h-4 mr-2" />
                {exporting ? `Mengunduh ${exporting.toUpperCase()}…` : "Ekspor"}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
              <DropdownMenuItem className="cursor-pointer font-bold" onSelect={() => handleExport("pdf")}>
                <FileText className="mr-2 h-4 w-4" /> PDF
              </DropdownMenuItem>
              <DropdownMenuItem className="cursor-pointer font-bold" onSelect={() => handleExport("xlsx")}>
                <FileSpreadsheet className="mr-2 h-4 w-4" /> Excel (XLSX)
              </DropdownMenuItem>
              <DropdownMenuItem className="cursor-pointer font-bold" onSelect={() => handleExport("csv")}>
                <FileType className="mr-2 h-4 w-4" /> CSV
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Date range */}
      <div className="border-2 rounded-lg border-foreground bg-card p-4 shadow-[4px_4px_0px_0px] shadow-foreground">
        <DateRangeFilter value={dateFilter} onChange={setDateFilter} />
      </div>

      {/* Tabs (URL-backed) */}
      <Tabs value={tab} onValueChange={setTab} className="gap-6">
        <TabsList className="flex w-full flex-wrap">
          {REPORT_TABS.map((t) => (
            <TabsTrigger key={t.id} value={t.id} className="cursor-pointer font-bold">
              {t.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="summary">
          <ReportsSummaryTab analytics={analytics} isLoading={analyticsLoading} type={reportType} />
        </TabsContent>

        <TabsContent value="spk">
          <ReportsSpkTab />
        </TabsContent>

        <TabsContent value="category">
          <div className="space-y-6">
            {/* Summary cards with trend-aware colouring */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-4">
              {[
                { label: "Total Pemasukan", value: report?.summary.total_income ?? 0, cls: "text-green-700" },
                { label: "Total Pengeluaran", value: report?.summary.total_expense ?? 0, cls: "text-red-600" },
                { label: "Total Tabungan", value: report?.summary.total_savings ?? 0, cls: "text-amber-600" },
                { label: "Surplus", value: report?.summary.surplus ?? 0, cls: "text-primary" },
              ].map((c) => (
                <div key={c.label} className="flex flex-col gap-2 border-2 rounded-lg border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
                  <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">{c.label}</span>
                  {isLoading ? <Skeleton className="h-6 w-24" /> : (
                    <span className={`font-mono text-xl font-black ${c.cls}`}>{formatIDR(c.value, reportCurrency)}</span>
                  )}
                </div>
              ))}
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <div className="flex h-[400px] flex-col border-2 rounded-lg border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
                <h3 className="mb-5 text-xs font-black uppercase tracking-wide">Distribusi {typeName} per Kategori</h3>
                <div className="relative flex-1 w-full">
                  <PieChartView raw currencyCode={reportCurrency} data={categories.map((c) => ({ name: c.category_name, value: c.total }))} />
                </div>
              </div>
              <div className="flex h-[400px] flex-col border-2 rounded-lg border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
                <h3 className="mb-5 text-xs font-black uppercase tracking-wide">Perbandingan Bulanan</h3>
                <div className="relative flex-1 w-full">
                  <BarChartView raw currencyCode={reportCurrency} data={(report?.monthly ?? []).map((m: MonthlyComparison) => ({ name: m.month, income: m.income, expense: m.expense, savings: m.savings }))} />
                </div>
              </div>
            </div>

            <CategoryTable
              categories={categories}
              type={reportType}
              currencyCode={reportCurrency}
              query={debouncedQuery}
              onPageChange={setPage}
              onPerPageChange={setPerPage}
              onSort={setSort}
              onSearch={setSearch}
              onFilter={setFilter}
              onReset={resetTable}
            />
          </div>
        </TabsContent>

        <TabsContent value="trend">
          <div className="space-y-6">
            <TopMoversPanel movers={analytics?.top_movers} />
            {catChart?.labels?.length && catChart?.series?.length ? (
              <div className="flex h-[420px] flex-col border-2 rounded-lg border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
                <h3 className="mb-5 text-xs font-black uppercase tracking-wide">Tren Kategori</h3>
                <div className="min-h-0 flex-1 w-full">
                  <MultiSeriesChart labels={catChart.labels} series={catChart.series} type="area" display />
                </div>
              </div>
            ) : null}
            <AnomalyPanel anomalies={analytics?.anomalies ?? []} />
          </div>
        </TabsContent>

        <TabsContent value="budget">
          <BudgetPanel
            period={dateFilter.startDate && dateFilter.endDate ? { startDate: dateFilter.startDate, endDate: dateFilter.endDate } : undefined}
            enabled={canBudget}
          />
        </TabsContent>

        <TabsContent value="calendar">
          <CalendarHeatmap daily={analytics?.daily} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
