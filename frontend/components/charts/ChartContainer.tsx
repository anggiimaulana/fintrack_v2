"use client";

import { useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { LineChartView } from "./LineChartView";
import { BarChartView } from "./BarChartView";
import { PieChartView } from "./PieChartView";
import { AreaChartView } from "./AreaChartView";
import { MultiSeriesChart } from "./MultiSeriesChart";
import { useDashboardCharts, useCategoryChart, type ChartDataPoint } from "@/hooks/useDashboard";

export type ChartType = "line" | "bar" | "pie" | "area";
export type PeriodType = "week" | "month" | "3months" | "6months" | "year";

const CHART_TYPES: ChartType[] = ["line", "bar", "pie", "area"];

const periodToApi: Record<PeriodType, string> = {
  week: "week",
  month: "month",
  "3months": "3month",
  "6months": "6month",
  year: "year",
};

export function ChartContainerSkeleton() {
  return (
    <div className="border-2 rounded-lg border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col h-[420px]">
      <div className="flex justify-between items-center mb-5">
        <Skeleton className="h-5 w-32 border-2 rounded-md border-foreground" />
        <div className="flex gap-2">
          {CHART_TYPES.map((t) => (
            <Skeleton key={t} className="h-8 w-12 border-2 rounded-md border-foreground" />
          ))}
        </div>
      </div>
      <Skeleton className="flex-1 w-full border-2 rounded-md border-foreground" />
    </div>
  );
}

interface ChartContainerProps {
  dateRange?: { startDate: string; endDate: string };
}

export function ChartContainer({ dateRange }: ChartContainerProps) {
  const [chartType, setChartType] = useState<ChartType>("line");
  const [period, setPeriod] = useState<PeriodType>("month");
  const [groupBy, setGroupBy] = useState<"summary" | "category">("summary");

  const chartParams = dateRange
    ? { startDate: dateRange.startDate, endDate: dateRange.endDate }
    : { period: periodToApi[period] };

  const { data: chartData, isLoading } = useDashboardCharts(chartParams);

  const categoryRange = dateRange || (() => {
    const now = new Date();
    const end = now;
    let start: Date;
    switch (period) {
      case "week": start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - now.getDay()); break;
      case "month": start = new Date(now.getFullYear(), now.getMonth(), 1); break;
      case "3months": start = new Date(now.getFullYear(), now.getMonth() - 2, 1); break;
      case "6months": start = new Date(now.getFullYear(), now.getMonth() - 5, 1); break;
      case "year": start = new Date(now.getFullYear(), 0, 1); break;
      default: start = new Date(now.getFullYear(), now.getMonth(), 1);
    }
    const fmt = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
    return { startDate: fmt(start), endDate: fmt(end) };
  })();
  const { data: catChartData, isLoading: catLoading } = useCategoryChart(categoryRange.startDate, categoryRange.endDate);

  const periodLabels: Record<PeriodType, string> = {
    week: "Minggu ini",
    month: "Bulan ini",
    "3months": "3 Bulan",
    "6months": "6 Bulan",
    year: "Tahun ini",
  };

  const renderChart = () => {
    const needsCat = (chartType === "line" || chartType === "area") && groupBy === "category";
    if (isLoading || (needsCat && catLoading)) {
      return (
        <div className="flex items-center justify-center h-full">
          <Skeleton className="w-full h-full border-2 rounded-md border-foreground" />
        </div>
      );
    }

    const isMulti = chartType === "line" || chartType === "area";

    if (isMulti && groupBy === "category" && catChartData?.labels?.length && catChartData?.series?.length) {
      return <MultiSeriesChart labels={catChartData.labels} series={catChartData.series} type={chartType} />;
    }

    if (isMulti && groupBy === "category") {
      return <div className="flex items-center justify-center h-full text-xs font-bold text-muted-foreground">Tidak ada data kategori</div>;
    }

    const data: ChartDataPoint[] = chartData || [];

    switch (chartType) {
      case "line":
        return <LineChartView data={data.length > 0 ? data : undefined} />;
      case "area":
        return <AreaChartView data={data.length > 0 ? data.map((d: ChartDataPoint) => ({ name: d.name, balance: d.income - d.expense })) : undefined} />;
      case "bar":
        return <BarChartView data={data.length > 0 ? data : undefined} />;
      case "pie": {
        const totalIncome = data.reduce((s: number, d: ChartDataPoint) => s + d.income, 0);
        const totalExpense = data.reduce((s: number, d: ChartDataPoint) => s + d.expense, 0);
        const totalSavings = data.reduce((s: number, d: ChartDataPoint) => s + d.savings, 0);
        const pieData = totalIncome > 0 || totalExpense > 0 || totalSavings > 0
          ? [
              { name: "Pemasukan", value: totalIncome },
              { name: "Pengeluaran", value: totalExpense },
              { name: "Tabungan", value: totalSavings },
            ]
          : undefined;
        return <PieChartView data={pieData} />;
      }
      default:
        return <LineChartView data={data.length > 0 ? data : undefined} />;
    }
  };

  return (
    <div className="border-2 rounded-lg border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col h-[420px]">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-5">
        <div>
          <h3 className="text-xs font-black uppercase tracking-wide text-foreground">Grafik Transaksi</h3>
          {!dateRange && (
            <div className="mt-1">
              <select
                className="text-xs border-2 rounded-md border-foreground bg-background font-bold text-muted-foreground focus:outline-none cursor-pointer px-2 py-0.5"
                value={period}
                onChange={(e) => setPeriod(e.target.value as PeriodType)}
              >
                {Object.entries(periodLabels).map(([key, label]) => (
                  <option key={key} value={key}>{label}</option>
                ))}
              </select>
            </div>
          )}
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {(chartType === "line" || chartType === "area") && (
            <div className="flex border-2 rounded-md border-foreground bg-background p-0.5">
              <button
                onClick={() => setGroupBy("summary")}
                className={`px-2 py-1 text-[10px] font-bold uppercase cursor-pointer transition-all ${
                  groupBy === "summary"
                    ? "bg-card text-foreground shadow-[1px_1px_0px_0px] shadow-foreground border-x-2 border-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                Ringkasan
              </button>
              <button
                onClick={() => setGroupBy("category")}
                className={`px-2 py-1 text-[10px] font-bold uppercase cursor-pointer transition-all ${
                  groupBy === "category"
                    ? "bg-card text-foreground shadow-[1px_1px_0px_0px] shadow-foreground border-x-2 border-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                Kategori
              </button>
            </div>
          )}
          <div className="flex border-2 rounded-md border-foreground bg-background p-0.5">
            {CHART_TYPES.map((type) => (
              <button
                key={type}
                onClick={() => { setChartType(type); if (type === "bar" || type === "pie") setGroupBy("summary"); }}
                className={`px-3 py-1 text-xs font-bold uppercase cursor-pointer transition-all ${
                  chartType === type
                    ? "bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground border-x-2 border-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {type === "line" ? "Garis" : type === "bar" ? "Batang" : type === "pie" ? "Lingkaran" : "Area"}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="flex-1 w-full relative min-w-0 min-h-0">
        {renderChart()}
      </div>
    </div>
  );
}
