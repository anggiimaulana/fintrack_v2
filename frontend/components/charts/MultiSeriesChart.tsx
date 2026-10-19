"use client";

import { Bar, BarChart, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis, CartesianGrid, Area, AreaChart } from "recharts";
import { formatDisplay, formatMoney } from "@/lib/currency";
import { formatRupiah } from "@/lib/format";

interface MultiSeriesChartProps {
  labels: string[];
  series: Array<{ name: string; data: number[]; color: string }>;
  type: "bar" | "line" | "area";
  /** When true the values are IDR base and are shown in the user's currency. */
  display?: boolean;
  /** When true the values were already converted server-side (report data). */
  raw?: boolean;
  /** Currency code for `raw` mode (the report's currency). */
  currencyCode?: string | null;
}

type Fmt = (value: number) => string;

function CustomTooltip({ active, payload, label, fmt }: { active?: boolean, payload?: Array<{value: number, color: string, name: string}>, label?: string, fmt: Fmt }) {
  if (!active || !payload?.length) return null;
  const total = payload.reduce((sum: number, p: {value: number, color: string, name: string}) => sum + ((p.value as number) || 0), 0);
  return (
    <div className="bg-card border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground p-3 text-xs font-bold space-y-1 max-h-48 overflow-y-auto">
      <p className="text-muted-foreground mb-1">{label}</p>
      {payload.map((p: {value: number, color: string, name: string}, i: number) => (
        <div key={i} className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: p.color }} />
          <span className="truncate max-w-[120px]">{p.name}:</span>
          <span className="font-mono whitespace-nowrap">{fmt(p.value)}</span>
          <span className="text-muted-foreground whitespace-nowrap">
            ({total > 0 ? ((p.value / total) * 100).toFixed(1) : 0}%)
          </span>
        </div>
      ))}
      <div className="border-t-2 border-foreground pt-1 mt-1 flex justify-between text-muted-foreground">
        <span>Total</span>
        <span className="font-mono">{fmt(total)}</span>
      </div>
    </div>
  );
}

export function MultiSeriesChart({ labels, series, type, display, raw, currencyCode }: MultiSeriesChartProps) {
  const fmt: Fmt = (v) => (raw ? formatDisplay(v, currencyCode) : display ? formatMoney(v) : formatRupiah(v));
  // Defensive: the API may return `null` for these fields when a slice is
  // empty (Go marshals nil slices to null), so never assume they are arrays.
  const safeLabels = Array.isArray(labels) ? labels : [];
  const safeSeries = (Array.isArray(series) ? series : []).filter(
    (s): s is { name: string; data: number[]; color: string } =>
      !!s && typeof s.name === "string" && Array.isArray(s.data)
  );

  const chartData = safeLabels.map((label, i) => {
    const point: Record<string, number | string> = { name: label };
    safeSeries.forEach(s => { point[s.name] = s.data[i] || 0; });
    return point;
  });

  if (chartData.length === 0 || safeSeries.length === 0) {
    return <div className="flex items-center justify-center h-full text-xs font-bold text-muted-foreground">Tidak ada data</div>;
  }

  const renderChart = () => {
    switch (type) {
      case "bar":
        return (
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData} margin={{ top: 10, right: 10, left: 5, bottom: 8 }}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
              <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={6} />
              <YAxis hide />
              <Tooltip content={<CustomTooltip fmt={fmt} />} />
              {safeSeries.map((s) => (
                <Bar key={s.name} dataKey={s.name} name={s.name} fill={s.color} stackId="a" />
              ))}
            </BarChart>
          </ResponsiveContainer>
        );

      case "line":
        return (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={{ top: 10, right: 10, left: 5, bottom: 8 }}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
              <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={6} />
              <YAxis hide domain={["dataMin", "auto"]} />
              <Tooltip content={<CustomTooltip fmt={fmt} />} />
              {safeSeries.map((s) => (
                <Line key={s.name} type="monotone" dataKey={s.name} name={s.name} stroke={s.color} strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
              ))}
            </LineChart>
          </ResponsiveContainer>
        );

      case "area":
        return (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={{ top: 10, right: 10, left: 5, bottom: 8 }}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
              <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={6} />
              <YAxis hide domain={["dataMin", "auto"]} />
              <Tooltip content={<CustomTooltip fmt={fmt} />} />
              {safeSeries.map((s) => (
                <Area key={s.name} type="monotone" dataKey={s.name} name={s.name} stroke={s.color} fill={s.color} fillOpacity={0.15} strokeWidth={2} />
              ))}
            </AreaChart>
          </ResponsiveContainer>
        );
    }
  };

  return (
    <div className="flex flex-col h-full w-full min-h-0 min-w-0">
      <div className="flex-1 w-full min-h-0 min-w-0 relative">
        {renderChart()}
      </div>

      <div className="mt-1 pt-2 border-t border-border/40 shrink-0 max-h-24 sm:max-h-28 overflow-y-auto pr-1">
        <div className="flex flex-wrap items-center justify-center gap-x-3 sm:gap-x-4 gap-y-1 px-1 text-[10px] sm:text-[11px] font-bold">
          {safeSeries.map((s) => (
            <div key={s.name} className="inline-flex items-center gap-1.5 whitespace-nowrap py-0.5">
              <span className="h-2 w-2 rounded-full shrink-0" style={{ backgroundColor: s.color }} />
              <span className="text-foreground truncate max-w-[130px] sm:max-w-[180px]">{s.name}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
