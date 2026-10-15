"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";
import type { ReportAnalytics } from "@/hooks/useReportAnalytics";
import type { ReportTypeId } from "@/lib/report-tabs";
import { trendIsGood } from "@/lib/report-tabs";

function formatIDR(value: number): string {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
}

interface DeltaChipProps {
  changePct: number;
  change: number;
  /** Whether a rise is good for this metric (income/savings) or bad (expense). */
  upIsGood: boolean;
  label: string;
}

/** A compact "vs previous period" indicator with type-aware colouring. */
function DeltaChip({ changePct, change, upIsGood, label }: DeltaChipProps) {
  const flat = change === 0;
  const up = change > 0;
  // Colour follows whether the move is *good*, not merely its sign — the fix for
  // the original "income increase shown red" bug.
  const good = flat ? true : up === upIsGood;
  const cls = flat ? "text-muted-foreground" : good ? "text-green-700" : "text-red-600";
  const Icon = flat ? Minus : up ? ArrowUpRight : ArrowDownRight;
  return (
    <span className={`inline-flex items-center gap-1 text-xs font-bold ${cls}`} title={`vs ${label}`}>
      <Icon className="h-3.5 w-3.5" aria-hidden="true" />
      {flat ? "sama" : `${up ? "+" : ""}${changePct.toFixed(1)}%`}
    </span>
  );
}

interface ReportsSummaryTabProps {
  analytics: ReportAnalytics | undefined;
  isLoading: boolean;
  type: ReportTypeId;
}

/** The Ringkasan tab: multi-period deltas, ratios, and best/worst month. */
export function ReportsSummaryTab({ analytics, isLoading, type }: ReportsSummaryTabProps) {
  if (isLoading || !analytics) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-28 animate-pulse border-2 rounded-md border-foreground bg-muted" />
        ))}
      </div>
    );
  }

  const { mom, qoq, yoy } = analytics.comparison;
  const { ratios, best_worst } = analytics;

  // Which comparison set is meaningful depends on the selected window; show MoM
  // by default but let the user see the others side by side.
  const comparisons = [
    { key: "mom", set: mom, label: "Bulan lalu" },
    { key: "qoq", set: qoq, label: "Kuartal lalu" },
    { key: "yoy", set: yoy, label: "Tahun lalu" },
  ];

  const metrics: { key: "income" | "expense" | "savings" | "surplus"; label: string; upIsGood: boolean }[] = [
    { key: "income", label: "Pemasukan", upIsGood: true },
    { key: "expense", label: "Pengeluaran", upIsGood: false },
    { key: "savings", label: "Tabungan", upIsGood: true },
    { key: "surplus", label: "Surplus", upIsGood: true },
  ];

  return (
    <div className="space-y-6">
      {/* Multi-period comparison */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {comparisons.map(({ key, set, label }) => (
          <Card key={key} className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
            <CardHeader>
              <CardTitle className="text-xs font-black uppercase tracking-wide">vs {label}</CardTitle>
              <CardDescription className="text-[11px]">{set.start} s/d {set.end}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {metrics.map((m) => {
                const d = set[m.key];
                return (
                  <div key={m.key} className="flex items-center justify-between gap-2">
                    <span className="text-xs font-bold text-muted-foreground">{m.label}</span>
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs font-bold">{formatIDR(d.current)}</span>
                      <DeltaChip changePct={d.change_pct} change={d.change} upIsGood={m.upIsGood} label={label} />
                    </div>
                  </div>
                );
              })}
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Ratios + best/worst */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
          <CardHeader>
            <CardTitle className="text-xs font-black uppercase tracking-wide">Rasio Keuangan</CardTitle>
            <CardDescription>Dihitung dari {ratios.months_spanned} bulan pada periode ini.</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-3 gap-3 text-center">
            <div className="border-2 rounded-md border-foreground bg-muted p-3">
              <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Rasio Tabungan</p>
              <p className="font-mono text-xl font-black text-green-700">{ratios.savings_rate.toFixed(1)}%</p>
            </div>
            <div className="border-2 rounded-md border-foreground bg-muted p-3">
              <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Rasio Beban</p>
              <p className="font-mono text-xl font-black text-red-600">{ratios.expense_ratio.toFixed(1)}%</p>
            </div>
            <div className="border-2 rounded-md border-foreground bg-muted p-3">
              <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Runway</p>
              <p className="font-mono text-xl font-black">
                {ratios.runway_months > 0 ? `${ratios.runway_months.toFixed(1)} bln` : "—"}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
          <CardHeader>
            <CardTitle className="text-xs font-black uppercase tracking-wide">Periode Terbaik &amp; Terburuk</CardTitle>
            <CardDescription>Berdasarkan surplus (pemasukan − pengeluaran) tahun ini.</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-3">
            <div className="border-2 rounded-md border-green-600 bg-green-50 p-3 dark:bg-green-950/30">
              <p className="text-[10px] font-black uppercase tracking-wide text-green-700">Terbaik</p>
              <p className="text-sm font-black">{best_worst.best_month || "—"}</p>
              <p className="font-mono text-xs font-bold text-green-700">{formatIDR(best_worst.best_value)}</p>
            </div>
            <div className="border-2 rounded-md border-red-600 bg-red-50 p-3 dark:bg-red-950/30">
              <p className="text-[10px] font-black uppercase tracking-wide text-red-700">Terburuk</p>
              <p className="text-sm font-black">{best_worst.worst_month || "—"}</p>
              <p className="font-mono text-xs font-bold text-red-700">{formatIDR(best_worst.worst_value)}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      <p className="text-[11px] font-bold text-muted-foreground">
        Analisis tipe <span className="uppercase">{type}</span>. Ganti tipe di header untuk melihat pemasukan atau tabungan.
      </p>
    </div>
  );
}

export { formatIDR, DeltaChip, trendIsGood };
