"use client";

import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis, CartesianGrid } from "recharts";
import { formatDisplay, formatMoney } from "@/lib/currency";
import { formatRupiah } from "@/lib/format";

interface BarChartViewProps {
  data?: Array<{ name: string; income: number; expense: number; savings?: number }>;
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
    <div className="bg-card border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground p-3 text-xs font-bold space-y-1">
      <p className="text-muted-foreground mb-1">{label}</p>
      {payload.map((p: {value: number, color: string, name: string}, i: number) => (
        <div key={i} className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full" style={{ backgroundColor: p.color }} />
          <span>{p.name}:</span>
          <span className="font-mono">{fmt(p.value)}</span>
          <span className="text-muted-foreground">
            ({total > 0 ? ((p.value / total) * 100).toFixed(1) : 0}%)
          </span>
        </div>
      ))}
    </div>
  );
}

export function BarChartView({ data, display, raw, currencyCode }: BarChartViewProps) {
  const fmt: Fmt = (v) => (raw ? formatDisplay(v, currencyCode) : display ? formatMoney(v) : formatRupiah(v));
  const chartData = data && data.length > 0 ? data : [
    { name: "Jan", income: 0, expense: 0, savings: 0 },
    { name: "Feb", income: 0, expense: 0, savings: 0 },
  ];

  return (
    <div className="flex flex-col h-full w-full min-h-0">
      <div className="flex-1 w-full min-h-0 relative">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={chartData} margin={{ top: 10, right: 10, left: 10, bottom: 8 }}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
            <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={6} />
            <YAxis hide />
            <Tooltip cursor={{ fill: "#f1f5f9" }} content={<CustomTooltip fmt={fmt} />} />
            <Bar dataKey="savings" name="Tabungan" fill="#f59e0b" />
            <Bar dataKey="income" name="Pemasukan" fill="#10b981" />
            <Bar dataKey="expense" name="Pengeluaran" fill="#f43f5e" />
          </BarChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-1 pt-2 border-t border-border/40 shrink-0">
        <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1 px-2 text-[11px] sm:text-xs font-bold">
          <div className="inline-flex items-center gap-1.5 whitespace-nowrap py-0.5">
            <span className="h-2 w-2 rounded-full shrink-0 bg-[#10b981]" />
            <span className="text-foreground">Pemasukan</span>
          </div>
          <div className="inline-flex items-center gap-1.5 whitespace-nowrap py-0.5">
            <span className="h-2 w-2 rounded-full shrink-0 bg-[#f43f5e]" />
            <span className="text-foreground">Pengeluaran</span>
          </div>
          <div className="inline-flex items-center gap-1.5 whitespace-nowrap py-0.5">
            <span className="h-2 w-2 rounded-full shrink-0 bg-[#f59e0b]" />
            <span className="text-foreground">Tabungan</span>
          </div>
        </div>
      </div>
    </div>
  );
}
