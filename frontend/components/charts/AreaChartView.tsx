"use client";

import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis, CartesianGrid } from "recharts";

interface AreaChartViewProps {
  data?: Array<{ name: string; balance: number }>;
}

function formatIDR(value: number) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
}

function CustomTooltip({ active, payload, label }: { active?: boolean, payload?: Array<{value: number, color: string, name: string}>, label?: string }) {
  if (!active || !payload?.length) return null;
  const total = payload.reduce((sum: number, p: {value: number, color: string, name: string}) => sum + Math.abs((p.value as number) || 0), 0);
  return (
    <div className="bg-card border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground p-3 text-xs font-bold space-y-1">
      <p className="text-muted-foreground mb-1">{label}</p>
      {payload.map((p: {value: number, color: string, name: string}, i: number) => (
        <div key={i} className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full" style={{ backgroundColor: p.color }} />
          <span>{p.name}:</span>
          <span className="font-mono">{formatIDR(p.value)}</span>
          <span className="text-muted-foreground">
            ({total > 0 ? ((Math.abs(p.value) / total) * 100).toFixed(1) : 0}%)
          </span>
        </div>
      ))}
    </div>
  );
}

export function AreaChartView({ data }: AreaChartViewProps) {
  const chartData = data && data.length > 0 ? data : [
    { name: "Jan", balance: 0 },
    { name: "Feb", balance: 0 },
    { name: "Mar", balance: 0 },
  ];

  return (
    <ResponsiveContainer width="100%" height="100%">
      <AreaChart data={chartData} margin={{ top: 10, right: 10, left: 10, bottom: 0 }}>
        <defs>
          <linearGradient id="colorBalance" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#3b5bdb" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#3b5bdb" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
        <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: "#94a3b8" }} dy={10} />
        <YAxis hide domain={["dataMin", "auto"]} />
        <Tooltip content={<CustomTooltip />} />
        <Area type="monotone" dataKey="balance" name="Saldo" stroke="#3b5bdb" strokeWidth={3} fillOpacity={1} fill="url(#colorBalance)" />
      </AreaChart>
    </ResponsiveContainer>
  );
}
