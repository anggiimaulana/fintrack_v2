"use client";

import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis, CartesianGrid, ReferenceLine } from "recharts";
import type { SpkHistoryPoint } from "@/hooks/useSpk";

interface SpkTrendChartProps {
  data: SpkHistoryPoint[];
}

/**
 * Line chart of the SPK score over time. The y-axis is fixed to 0–100 (the
 * score's range) so the shape of the line is honest — an auto-scaled axis would
 * make a 2-point change look dramatic.
 */
export function SpkTrendChart({ data }: SpkTrendChartProps) {
  if (data.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-xs font-bold text-muted-foreground">
        Belum ada riwayat skor. Hitung ulang untuk mulai mencatat.
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height="100%">
      <LineChart data={data} margin={{ top: 10, right: 12, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
        <XAxis dataKey="label" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={8} />
        <YAxis domain={[0, 100]} axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} width={28} />
        <Tooltip
          contentStyle={{ borderRadius: 0, border: "2px solid #000", boxShadow: "4px 4px 0 0 #000", fontSize: 12, fontWeight: 700 }}
          formatter={(value) => {
            const n = typeof value === "number" ? value : Number(value);
            return [`${n.toFixed(1)} (${statusForScore(n)})`, "Skor"] as [string, string];
          }}
        />
        {/* The "Baik" threshold, so a dip below it is visually obvious. */}
        <ReferenceLine y={60} stroke="#f59e0b" strokeDasharray="4 4" />
        <Line type="monotone" dataKey="score" stroke="#f43f5e" strokeWidth={3} dot={{ r: 3 }} activeDot={{ r: 6 }} />
      </LineChart>
    </ResponsiveContainer>
  );
}

/** A lightweight status label for the tooltip (mirrors the default thresholds). */
function statusForScore(score: number): string {
  if (score >= 80) return "Sangat Baik";
  if (score >= 60) return "Baik";
  if (score >= 40) return "Cukup";
  if (score >= 20) return "Boros";
  return "Kritis";
}
