"use client";

import * as React from "react";
import {
  Bar,
  BarChart,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  CartesianGrid,
  Legend,
} from "recharts";
import { BarChart3, LineChart as LineChartIcon, Loader2 } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorState } from "@/components/shared/ErrorState";
import { useUsageTimeseries, type UsageSeries } from "@/hooks/use-usage";

/** The chart range presets exposed in the UI. */
export type UsageRange = "24h" | "7d" | "30d" | "3m" | "1y" | "custom";

const RANGE_LABELS: { key: UsageRange; label: string }[] = [
  { key: "24h", label: "24 Jam" },
  { key: "7d", label: "7 Hari" },
  { key: "30d", label: "1 Bulan" },
  { key: "3m", label: "3 Bulan" },
  { key: "1y", label: "1 Tahun" },
  { key: "custom", label: "Kustom" },
];

/** Fixed, accessible palette per series key (stable across renders). */
const SERIES_COLORS: Record<string, string> = {
  transaction: "#2563eb",
  ai_token: "#7c3aed",
  ai_request: "#f59e0b",
  ocr: "#10b981",
  voice: "#f43f5e",
};

function colorFor(key: string, i: number): string {
  const fallback = ["#2563eb", "#7c3aed", "#f59e0b", "#10b981", "#f43f5e"];
  return SERIES_COLORS[key] ?? fallback[i % fallback.length];
}

function formatNumber(v: number): string {
  return new Intl.NumberFormat("id-ID").format(v);
}

function UsageTooltip({
  active,
  payload,
  label,
}: {
  active?: boolean;
  payload?: Array<{ value: number; color: string; name: string }>;
  label?: string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="space-y-1 rounded-md border-2 border-foreground bg-card p-3 text-xs font-bold shadow-[4px_4px_0px_0px] shadow-foreground">
      <p className="mb-1 text-muted-foreground">{label}</p>
      {payload.map((p, i) => (
        <div key={i} className="flex items-center gap-2">
          <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: p.color }} />
          <span className="truncate">{p.name}:</span>
          <span className="font-mono">{formatNumber(p.value)}</span>
        </div>
      ))}
    </div>
  );
}

interface UsageChartProps {
  range: UsageRange;
  start?: string;
  end?: string;
  onRangeChange: (range: UsageRange) => void;
  onCustomChange: (start: string, end: string) => void;
}

/**
 * Usage analytics chart: a bar/line toggle over the shared per-kind series from
 * `GET /me/usage/timeseries`, with a range filter (24h … 1y) and a custom
 * date window.
 */
export function UsageChart({ range, start, end, onRangeChange, onCustomChange }: UsageChartProps) {
  const [chartType, setChartType] = React.useState<"bar" | "line">("bar");
  const { timeseries, isLoading, isFetching, isError } = useUsageTimeseries(range, {
    start,
    end,
    enabled: range !== "custom" || Boolean(start && end),
  });

  const chartData = React.useMemo(() => {
    if (!timeseries) return [];
    const labels = Array.isArray(timeseries.labels) ? timeseries.labels : [];
    const series = (Array.isArray(timeseries.series) ? timeseries.series : []).filter(
      (s): s is UsageSeries => !!s && typeof s.key === "string" && Array.isArray(s.data)
    );
    return labels.map((label, i) => {
      const point: Record<string, number | string> = { name: label };
      for (const s of series) point[s.label] = s.data[i] || 0;
      return point;
    });
  }, [timeseries]);

  const series = (timeseries?.series ?? []).filter((s) => Array.isArray(s.data));

  const renderChart = () => {
    if (isLoading) {
      return <Skeleton className="h-full w-full rounded-md border-2 border-foreground" />;
    }
    if (isError) {
      return <ErrorState description="Gagal memuat data penggunaan." />;
    }
    if (chartData.length === 0 || series.length === 0) {
      return (
        <EmptyState
          title="Belum ada data"
          description="Belum ada aktivitas pemakaian pada rentang waktu ini."
        />
      );
    }

    const commonAxis = (
      <>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
        <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={8} />
        <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} width={40} />
        <Tooltip cursor={{ fill: "#f1f5f9" }} content={<UsageTooltip />} />
        <Legend wrapperStyle={{ fontSize: 10, fontWeight: 700 }} iconType="circle" />
      </>
    );

    if (chartType === "line") {
      return (
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
            {commonAxis}
            {series.map((s, i) => (
              <Line
                key={s.key}
                type="monotone"
                dataKey={s.label}
                name={s.label}
                stroke={colorFor(s.key, i)}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4 }}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      );
    }

    return (
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
          {commonAxis}
          {series.map((s, i) => (
            <Bar key={s.key} dataKey={s.label} name={s.label} fill={colorFor(s.key, i)} />
          ))}
        </BarChart>
      </ResponsiveContainer>
    );
  };

  return (
    <div className="flex h-[420px] flex-col rounded-lg border-2 border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h3 className="flex items-center gap-2 text-xs font-black uppercase tracking-wide text-foreground">
            Grafik Pemakaian
            {isFetching && !isLoading && <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" aria-hidden="true" />}
          </h3>
          <p className="mt-0.5 text-[11px] font-bold text-muted-foreground">
            Jumlah pemakaian per kategori pada rentang waktu terpilih.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex rounded-md border-2 border-foreground bg-background p-0.5">
            <button
              type="button"
              onClick={() => setChartType("bar")}
              aria-pressed={chartType === "bar"}
              title="Grafik batang"
              className={`cursor-pointer px-2 py-1 text-xs font-bold uppercase transition-all ${
                chartType === "bar"
                  ? "border-x-2 border-foreground bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <BarChart3 className="h-4 w-4" aria-hidden="true" />
              <span className="sr-only">Batang</span>
            </button>
            <button
              type="button"
              onClick={() => setChartType("line")}
              aria-pressed={chartType === "line"}
              title="Grafik garis"
              className={`cursor-pointer px-2 py-1 text-xs font-bold uppercase transition-all ${
                chartType === "line"
                  ? "border-x-2 border-foreground bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <LineChartIcon className="h-4 w-4" aria-hidden="true" />
              <span className="sr-only">Garis</span>
            </button>
          </div>
        </div>
      </div>

      {/* Range presets */}
      <div className="-mb-1 flex flex-nowrap items-center gap-1 overflow-x-auto pb-1">
        {RANGE_LABELS.map((r) => (
          <button
            key={r.key}
            type="button"
            onClick={() => onRangeChange(r.key)}
            aria-pressed={range === r.key}
            className={`shrink-0 cursor-pointer whitespace-nowrap rounded-md border-2 px-3 py-1.5 text-xs font-bold uppercase transition-all ${
              range === r.key
                ? "border-foreground bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                : "border-transparent bg-background text-muted-foreground hover:border-foreground hover:text-foreground"
            }`}
          >
            {r.label}
          </button>
        ))}
      </div>

      {range === "custom" && (
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2">
            <label htmlFor="usage-from" className="text-xs font-bold text-muted-foreground">Dari</label>
            <input
              id="usage-from"
              type="date"
              value={start ?? ""}
              onChange={(e) => onCustomChange(e.target.value, end ?? "")}
              className="rounded-md border-2 border-foreground bg-background px-2 py-1 text-xs font-bold focus:outline-none"
            />
          </div>
          <div className="flex items-center gap-2">
            <label htmlFor="usage-to" className="text-xs font-bold text-muted-foreground">Ke</label>
            <input
              id="usage-to"
              type="date"
              value={end ?? ""}
              onChange={(e) => onCustomChange(start ?? "", e.target.value)}
              className="rounded-md border-2 border-foreground bg-background px-2 py-1 text-xs font-bold focus:outline-none"
            />
          </div>
        </div>
      )}

      <div className="relative mt-3 min-h-0 min-w-0 flex-1">{renderChart()}</div>
    </div>
  );
}
