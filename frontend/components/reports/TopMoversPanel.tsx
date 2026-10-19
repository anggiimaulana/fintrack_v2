"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { TrendingDown, TrendingUp } from "lucide-react";
import type { CategoryAnomaly, TopMovers } from "@/hooks/useReportAnalytics";
import { formatMoney } from "@/lib/currency";

function formatIDR(value: number): string {
  return formatMoney(value);
}

interface MoverListProps {
  title: string;
  movers: TopMovers["up"];
  variant: "up" | "down";
}

function MoverList({ title, movers, variant }: MoverListProps) {
  const Icon = variant === "up" ? TrendingUp : TrendingDown;
  const accent = variant === "up" ? "text-red-600" : "text-green-700";
  return (
    <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-xs font-black uppercase tracking-wide">
          <Icon className={`h-4 w-4 ${accent}`} aria-hidden="true" /> {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {movers.length === 0 ? (
          <p className="py-4 text-center text-xs font-bold text-muted-foreground">Tidak ada perubahan berarti.</p>
        ) : (
          movers.map((m) => (
            <div key={m.category_name} className="flex items-center justify-between gap-2 border-b-2 border-foreground/10 pb-2 last:border-b-0">
              <span className="min-w-0 truncate text-sm font-bold">{m.category_name}</span>
              <div className="flex shrink-0 items-center gap-2">
                <span className={`font-mono text-xs font-black ${accent}`}>
                  {m.change > 0 ? "+" : ""}{formatIDR(m.change)}
                </span>
                <span className={`w-16 text-right font-mono text-[11px] font-bold ${accent}`}>
                  {m.change_pct > 0 ? "+" : ""}{m.change_pct.toFixed(0)}%
                </span>
              </div>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  );
}

interface TopMoversPanelProps {
  movers: TopMovers | undefined;
}

/** Top Movers: the categories that rose and fell the most versus last period. */
export function TopMoversPanel({ movers }: TopMoversPanelProps) {
  if (!movers) {
    return <div className="h-48 animate-pulse border-2 rounded-md border-foreground bg-muted" />;
  }
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <MoverList title="Kenaikan Terbesar" movers={movers.up} variant="up" />
      <MoverList title="Penurunan Terbesar" movers={movers.down} variant="down" />
    </div>
  );
}

interface AnomalyPanelProps {
  anomalies: CategoryAnomaly[];
}

/** Anomaly detection: categories deviating > 2σ from their own recent history. */
export function AnomalyPanel({ anomalies }: AnomalyPanelProps) {
  return (
    <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader>
        <CardTitle className="text-xs font-black uppercase tracking-wide">Deteksi Anomali</CardTitle>
        <CardDescription>Kategori yang menyimpang lebih dari 2 simpangan baku dari kebiasaannya.</CardDescription>
      </CardHeader>
      <CardContent>
        {anomalies.length === 0 ? (
          <p className="py-4 text-center text-xs font-bold text-muted-foreground">
            Tidak ada anomali terdeteksi. Pola belanja Anda stabil.
          </p>
        ) : (
          <ul className="space-y-2">
            {anomalies.map((a) => (
              <li key={a.category_name} className="flex items-center justify-between gap-3 border-2 rounded-md border-foreground p-2">
                <span className="min-w-0 truncate text-sm font-bold">{a.category_name}</span>
                <div className="flex shrink-0 items-center gap-2">
                  <span className={`border-2 rounded-md px-2 py-0.5 text-[10px] font-black uppercase ${a.direction === "spike" ? "border-red-600 text-red-600" : "border-green-700 text-green-700"}`}>
                    {a.direction === "spike" ? "Lonjakan" : "Penurunan"}
                  </span>
                  <span className="font-mono text-xs font-bold">z={a.z_score.toFixed(1)}</span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

export { formatIDR };
