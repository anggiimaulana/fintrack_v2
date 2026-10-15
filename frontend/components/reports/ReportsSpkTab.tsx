"use client";

import * as React from "react";
import { toast } from "sonner";
import { RefreshCw, SlidersHorizontal } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { SpkBreakdown } from "@/components/reports/SpkBreakdown";
import { SpkSimulator } from "@/components/reports/SpkSimulator";
import { SpkTrendChart } from "@/components/reports/SpkTrendChart";
import { SpkMetricEditor } from "@/components/reports/SpkMetricEditor";
import { useCalculateSpk, useSpkHistory, useSpkLatest } from "@/hooks/useSpk";
import { useMySpkModel } from "@/hooks/useSpkMetrics";

/**
 * The "Skor SPK" tab: current score, indicator breakdown, score trend, a
 * what-if simulator, and the metric-model editor (super-admin default vs a
 * personal override).
 */
export function ReportsSpkTab() {
  const { data: latest, isLoading: latestLoading } = useSpkLatest();
  const { data: history = [], isLoading: historyLoading } = useSpkHistory(12);
  const { data: modelResp, isLoading: modelLoading } = useMySpkModel();
  const calculate = useCalculateSpk();
  const [editorOpen, setEditorOpen] = React.useState(false);

  const model = modelResp?.model;
  const breakdown = latest?.breakdown ?? {};

  const handleRecalculate = async () => {
    try {
      await calculate.mutateAsync();
      toast.success("Skor SPK dihitung ulang");
    } catch {
      toast.error("Gagal menghitung ulang skor SPK");
    }
  };

  return (
    <div className="space-y-6">
      {/* Score header */}
      <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div>
            <CardTitle className="text-xs font-black uppercase tracking-wide">Skor Kesehatan Finansial</CardTitle>
            <CardDescription>
              {model?.source === "user"
                ? "Menggunakan metrik kustom Anda."
                : `Menggunakan metrik default${model?.name ? `: ${model.name}` : ""}.`}
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="cursor-pointer"
              onClick={() => setEditorOpen(true)}
              disabled={!model}
            >
              <SlidersHorizontal className="mr-1.5 h-3.5 w-3.5" /> Atur Metrik
            </Button>
            <Button
              type="button"
              size="sm"
              className="cursor-pointer"
              onClick={handleRecalculate}
              disabled={calculate.isPending}
            >
              <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${calculate.isPending ? "animate-spin" : ""}`} />
              {calculate.isPending ? "Menghitung…" : "Hitung ulang"}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {latestLoading ? (
            <div className="h-20 animate-pulse bg-muted" />
          ) : (
            <div className="flex flex-wrap items-end gap-6">
              <div>
                <p className="font-mono text-4xl font-black">{latest?.score?.toFixed(1) ?? "—"}</p>
                <p className="text-sm font-bold text-muted-foreground">{latest?.status ?? "Belum Ada Data"}</p>
              </div>
              <p className="max-w-md text-xs font-bold text-muted-foreground">
                Skor 0–100 dari bobot indikator Anda. Hitung ulang setelah mencatat transaksi baru agar tetap akurat.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Trend */}
      <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader>
          <CardTitle className="text-xs font-black uppercase tracking-wide">Tren Skor</CardTitle>
          <CardDescription>Garis putus-putus menandai ambang &ldquo;Baik&rdquo; (60).</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-[260px] w-full">
            {historyLoading ? <div className="h-full animate-pulse bg-muted" /> : <SpkTrendChart data={history} />}
          </div>
        </CardContent>
      </Card>

      {/* Breakdown + simulator */}
      {modelLoading || !model ? (
        <div className="h-64 animate-pulse border-2 rounded-md border-foreground bg-muted" />
      ) : (
        <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
          <SpkBreakdown model={model} breakdown={breakdown} />
          <SpkSimulator model={model} breakdown={breakdown} />
        </div>
      )}

      {model && (
        <SpkMetricEditor open={editorOpen} onOpenChange={setEditorOpen} current={model} isOverride={modelResp?.is_override ?? false} />
      )}
    </div>
  );
}
