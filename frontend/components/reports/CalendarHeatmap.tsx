"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { DailyPoint } from "@/hooks/useReportAnalytics";

function formatIDR(value: number): string {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
}

/** Maps a day's expense to a heat level 0..4 using the period's max as the scale. */
function heatLevel(value: number, max: number): number {
  if (value <= 0 || max <= 0) return 0;
  const ratio = value / max;
  if (ratio >= 0.75) return 4;
  if (ratio >= 0.5) return 3;
  if (ratio >= 0.25) return 2;
  return 1;
}

const HEAT_CLASSES = [
  "bg-muted",
  "bg-primary/20",
  "bg-primary/40",
  "bg-primary/70",
  "bg-primary",
];

const DAY_LABELS = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];

interface CalendarHeatmapProps {
  daily: DailyPoint[] | undefined;
}

/**
 * Calendar heatmap of daily spending. Cells are laid out by weekday column so
 * the weekly spending rhythm is visible at a glance. Uses CSS grid rather than a
 * chart library — a heatmap is just a grid of coloured cells.
 */
export function CalendarHeatmap({ daily }: CalendarHeatmapProps) {
  const [selected, setSelected] = React.useState<DailyPoint | null>(null);

  if (!daily || daily.length === 0) {
    return (
      <Card className="border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader>
          <CardTitle className="text-xs font-black uppercase tracking-wide">Kalender Pengeluaran</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="py-8 text-center text-xs font-bold text-muted-foreground">Belum ada data harian.</p>
        </CardContent>
      </Card>
    );
  }

  const maxExpense = Math.max(...daily.map((d) => d.expense), 0);

  // Leading blanks so the first day lands in the right weekday column.
  // getDay(): 0=Sunday; we render Monday-first, so shift by -1 (mod 7).
  const first = new Date(daily[0].date + "T00:00:00");
  const leading = (first.getDay() + 6) % 7;

  const totalExpense = daily.reduce((s, d) => s + d.expense, 0);
  const activeDays = daily.filter((d) => d.expense > 0).length;
  const avgPerActiveDay = activeDays > 0 ? totalExpense / activeDays : 0;

  return (
    <Card className="border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader>
        <CardTitle className="text-xs font-black uppercase tracking-wide">Kalender Pengeluaran</CardTitle>
        <CardDescription>
          Warna makin pekat = pengeluaran makin besar. Klik sel untuk detail hari.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap gap-4 text-xs font-bold">
          <span>Total: <span className="font-mono">{formatIDR(totalExpense)}</span></span>
          <span>Hari aktif: <span className="font-mono">{activeDays}</span></span>
          <span>Rata-rata/hari aktif: <span className="font-mono">{formatIDR(avgPerActiveDay)}</span></span>
        </div>

        <div className="overflow-x-auto">
          <div className="min-w-[420px]">
            <div className="mb-1 grid grid-cols-7 gap-1">
              {DAY_LABELS.map((d) => (
                <div key={d} className="text-center text-[10px] font-black uppercase text-muted-foreground">{d}</div>
              ))}
            </div>
            <div className="grid grid-cols-7 gap-1">
              {Array.from({ length: leading }).map((_, i) => (
                <div key={`blank-${i}`} aria-hidden="true" />
              ))}
              {daily.map((d) => {
                const level = heatLevel(d.expense, maxExpense);
                const day = Number(d.date.slice(8, 10));
                return (
                  <button
                    key={d.date}
                    type="button"
                    onClick={() => setSelected(d)}
                    title={`${d.date}: ${formatIDR(d.expense)}`}
                    aria-label={`${d.date}, pengeluaran ${formatIDR(d.expense)}`}
                    className={`flex aspect-square items-center justify-center border-2 border-foreground text-[10px] font-bold transition-transform hover:scale-105 ${HEAT_CLASSES[level]} ${level >= 3 ? "text-primary-foreground" : "text-foreground"}`}
                  >
                    {day}
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2 text-[10px] font-bold text-muted-foreground">
          <span>Sedikit</span>
          {HEAT_CLASSES.map((c, i) => (
            <span key={i} className={`h-3 w-3 border border-foreground ${c}`} />
          ))}
          <span>Banyak</span>
        </div>

        {selected && (
          <div className="border-2 border-foreground bg-muted p-3 text-xs font-bold">
            <p className="mb-1 font-black">{selected.date}</p>
            <div className="grid grid-cols-2 gap-1 sm:grid-cols-4">
              <span>Masuk: <span className="font-mono text-green-700">{formatIDR(selected.income)}</span></span>
              <span>Keluar: <span className="font-mono text-red-600">{formatIDR(selected.expense)}</span></span>
              <span>Tabungan: <span className="font-mono text-amber-600">{formatIDR(selected.savings)}</span></span>
              <span>Net: <span className="font-mono">{formatIDR(selected.net)}</span></span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
