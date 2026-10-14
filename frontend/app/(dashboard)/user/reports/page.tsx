"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
import { Download, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/shared/EmptyState";
import { useReports, type CategoryReport, type MonthlyComparison } from "@/hooks/useReports";
import { useCategoryChart } from "@/hooks/useDashboard";
import { useEntitlements } from "@/hooks/use-entitlements";
import { DateRangeFilter, type DateRangeValue } from "@/components/shared/DateRangeFilter";

/**
 * Charts pull in recharts (~500 KB). They are client-only and below the fold,
 * so they are code-split and rendered after the page shell paints — the summary
 * numbers show immediately instead of waiting on the chart bundle.
 */
const chartFallback = <Skeleton className="w-full h-full border-2 border-foreground" />;
const PieChartView = dynamic(() => import("@/components/charts/PieChartView").then((m) => m.PieChartView), { ssr: false, loading: () => chartFallback });
const BarChartView = dynamic(() => import("@/components/charts/BarChartView").then((m) => m.BarChartView), { ssr: false, loading: () => chartFallback });
const MultiSeriesChart = dynamic(() => import("@/components/charts/MultiSeriesChart").then((m) => m.MultiSeriesChart), { ssr: false, loading: () => chartFallback });

function formatIDR(value: number) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
}

const REPORT_TABS = [
  { key: "expense", label: "Pengeluaran" },
  { key: "income", label: "Pemasukan" },
  { key: "savings", label: "Tabungan" },
];

export default function ReportsPage() {
  const now = useMemo(() => new Date(), []);
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;

  const [dateFilter, setDateFilter] = useState<DateRangeValue>({
    preset: "this_month",
    startDate: `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-01`,
    endDate: today,
    compareStart: "",
    compareEnd: "",
    enableCompare: false,
  });
  const [reportType, setReportType] = useState<string>("expense");
  const [catChartType, setCatChartType] = useState<"line" | "area">("line");

  // `GET /reports` is gated by the `reports_export` plan feature (Pro/Business).
  // Resolve the plan first so a Free account renders the upgrade CTA instead of
  // firing a request that always 403s.
  const { hasFeature, isLoading: entLoading } = useEntitlements();
  const canViewReports = hasFeature("reports_export");
  const reportsEnabled = !entLoading && canViewReports;

  const reportParams = dateFilter.startDate && dateFilter.endDate
    ? {
        type: reportType,
        startDate: dateFilter.startDate,
        endDate: dateFilter.endDate,
        compareStart: dateFilter.enableCompare && dateFilter.compareStart ? dateFilter.compareStart : undefined,
        compareEnd: dateFilter.enableCompare && dateFilter.compareEnd ? dateFilter.compareEnd : undefined,
      }
    : undefined;

  const { data: report, isLoading } = useReports(reportParams, { enabled: reportsEnabled });
  const { data: catChart } = useCategoryChart(dateFilter.startDate, dateFilter.endDate, { enabled: reportsEnabled });

  const typeLabel = reportType === "income" ? "Pemasukan" : reportType === "savings" ? "Tabungan" : "Pengeluaran";

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto h-full">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Laporan Keuangan</h2>
          <p className="text-xs font-bold text-muted-foreground mt-1">Analisis mendalam mengenai pengeluaran dan pemasukan Anda.</p>
        </div>
        <Button
          className="cursor-pointer disabled:cursor-not-allowed"
          disabled={!canViewReports}
          title={canViewReports ? undefined : "Ekspor laporan membutuhkan paket Pro atau lebih tinggi."}
        >
          <Download className="w-4 h-4 mr-2" />
          Export PDF
        </Button>
      </div>

      {entLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-24 border-2 border-foreground" />
          ))}
        </div>
      ) : !canViewReports ? (
        <EmptyState
          title="Laporan tersedia di paket Pro"
          description="Analisis laporan, distribusi kategori, dan tren bulanan tersedia mulai paket Pro. Tingkatkan paket untuk membuka halaman ini."
          action={
            <Button asChild className="cursor-pointer">
              <Link href="/user/settings/subscription">
                <Sparkles className="w-4 h-4 mr-2" />
                Tingkatkan Paket
              </Link>
            </Button>
          }
        />
      ) : (
        <>
          <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4">
            <DateRangeFilter value={dateFilter} onChange={setDateFilter} />
          </div>

          <div className="flex border-2 border-foreground p-1 bg-background">
            {REPORT_TABS.map((f) => (
              <button
                key={f.key}
                onClick={() => setReportType(f.key)}
                className={`flex-1 px-4 py-1.5 text-xs font-bold cursor-pointer transition-colors ${
                  reportType === f.key
                    ? "bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground border-x-2 border-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          {isLoading ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
              {[1, 2, 3, 4].map((i) => (
                <Skeleton key={i} className="h-24 border-2 border-foreground" />
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
              <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col gap-2">
                <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Total Pemasukan</span>
                <span className="text-xl font-black font-mono text-green-700">{report ? formatIDR(report.summary.total_income) : "Rp 0"}</span>
              </div>
              <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col gap-2">
                <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Total Pengeluaran</span>
                <span className="text-xl font-black font-mono text-red-600">{report ? formatIDR(report.summary.total_expense) : "Rp 0"}</span>
              </div>
              <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col gap-2">
                <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Total Tabungan</span>
                <span className="text-xl font-black font-mono text-amber-600">{report ? formatIDR(report.summary.total_savings) : "Rp 0"}</span>
              </div>
              <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col gap-2">
                <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Surplus</span>
                <span className="text-xl font-black font-mono text-primary">{report ? formatIDR(report.summary.surplus) : "Rp 0"}</span>
              </div>
            </div>
          )}

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 h-[400px] flex flex-col">
              <h3 className="text-xs font-black uppercase tracking-wide text-foreground mb-5">Distribusi {typeLabel} per Kategori</h3>
              <div className="flex-1 w-full relative">
                <PieChartView
                  data={report?.categories.map((c: CategoryReport) => ({ name: c.category_name, value: c.total }))}
                />
              </div>
            </div>
            <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 h-[400px] flex flex-col">
              <h3 className="text-xs font-black uppercase tracking-wide text-foreground mb-5">Perbandingan Bulanan</h3>
              <div className="flex-1 w-full relative">
                <BarChartView
                  data={report?.monthly.map((m: MonthlyComparison) => ({ name: m.month, income: m.income, expense: m.expense, savings: m.savings }))}
                />
              </div>
            </div>
          </div>

          {catChart?.labels?.length && catChart?.series?.length ? (
            <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 h-[400px] flex flex-col">
              <div className="flex items-center justify-between mb-5">
                <h3 className="text-xs font-black uppercase tracking-wide text-foreground">Tren Kategori</h3>
                <div className="flex border-2 border-foreground bg-background p-0.5">
                  {(["line", "area"] as const).map((t) => (
                    <button
                      key={t}
                      onClick={() => setCatChartType(t)}
                      className={`px-3 py-1 text-[10px] font-bold uppercase cursor-pointer ${
                        catChartType === t
                          ? "bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground border-x-2 border-foreground"
                          : "text-muted-foreground hover:text-foreground"
                      }`}
                    >
                      {t === "line" ? "Garis" : "Area"}
                    </button>
                  ))}
                </div>
              </div>
              <div className="flex-1 w-full relative min-h-0" style={{ minHeight: "250px" }}>
                <MultiSeriesChart labels={catChart.labels} series={catChart.series} type={catChartType} />
              </div>
            </div>
          ) : null}

          <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground overflow-hidden">
            <div className="px-5 py-4 border-b-2 border-foreground">
              <h3 className="text-xs font-black uppercase tracking-wide text-foreground">Detail per Kategori</h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted text-muted-foreground font-black border-b-2 border-foreground text-xs uppercase tracking-wide">
                  <tr>
                    <th className="px-6 py-4 text-left">Kategori</th>
                    <th className="px-6 py-4 text-left">Total</th>
                    <th className="px-6 py-4 text-left">% dari {typeLabel}</th>
                    <th className="px-6 py-4 text-right">Trend vs Periode Sebelumnya</th>
                  </tr>
                </thead>
                <tbody className="divide-y-2 divide-foreground">
                  {report && report.categories.length > 0 ? report.categories.map((cat: CategoryReport) => (
                    <tr key={cat.category_name} className="hover:bg-muted">
                      <td className="px-6 py-4 font-bold text-foreground">{cat.category_name}</td>
                      <td className="px-6 py-4 font-mono font-bold">{formatIDR(cat.total)}</td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-3">
                          <div className="w-24 h-2 bg-muted-foreground/20">
                            <div className="h-full bg-primary" style={{ width: `${Math.min(cat.percentage, 100)}%` }}></div>
                          </div>
                          <span className="font-bold">{cat.percentage}%</span>
                        </div>
                      </td>
                      <td className={`px-6 py-4 text-right font-bold ${cat.trend <= 0 ? "text-green-700" : "text-red-700"}`}>
                        {cat.trend === 0 ? "-" : `${cat.trend > 0 ? "Naik" : "Turun"} ${Math.abs(cat.trend).toFixed(1)}%`}
                      </td>
                    </tr>
                  )) : (
                    <tr>
                      <td colSpan={4} className="px-6 py-8 text-center text-muted-foreground font-bold">
                        Belum ada data transaksi untuk periode ini
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
