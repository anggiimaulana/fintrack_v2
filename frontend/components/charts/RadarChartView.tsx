"use client";

import { Radar, RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, ResponsiveContainer, Tooltip } from "recharts";

export interface RadarDatum {
  subject: string;
  value: number;
  /** Optional per-datum cap; defaults to 100 (all SPK sub-scores are 0..100). */
  fullMark?: number;
}

interface RadarChartViewProps {
  data?: RadarDatum[];
  /** Series label shown in the tooltip. */
  name?: string;
  /** Hex colour for the filled area. */
  color?: string;
}

/**
 * Radar chart for the SPK indicator breakdown. Accepts real data; falls back to
 * an empty chart (not demo data) when nothing is supplied, so a user with no
 * score sees a blank grid rather than someone else's numbers.
 */
export function RadarChartView({ data = [], name = "Skor", color = "#f43f5e" }: RadarChartViewProps) {
  const chartData = data.map((d) => ({ ...d, fullMark: d.fullMark ?? 100 }));

  if (chartData.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-xs font-bold text-muted-foreground">
        Belum ada data indikator.
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height="100%">
      <RadarChart cx="50%" cy="50%" outerRadius="75%" data={chartData} margin={{ top: 20, right: 30, bottom: 20, left: 30 }}>
        <PolarGrid stroke="#e2e8f0" />
        <PolarAngleAxis dataKey="subject" tick={{ fill: "#475569", fontSize: 11 }} />
        <PolarRadiusAxis angle={30} domain={[0, 100]} tick={false} axisLine={false} />
        <Radar name={name} dataKey="value" stroke={color} fill={color} fillOpacity={0.45} />
        <Tooltip
          contentStyle={{ borderRadius: "8px", border: "none", boxShadow: "0 4px 12px rgba(0,0,0,0.08)" }}
        />
      </RadarChart>
    </ResponsiveContainer>
  );
}
