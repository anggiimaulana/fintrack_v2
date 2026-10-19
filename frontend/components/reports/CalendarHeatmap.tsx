"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { useTransactions } from "@/hooks/useTransactions";
import { useAdminCategories } from "@/hooks/useAdminCategories";
import type { DailyPoint } from "@/hooks/useReportAnalytics";
import { formatMoney } from "@/lib/currency";

function formatIDR(value: number): string {
  return formatMoney(value);
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

function DayDetailModal({
  selected,
  onClose,
}: {
  selected: DailyPoint | null;
  onClose: () => void;
}) {
  const { data: txData, isLoading } = useTransactions(
    { dateFrom: selected?.date, dateTo: selected?.date, perPage: 50 },
    { enabled: !!selected?.date }
  );

  const { categories: incCats } = useAdminCategories("income");
  const { categories: expCats } = useAdminCategories("expense");
  const { categories: savCats } = useAdminCategories("savings");

  const catMap = React.useMemo(() => {
    const map: Record<string, string> = {};
    incCats?.forEach((c) => { map[c.id] = c.name; });
    expCats?.forEach((c) => { map[c.id] = c.name; });
    savCats?.forEach((c) => { map[c.id] = c.name; });
    return map;
  }, [incCats, expCats, savCats]);

  if (!selected) return null;

  const dateFormatted = new Date(selected.date + "T00:00:00").toLocaleDateString("id-ID", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  });
  const txs = txData?.data ?? [];

  return (
    <Dialog open={!!selected} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg border-2 rounded-md border-foreground bg-card shadow-[6px_6px_0px_0px] shadow-foreground">
        <DialogHeader>
          <DialogTitle className="text-base font-black uppercase tracking-wide">
            Detail Harian
          </DialogTitle>
          <DialogDescription className="text-xs font-bold text-foreground">
            {dateFormatted}
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 my-2">
          <div className="border-2 rounded-md border-foreground bg-background p-2.5">
            <p className="text-[10px] font-black uppercase text-muted-foreground">Pemasukan</p>
            <p className="font-mono text-xs font-black text-green-700">{formatIDR(selected.income)}</p>
          </div>
          <div className="border-2 rounded-md border-foreground bg-background p-2.5">
            <p className="text-[10px] font-black uppercase text-muted-foreground">Pengeluaran</p>
            <p className="font-mono text-xs font-black text-red-600">{formatIDR(selected.expense)}</p>
          </div>
          <div className="border-2 rounded-md border-foreground bg-background p-2.5">
            <p className="text-[10px] font-black uppercase text-muted-foreground">Tabungan</p>
            <p className="font-mono text-xs font-black text-amber-600">{formatIDR(selected.savings)}</p>
          </div>
          <div className="border-2 rounded-md border-foreground bg-background p-2.5">
            <p className="text-[10px] font-black uppercase text-muted-foreground">Net</p>
            <p className={`font-mono text-xs font-black ${selected.net >= 0 ? "text-primary" : "text-red-600"}`}>
              {formatIDR(selected.net)}
            </p>
          </div>
        </div>

        <div className="space-y-2 mt-2">
          <div className="flex items-center justify-between">
            <h4 className="text-xs font-black uppercase tracking-wide text-foreground">
              Daftar Transaksi
            </h4>
            <span className="text-[11px] font-bold text-muted-foreground">
              {txs.length} transaksi
            </span>
          </div>

          {isLoading ? (
            <div className="space-y-2 py-2">
              <Skeleton className="h-11 w-full border-2 rounded-md border-foreground" />
              <Skeleton className="h-11 w-full border-2 rounded-md border-foreground" />
            </div>
          ) : txs.length === 0 ? (
            <div className="border-2 rounded-md border-foreground bg-muted p-4 text-center text-xs font-bold text-muted-foreground">
              Tidak ada catatan transaksi pada tanggal ini.
            </div>
          ) : (
            <div className="space-y-2 max-h-[260px] overflow-y-auto pr-1">
              {txs.map((tx) => {
                const isInc = tx.type === "income";
                const isSav = (tx.type as string) === "savings";
                const sign = isInc ? "+" : isSav ? "→" : "-";
                const amountCls = isInc ? "text-green-700" : isSav ? "text-amber-600" : "text-red-600";
                const catId = tx.expense_category_id || tx.income_category_id || tx.savings_category_id || "";
                const catName = tx.category || tx.category_name || (catId ? catMap[catId] : undefined) || "Umum";

                return (
                  <div
                    key={tx.id}
                    className="flex items-center justify-between gap-3 border-2 rounded-md border-foreground bg-background p-2.5 text-xs"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-black truncate">{catName}</span>
                        {tx.source && tx.source !== "manual" && (
                          <span className="text-[9px] font-black uppercase px-1 py-0.5 border rounded border-foreground bg-muted">
                            {tx.source}
                          </span>
                        )}
                      </div>
                      {tx.description && (
                        <p className="text-[11px] font-bold text-muted-foreground truncate">{tx.description}</p>
                      )}
                    </div>
                    <span className={`font-mono font-black shrink-0 ${amountCls}`}>
                      {sign} {formatIDR(tx.amount)}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

/**
 * Calendar heatmap of daily spending.
 * Renders the full month grid.
 * Clicking a day cell opens the detail modal.
 */
export function CalendarHeatmap({ daily }: CalendarHeatmapProps) {
  const [selected, setSelected] = React.useState<DailyPoint | null>(null);

  const fullDaily = React.useMemo(() => {
    if (!daily || daily.length === 0) {
      const now = new Date();
      const y = now.getFullYear();
      const m = now.getMonth();
      const totalDays = new Date(y, m + 1, 0).getDate();
      const result: DailyPoint[] = [];
      for (let day = 1; day <= totalDays; day++) {
        const dateStr = `${y}-${String(m + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
        result.push({ date: dateStr, income: 0, expense: 0, savings: 0, net: 0 });
      }
      return result;
    }

    const firstDate = new Date(daily[0].date + "T00:00:00");
    const y = firstDate.getFullYear();
    const m = firstDate.getMonth();
    const totalDays = new Date(y, m + 1, 0).getDate();

    const isSingleMonth = daily.every((d) => {
      const dt = new Date(d.date + "T00:00:00");
      return dt.getFullYear() === y && dt.getMonth() === m;
    });

    if (isSingleMonth && daily.length < totalDays) {
      const byDate = new Map(daily.map((d) => [d.date, d]));
      const result: DailyPoint[] = [];
      for (let day = 1; day <= totalDays; day++) {
        const dateStr = `${y}-${String(m + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
        result.push(
          byDate.get(dateStr) ?? {
            date: dateStr,
            income: 0,
            expense: 0,
            savings: 0,
            net: 0,
          }
        );
      }
      return result;
    }
    return daily;
  }, [daily]);

  const maxExpense = Math.max(...fullDaily.map((d) => d.expense), 0);

  // Leading blanks so the first day lands in the right weekday column.
  // getDay(): 0=Sunday; we render Monday-first, so shift by -1 (mod 7).
  const first = new Date(fullDaily[0].date + "T00:00:00");
  const leading = (first.getDay() + 6) % 7;

  const totalExpense = fullDaily.reduce((s, d) => s + d.expense, 0);
  const activeDays = fullDaily.filter((d) => d.expense > 0).length;
  const avgPerActiveDay = activeDays > 0 ? totalExpense / activeDays : 0;
  const monthTitle = fullDaily[0]
    ? new Date(fullDaily[0].date + "T00:00:00").toLocaleDateString("id-ID", { month: "long", year: "numeric" })
    : "";

  return (
    <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader>
        <CardTitle className="text-xs font-black uppercase tracking-wide">
          Kalender Pengeluaran{monthTitle ? ` — ${monthTitle}` : ""}
        </CardTitle>
        <CardDescription>
          Warna makin pekat = pengeluaran makin besar. Klik sel untuk detail transaksi harian.
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
              {fullDaily.map((d) => {
                const level = heatLevel(d.expense, maxExpense);
                const day = Number(d.date.slice(8, 10));
                return (
                  <button
                    key={d.date}
                    type="button"
                    onClick={() => setSelected(d)}
                    title={`${d.date}: ${formatIDR(d.expense)}`}
                    aria-label={`${d.date}, pengeluaran ${formatIDR(d.expense)}`}
                    className={`flex aspect-square items-center justify-center border-2 rounded-md border-foreground text-[10px] font-bold transition-transform hover:scale-105 cursor-pointer ${HEAT_CLASSES[level]} ${level >= 3 ? "text-primary-foreground" : "text-foreground"}`}
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
            <span key={i} className={`h-3 w-3 border rounded-md border-foreground ${c}`} />
          ))}
          <span>Banyak</span>
        </div>

        <DayDetailModal selected={selected} onClose={() => setSelected(null)} />
      </CardContent>
    </Card>
  );
}
