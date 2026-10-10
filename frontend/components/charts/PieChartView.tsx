"use client";

import { Pie, PieChart, ResponsiveContainer, Tooltip, Cell, Legend } from "recharts";

interface PieChartViewProps {
  data?: Array<{ name: string; value: number }>;
  compact?: boolean;
}

const COLORS = ["#3b5bdb", "#10b981", "#f59e0b", "#f43f5e", "#06b6d4", "#8b5cf6", "#ec4899", "#14b8a6"];

function formatIDR(value: number) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
}

function CustomTooltip({ active, payload }: { active?: boolean, payload?: Array<{ color: string, payload: { name: string, value: number } }> }) {
  if (!active || !payload?.length) return null;
  const data = payload[0].payload;
  return (
    <div className="bg-card border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground p-3 text-xs font-bold space-y-1">
      <p className="flex items-center gap-2">
        <span className="w-2 h-2 rounded-full" style={{ backgroundColor: payload[0].color }} />
        {data.name}
      </p>
      <p className="font-mono">{formatIDR(data.value)}</p>
    </div>
  );
}

export function PieChartView({ data, compact }: PieChartViewProps) {
  const chartData = data && data.length > 0 ? data : [{ name: "Belum ada data", value: 1 }];
  const total = chartData.reduce((s, d) => s + d.value, 0);

  const innerR = compact ? 30 : 50;
  const outerR = compact ? 55 : 80;

  return (
    <ResponsiveContainer width="100%" height="100%">
      <PieChart margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
        <Pie
          data={chartData}
          cx="50%" cy="50%"
          innerRadius={innerR} outerRadius={outerR}
          paddingAngle={2}
          dataKey="value"
          stroke="none"
          label={compact ? false : (props: unknown) => {
            const p = props as { name: string, percent?: number };
            return `${p.name} ${(p.percent != null ? (p.percent * 100) : 0).toFixed(1)}%`;
          }}
          labelLine={!compact}
        >
          {chartData.map((entry, index) => (
            <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
          ))}
        </Pie>
        <Tooltip content={<CustomTooltip />} />
        <Legend
          verticalAlign="bottom"
          height={36}
          iconType="circle"
          formatter={(value: string) => {
            const item = chartData.find(d => d.name === value);
            const pct = item && total > 0 ? ((item.value / total) * 100).toFixed(1) : "0";
            return `${value} (${pct}%)`;
          }}
        />
      </PieChart>
    </ResponsiveContainer>
  );
}
