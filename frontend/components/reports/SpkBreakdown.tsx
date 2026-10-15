"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { RadarChartView, type RadarDatum } from "@/components/charts/RadarChartView";
import type { SpkMetricModel } from "@/hooks/useSpkMetrics";

interface SpkBreakdownProps {
  model: SpkMetricModel;
  /** Weighted sub-score per indicator key (from the SPK result breakdown). */
  breakdown: Record<string, number>;
}

/**
 * Renders the SPK indicator breakdown as a radar chart plus a per-indicator
 * list. The labels and weights come from the *model*, so a user with a custom
 * model sees their own indicator names, not the defaults.
 */
export function SpkBreakdown({ model, breakdown }: SpkBreakdownProps) {
  const indicators = model.indicators ?? [];
  const radarData: RadarDatum[] = indicators.map((ind) => ({
    subject: ind.label,
    value: Math.round((breakdown[ind.key] ?? 0) * 10) / 10,
  }));

  return (
    <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader>
        <CardTitle className="text-xs font-black uppercase tracking-wide">Rincian Indikator</CardTitle>
        <CardDescription>
          Setiap indikator dinilai 0–100 lalu dibobot. Bobot total = 100%.
        </CardDescription>
      </CardHeader>
      <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="h-[280px] w-full">
          <RadarChartView data={radarData} name="Sub-skor" color="var(--primary, #f43f5e)" />
        </div>

        <ul className="flex flex-col justify-center gap-3">
          {indicators.length === 0 ? (
            <li className="text-sm font-bold text-muted-foreground">Model tidak memiliki indikator.</li>
          ) : (
            indicators.map((ind) => {
              const value = breakdown[ind.key] ?? 0;
              return (
                <li key={ind.key} className="space-y-1">
                  <div className="flex items-baseline justify-between gap-2">
                    <span className="text-sm font-bold text-foreground">{ind.label}</span>
                    <span className="font-mono text-xs font-bold text-muted-foreground">
                      bobot {(ind.weight * 100).toFixed(0)}%
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="h-2.5 flex-1 border-2 rounded-md border-foreground bg-muted">
                      <div
                        className="h-full bg-primary"
                        style={{ width: `${Math.max(0, Math.min(100, value))}%` }}
                      />
                    </div>
                    <span className="w-10 text-right font-mono text-xs font-black">{value.toFixed(0)}</span>
                  </div>
                </li>
              );
            })
          )}
        </ul>
      </CardContent>
    </Card>
  );
}
