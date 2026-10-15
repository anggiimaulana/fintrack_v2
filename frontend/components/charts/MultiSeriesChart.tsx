"use client";

import { Bar, BarChart, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis, CartesianGrid, Legend, Area, AreaChart } from "recharts";

interface MultiSeriesChartProps {
  labels: string[];
  series: Array<{ name: string; data: number[]; color: string }>;
  type: "bar" | "line" | "area";
}

function formatIDR(value: number) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
}

function CustomTooltip({ active, payload, label }: { active?: boolean, payload?: Array<{value: number, color: string, name: string}>, label?: string }) {
  if (!active || !payload?.length) return null;
  const total = payload.reduce((sum: number, p: {value: number, color: string, name: string}) => sum + ((p.value as number) || 0), 0);
  return (
    <div className="bg-card border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground p-3 text-xs font-bold space-y-1 max-h-48 overflow-y-auto">
      <p className="text-muted-foreground mb-1">{label}</p>
      {payload.map((p: {value: number, color: string, name: string}, i: number) => (
        <div key={i} className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: p.color }} />
          <span className="truncate max-w-[120px]">{p.name}:</span>
          <span className="font-mono whitespace-nowrap">{formatIDR(p.value)}</span>
          <span className="text-muted-foreground whitespace-nowrap">
            ({total > 0 ? ((p.value / total) * 100).toFixed(1) : 0}%)
          </span>
        </div>
      ))}
      <div className="border-t-2 border-foreground pt-1 mt-1 flex justify-between text-muted-foreground">
        <span>Total</span>
        <span className="font-mono">{formatIDR(total)}</span>
      </div>
    </div>
  );
}

export function MultiSeriesChart({ labels, series, type }: MultiSeriesChartProps) {
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

  const commonProps = {
    width: "100%" as const,
    height: "100%" as const,
  };

  switch (type) {
    case "bar":
      return (
        <ResponsiveContainer {...commonProps}>
          <BarChart data={chartData} margin={{ top: 10, right: 10, left: 5, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
            <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={8} />
            <YAxis hide />
            <Tooltip content={<CustomTooltip />} />
            <Legend wrapperStyle={{ fontSize: 10, fontWeight: 700 }} iconType="circle" />
            {safeSeries.map((s) => (
              <Bar key={s.name} dataKey={s.name} name={s.name} fill={s.color} stackId="a" />
            ))}
          </BarChart>
        </ResponsiveContainer>
      );

    case "line":
      return (
        <ResponsiveContainer {...commonProps}>
          <LineChart data={chartData} margin={{ top: 10, right: 10, left: 5, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
            <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={8} />
            <YAxis hide domain={["dataMin", "auto"]} />
            <Tooltip content={<CustomTooltip />} />
            <Legend wrapperStyle={{ fontSize: 10, fontWeight: 700 }} iconType="circle" />
            {safeSeries.map((s) => (
              <Line key={s.name} type="monotone" dataKey={s.name} name={s.name} stroke={s.color} strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
            ))}
          </LineChart>
        </ResponsiveContainer>
      );

    case "area":
      return (
        <ResponsiveContainer {...commonProps}>
          <AreaChart data={chartData} margin={{ top: 10, right: 10, left: 5, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
            <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={8} />
            <YAxis hide domain={["dataMin", "auto"]} />
            <Tooltip content={<CustomTooltip />} />
            <Legend wrapperStyle={{ fontSize: 10, fontWeight: 700 }} iconType="circle" />
            {safeSeries.map((s) => (
              <Area key={s.name} type="monotone" dataKey={s.name} name={s.name} stroke={s.color} fill={s.color} fillOpacity={0.15} strokeWidth={2} />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      );
  }
}
