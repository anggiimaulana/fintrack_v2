"use client";

import { Pie, PieChart, ResponsiveContainer, Tooltip, Cell } from "recharts";

interface PieChartViewProps {
  data?: Array<{ name: string; value: number; color?: string }>;
  compact?: boolean;
  showLegend?: boolean;
}

const COLORS = ["#3b5bdb", "#10b981", "#f59e0b", "#f43f5e", "#06b6d4", "#8b5cf6", "#ec4899", "#14b8a6"];

function formatIDR(value: number) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
}

function CustomTooltip({ active, payload }: { active?: boolean, payload?: Array<{ color: string, payload: { name: string, value: number } }> }) {
  if (!active || !payload?.length) return null;
  const data = payload[0].payload;
  return (
    <div className="bg-card border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground p-3 text-xs font-bold space-y-1">
      <p className="flex items-center gap-2">
        <span className="w-2 h-2 rounded-full" style={{ backgroundColor: payload[0].color }} />
        {data.name}
      </p>
      <p className="font-mono">{formatIDR(data.value)}</p>
    </div>
  );
}

export function PieChartView({ data, compact, showLegend = !compact }: PieChartViewProps) {
  const chartData = data && data.length > 0 ? data : [{ name: "Belum ada data", value: 1 }];
  const total = chartData.reduce((s, d) => s + d.value, 0);

  const innerR = compact ? 28 : 45;
  const outerR = compact ? 52 : 75;

  return (
    <div className="flex flex-col h-full w-full min-h-0 min-w-0">
      <div className="flex-1 w-full min-h-0 relative">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
            <Pie
              data={chartData}
              cx="50%" cy="50%"
              innerRadius={innerR} outerRadius={outerR}
              paddingAngle={2}
              dataKey="value"
              stroke="none"
              label={false}
              labelLine={false}
            >
              {chartData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={entry.color ?? COLORS[index % COLORS.length]} />
              ))}
            </Pie>
            <Tooltip content={<CustomTooltip />} />
          </PieChart>
        </ResponsiveContainer>
      </div>

      {showLegend && (
        <div className="mt-1 pt-2 border-t border-border/40 shrink-0 max-h-24 sm:max-h-28 overflow-y-auto pr-1">
          <div className="flex flex-wrap items-center justify-center gap-x-3 sm:gap-x-4 gap-y-1.5 px-2 text-[11px] sm:text-xs font-bold">
            {chartData.map((item, idx) => {
              const pct = total > 0 ? ((item.value / total) * 100).toFixed(1) : "0";
              const color = item.color ?? COLORS[idx % COLORS.length];
              return (
                <div key={item.name} className="inline-flex items-center gap-1.5 whitespace-nowrap py-0.5">
                  <span className="h-2 w-2 rounded-full shrink-0" style={{ backgroundColor: color }} />
                  <span className="text-foreground">{item.name}</span>
                  <span className="font-mono text-muted-foreground">({pct}%)</span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
