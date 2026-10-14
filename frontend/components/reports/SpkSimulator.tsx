"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { RotateCcw } from "lucide-react";
import { projectScore } from "@/lib/spk-score";
import type { SpkMetricModel } from "@/hooks/useSpkMetrics";

interface SpkSimulatorProps {
  model: SpkMetricModel;
  breakdown: Record<string, number>;
}

/**
 * What-if simulator: adjust a "needs vs wants" style indicator by a percentage
 * and see the projected score instantly, computed on the client with the same
 * formula the backend uses (see lib/spk-score.ts).
 *
 * This is a projection, not a promise — the stored score only changes when the
 * transactions do, which is why the copy says "perkiraan".
 */
export function SpkSimulator({ model, breakdown }: SpkSimulatorProps) {
  const [changes, setChanges] = React.useState<Record<string, number>>({});

  const projection = React.useMemo(
    () => projectScore(model, breakdown, changes),
    [model, breakdown, changes]
  );

  const dirty = Object.values(changes).some((v) => v !== 0);

  return (
    <Card className="border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle className="text-xs font-black uppercase tracking-wide">Simulasi What-If</CardTitle>
          <CardDescription>
            Ubah indikator dan lihat perkiraan skor. Ini proyeksi, bukan skor tersimpan.
          </CardDescription>
        </div>
        {dirty && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="cursor-pointer"
            onClick={() => setChanges({})}
          >
            <RotateCcw className="mr-1.5 h-3.5 w-3.5" /> Reset
          </Button>
        )}
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid grid-cols-2 gap-3 border-2 border-foreground bg-muted p-4">
          <div>
            <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Skor saat ini</p>
            <p className="font-mono text-2xl font-black">{projection.current.toFixed(1)}</p>
            <p className="text-xs font-bold text-muted-foreground">{projection.currentStatus}</p>
          </div>
          <div>
            <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Perkiraan skor</p>
            <p className={`font-mono text-2xl font-black ${projection.delta > 0 ? "text-green-700" : projection.delta < 0 ? "text-red-600" : ""}`}>
              {projection.projected.toFixed(1)}
            </p>
            <p className="text-xs font-bold text-muted-foreground">
              {projection.delta === 0 ? "tidak berubah" : `${projection.delta > 0 ? "+" : ""}${projection.delta.toFixed(1)} poin`}
              {projection.projectedStatus !== projection.currentStatus && ` · ${projection.projectedStatus}`}
            </p>
          </div>
        </div>

        {model.indicators.map((ind) => {
          const value = changes[ind.key] ?? 0;
          return (
            <div key={ind.key} className="space-y-1.5">
              <div className="flex items-center justify-between gap-2">
                <label htmlFor={`sim-${ind.key}`} className="text-sm font-bold text-foreground">
                  {ind.label}
                </label>
                <span className="font-mono text-xs font-black">
                  {value > 0 ? "+" : ""}
                  {value}%
                </span>
              </div>
              <input
                id={`sim-${ind.key}`}
                type="range"
                min={-50}
                max={50}
                step={5}
                value={value}
                onChange={(e) => setChanges((prev) => ({ ...prev, [ind.key]: Number(e.target.value) }))}
                className="w-full cursor-pointer accent-[var(--primary)]"
                aria-label={`Ubah ${ind.label} sebesar persen`}
              />
              <p className="text-[11px] font-bold text-muted-foreground">
                {value === 0
                  ? "Tidak diubah"
                  : `${value > 0 ? "Naikkan" : "Turunkan"} nilai dasar indikator ini ${Math.abs(value)}%`}
              </p>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
