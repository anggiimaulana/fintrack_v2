"use client";

import * as React from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts";
import { BarChart3, LineChart as LineChartIcon, AreaChart as AreaChartIcon, PieChart as PieChartIcon, Loader2 } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorState } from "@/components/shared/ErrorState";
import { useUsageTimeseries, type UsageSeries } from "@/hooks/use-usage";

/** The chart range presets exposed in the UI. */
export type UsageRange = "24h" | "7d" | "30d" | "3m" | "1y" | "custom";

/** The chart visualisations the user can switch between. */
export type UsageChartType = "bar" | "line" | "area" | "pie";

export const USAGE_CHART_TYPES: UsageChartType[] = ["bar", "line", "area", "pie"];

const RANGE_LABELS: { key: UsageRange; label: string }[] = [
  { key: "24h", label: "24 Jam" },
  { key: "7d", label: "7 Hari" },
  { key: "30d", label: "1 Bulan" },
  { key: "3m", label: "3 Bulan" },
  { key: "1y", label: "1 Tahun" },
  { key: "custom", label: "Kustom" },
];

const CHART_TYPE_META: Record<UsageChartType, { label: string; Icon: React.ComponentType<{ className?: string }> }> = {
  bar: { label: "Batang", Icon: BarChart3 },
  line: { label: "Garis", Icon: LineChartIcon },
  area: { label: "Area", Icon: AreaChartIcon },
  pie: { label: "Lingkaran", Icon: PieChartIcon },
};

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
      {label ? <p className="mb-1 text-muted-foreground">{label}</p> : null}
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
  chartType: UsageChartType;
  /** Series keys the user has hidden. */
  hiddenSeries: string[];
  onRangeChange: (range: UsageRange) => void;
  onCustomChange: (start: string, end: string) => void;
  onChartTypeChange: (type: UsageChartType) => void;
  onToggleSeries: (key: string) => void;
}

/**
 * Usage analytics chart over the shared per-kind series from
 * `GET /me/usage/timeseries`. Supports bar / line / area / pie views, a range
 * filter (24h … 1y) with a custom window, and per-series visibility toggles.
 * All of it is driven by the parent (and persisted in the URL).
 */
export function UsageChart({
  range,
  start,
  end,
  chartType,
  hiddenSeries,
  onRangeChange,
  onCustomChange,
  onChartTypeChange,
  onToggleSeries,
}: UsageChartProps) {
  const { timeseries, isLoading, isFetching, isError } = useUsageTimeseries(range, {
    start,
    end,
    enabled: range !== "custom" || Boolean(start && end),
  });

  const allSeries = React.useMemo(
    () =>
      (timeseries?.series ?? []).filter(
        (s): s is UsageSeries => !!s && typeof s.key === "string" && Array.isArray(s.data)
      ),
    [timeseries]
  );

  const visibleSeries = React.useMemo(
    () => allSeries.filter((s) => !hiddenSeries.includes(s.key)),
    [allSeries, hiddenSeries]
  );

  const chartData = React.useMemo(() => {
    const labels = Array.isArray(timeseries?.labels) ? timeseries.labels : [];
    return labels.map((label, i) => {
      const point: Record<string, number | string> = { name: label };
      for (const s of visibleSeries) point[s.label] = s.data[i] || 0;
      return point;
    });
  }, [timeseries, visibleSeries]);

  // Pie shows the TOTAL per series across the window, not per bucket.
  const pieData = React.useMemo(
    () =>
      visibleSeries.map((s, i) => ({
        name: s.label,
        value: s.data.reduce((sum, v) => sum + (v || 0), 0),
        color: colorFor(s.key, i),
      })),
    [visibleSeries]
  );

  const renderChart = () => {
    if (isLoading) {
      return <Skeleton className="h-full w-full rounded-md border-2 border-foreground" />;
    }
    if (isError) {
      return <ErrorState description="Gagal memuat data penggunaan." />;
    }
    if (visibleSeries.length === 0) {
      return (
        <EmptyState
          title="Tidak ada seri terpilih"
          description="Pilih minimal satu kategori untuk ditampilkan."
        />
      );
    }
    if (chartData.length === 0) {
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

    if (chartType === "pie") {
      const total = pieData.reduce((sum, d) => sum + d.value, 0);
      if (total === 0) {
        return (
          <EmptyState
            title="Belum ada data"
            description="Belum ada aktivitas pemakaian pada rentang waktu ini."
          />
        );
      }
      return (
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Tooltip content={<UsageTooltip />} />
            <Legend wrapperStyle={{ fontSize: 10, fontWeight: 700 }} iconType="circle" />
            <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius="75%" label={false}>
              {pieData.map((d) => (
                <Cell key={d.name} fill={d.color} />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>
      );
    }

    if (chartType === "line") {
      return (
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
            {commonAxis}
            {visibleSeries.map((s, i) => (
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

    if (chartType === "area") {
      return (
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
            {commonAxis}
            {visibleSeries.map((s, i) => (
              <Area
                key={s.key}
                type="monotone"
                dataKey={s.label}
                name={s.label}
                stroke={colorFor(s.key, i)}
                fill={colorFor(s.key, i)}
                fillOpacity={0.25}
                strokeWidth={2}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      );
    }

    return (
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
          {commonAxis}
          {visibleSeries.map((s, i) => (
            <Bar key={s.key} dataKey={s.label} name={s.label} fill={colorFor(s.key, i)} />
          ))}
        </BarChart>
      </ResponsiveContainer>
    );
  };

  return (
    <div className="flex h-[460px] flex-col rounded-lg border-2 border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
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

        <div className="flex rounded-md border-2 border-foreground bg-background p-0.5">
          {USAGE_CHART_TYPES.map((t) => {
            const { label, Icon } = CHART_TYPE_META[t];
            return (
              <button
                key={t}
                type="button"
                onClick={() => onChartTypeChange(t)}
                aria-pressed={chartType === t}
                aria-label={label}
                title={`Grafik ${label}`}
                className={`cursor-pointer px-2 py-1 text-xs font-bold uppercase transition-all ${
                  chartType === t
                    ? "border-x-2 border-foreground bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <Icon className="h-4 w-4" />
              </button>
            );
          })}
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

      {/* Per-series visibility toggles */}
      {allSeries.length > 0 && (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          {allSeries.map((s, i) => {
            const hidden = hiddenSeries.includes(s.key);
            return (
              <button
                key={s.key}
                type="button"
                onClick={() => onToggleSeries(s.key)}
                aria-pressed={!hidden}
                className={`flex cursor-pointer items-center gap-1.5 rounded-md border-2 px-2 py-1 text-[11px] font-bold transition-all ${
                  hidden
                    ? "border-transparent bg-background text-muted-foreground line-through hover:border-foreground"
                    : "border-foreground bg-card text-foreground"
                }`}
              >
                <span
                  className="h-2.5 w-2.5 shrink-0 rounded-full"
                  style={{ backgroundColor: hidden ? "#cbd5e1" : colorFor(s.key, i) }}
                />
                {s.label}
              </button>
            );
          })}
        </div>
      )}

      <div className="relative mt-3 min-h-0 min-w-0 flex-1">{renderChart()}</div>
    </div>
  );
}
