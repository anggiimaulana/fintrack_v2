"use client";

import * as React from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
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
import { DateRangeFilter, type DateRangeValue } from "@/components/shared/DateRangeFilter";
import { useReports, type CategoryReport, type MonthlyComparison } from "@/hooks/useReports";
import { useCategoryChart } from "@/hooks/useDashboard";
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
  typeLabel,
  type ReportTypeId,
} from "@/lib/report-tabs";
import type { TableQuery } from "@/lib/client-table";
import { toast } from "sonner";

// Charts pull in recharts (~500 KB); keep them code-split and below the fold.
const chartFallback = <Skeleton className="h-full w-full border-2 rounded-md border-foreground" />;
const PieChartView = dynamic(() => import("@/components/charts/PieChartView").then((m) => m.PieChartView), { ssr: false, loading: () => chartFallback });
const BarChartView = dynamic(() => import("@/components/charts/BarChartView").then((m) => m.BarChartView), { ssr: false, loading: () => chartFallback });
const MultiSeriesChart = dynamic(() => import("@/components/charts/MultiSeriesChart").then((m) => m.MultiSeriesChart), { ssr: false, loading: () => chartFallback });

function formatIDR(value: number): string {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
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
  const now = React.useMemo(() => new Date(), []);
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;

  // ── URL-backed state ─────────────────────────────────────────
  // tab / type / date range / table state all live in the query string so the
  // page is bookmarkable and every interaction is a shallow replace.
  const tab = parseTab(searchParams.get("tab"));
  const reportType = parseReportType(searchParams.get("type"));

  const [dateFilter, setDateFilterState] = React.useState<DateRangeValue>(() => ({
    preset: "this_month",
    startDate: `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-01`,
    endDate: today,
    compareStart: "",
    compareEnd: "",
    enableCompare: false,
  }));

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

  /** Applies a shallow URL update, preserving unrelated params. */
  const updateParams = React.useCallback(
    (patch: Record<string, string | null>) => {
      const p = new URLSearchParams(searchParams.toString());
      for (const [k, v] of Object.entries(patch)) {
        if (v === null || v === "") p.delete(k);
        else p.set(k, v);
      }
      const qs = p.toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    },
    [router, pathname, searchParams]
  );

  const setTab = (t: string) => updateParams({ tab: t });
  const setType = (t: ReportTypeId) => updateParams({ type: t, page: null });

  // ── Table state writers (reset page on anything but page/per_page) ──
  const setPage = (page: number) => updateParams({ page: page > 1 ? String(page) : null });
  const setPerPage = (perPage: number) => updateParams({ per_page: perPage !== DEFAULT_PER_PAGE ? String(perPage) : null, page: null });
  const setSort = (sort: string, order: "asc" | "desc") => updateParams({ sort, order, page: null });
  const setSearch = (q: string) => updateParams({ q: q || null, page: null });
  const setFilter = (key: string, value: string) => updateParams({ [`filter_${key}`]: value || null, page: null });
  const resetTable = () => updateParams({ q: null, sort: null, order: null, page: null, filter_min_pct: null });

  // ── Data ─────────────────────────────────────────────────────
  const { hasFeature, isLoading: entLoading } = useEntitlements();
  const canViewReports = hasFeature("reports_export");
  const canBudget = hasFeature("advanced_budgeting");
  const reportsEnabled = !entLoading && canViewReports;

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
    <div className="mx-auto flex h-full w-full max-w-7xl flex-col gap-6">
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
        <DateRangeFilter value={dateFilter} onChange={setDateFilterState} />
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

        <TabsContent value="ringkasan">
          <ReportsSummaryTab analytics={analytics} isLoading={analyticsLoading} type={reportType} />
        </TabsContent>

        <TabsContent value="spk">
          <ReportsSpkTab />
        </TabsContent>

        <TabsContent value="kategori">
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
                    <span className={`font-mono text-xl font-black ${c.cls}`}>{formatIDR(c.value)}</span>
                  )}
                </div>
              ))}
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <div className="flex h-[400px] flex-col border-2 rounded-lg border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
                <h3 className="mb-5 text-xs font-black uppercase tracking-wide">Distribusi {typeName} per Kategori</h3>
                <div className="relative flex-1 w-full">
                  <PieChartView data={categories.map((c) => ({ name: c.category_name, value: c.total }))} />
                </div>
              </div>
              <div className="flex h-[400px] flex-col border-2 rounded-lg border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
                <h3 className="mb-5 text-xs font-black uppercase tracking-wide">Perbandingan Bulanan</h3>
                <div className="relative flex-1 w-full">
                  <BarChartView data={(report?.monthly ?? []).map((m: MonthlyComparison) => ({ name: m.month, income: m.income, expense: m.expense, savings: m.savings }))} />
                </div>
              </div>
            </div>

            <CategoryTable
              categories={categories}
              type={reportType}
              query={query}
              onPageChange={setPage}
              onPerPageChange={setPerPage}
              onSort={setSort}
              onSearch={setSearch}
              onFilter={setFilter}
              onReset={resetTable}
            />
          </div>
        </TabsContent>

        <TabsContent value="tren">
          <div className="space-y-6">
            <TopMoversPanel movers={analytics?.top_movers} />
            {catChart?.labels?.length && catChart?.series?.length ? (
              <div className="flex h-[420px] flex-col border-2 rounded-lg border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
                <h3 className="mb-5 text-xs font-black uppercase tracking-wide">Tren Kategori</h3>
                <div className="min-h-0 flex-1 w-full">
                  <MultiSeriesChart labels={catChart.labels} series={catChart.series} type="area" />
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

        <TabsContent value="kalender">
          <CalendarHeatmap daily={analytics?.daily} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
