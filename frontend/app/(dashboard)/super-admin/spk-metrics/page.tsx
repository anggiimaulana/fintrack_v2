"use client";

import * as React from "react";
import { toast } from "sonner";
import { Plus, Trash2, Save, Info } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  useSpkDefaultModel,
  useSaveSpkDefault,
  SPK_METRIC_TYPE_LABELS,
  type SpkMetricIndicator,
  type SpkMetricModel,
  type SpkMetricType,
} from "@/hooks/useSpkMetrics";

const METRIC_TYPES = Object.keys(SPK_METRIC_TYPE_LABELS) as SpkMetricType[];

/** Deep copy so edits never mutate the query cache. */
function cloneModel(m: SpkMetricModel): SpkMetricModel {
  return {
    indicators: m.indicators.map((i) => ({ ...i, categories: i.categories ? [...i.categories] : undefined })),
    status_thresholds: m.status_thresholds.map((t) => ({ ...t })),
    source: m.source,
    name: m.name,
  };
}

export default function SuperAdminSpkMetricsPage() {
  const { data: model, isLoading } = useSpkDefaultModel();
  const save = useSaveSpkDefault();

  const [draft, setDraft] = React.useState<SpkMetricModel | null>(null);
  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  // Track the model we seeded from, so a background refetch does not clobber an
  // in-progress edit. Adjusting state during render (rather than in an effect) is
  // the React-recommended alternative to setState-in-effect.
  const [seededFrom, setSeededFrom] = React.useState<SpkMetricModel | null>(null);

  if (model && model !== seededFrom) {
    setSeededFrom(model);
    setDraft(cloneModel(model));
    setName(model.name ?? "Model SPK Default");
  }

  const weightSum = draft?.indicators.reduce((s, i) => s + (Number(i.weight) || 0), 0) ?? 0;
  const weightsValid = Math.abs(weightSum - 1) < 1e-6;

  const updateIndicator = (idx: number, patch: Partial<SpkMetricIndicator>) => {
    setDraft((d) => (d ? { ...d, indicators: d.indicators.map((ind, i) => (i === idx ? { ...ind, ...patch } : ind)) } : d));
  };

  const addIndicator = () => {
    setDraft((d) => (d ? {
      ...d,
      indicators: [...d.indicators, { key: `indikator_${d.indicators.length + 1}`, label: "Indikator Baru", weight: 0, target: 1, type: "consistency" }],
    } : d));
  };

  const removeIndicator = (idx: number) => {
    setDraft((d) => (d ? { ...d, indicators: d.indicators.filter((_, i) => i !== idx) } : d));
  };

  const handleSave = async () => {
    if (!draft) return;
    if (!weightsValid) {
      toast.error("Total bobot harus tepat 1.0 (100%)");
      return;
    }
    try {
      await save.mutateAsync({
        name,
        description,
        indicators: draft.indicators,
        status_thresholds: draft.status_thresholds,
      });
      toast.success("Model SPK default disimpan");
    } catch (e) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message ?? "Gagal menyimpan model";
      toast.error(msg);
    }
  };

  if (isLoading || !draft) {
    return <div className="h-64 animate-pulse border-2 border-foreground bg-muted" />;
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div>
        <h1 className="text-xl font-black uppercase tracking-wide">Metrik SPK Default</h1>
        <p className="mt-1 text-xs font-bold text-muted-foreground">
          Model penilaian kesehatan finansial yang berlaku untuk semua pengguna. Pengguna dapat menimpanya dengan metrik pribadi.
        </p>
      </div>

      <div className="flex items-start gap-2 border-2 border-blue-600 bg-blue-50 p-3 text-xs font-bold text-blue-800 dark:bg-blue-950/30 dark:text-blue-200">
        <Info className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
        <p>
          Setiap indikator dinilai 0–100 terhadap <em>target</em>, lalu dikalikan <em>bobot</em>. Total bobot wajib 100%.
          Menyimpan membuat versi baru; model lama tetap tersimpan sebagai riwayat.
        </p>
      </div>

      <Card className="border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader>
          <CardTitle className="text-xs font-black uppercase tracking-wide">Identitas Model</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label htmlFor="spk-name" className="text-[10px] font-black uppercase text-muted-foreground">Nama</label>
            <Input id="spk-name" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div>
            <label htmlFor="spk-desc" className="text-[10px] font-black uppercase text-muted-foreground">Deskripsi</label>
            <Input id="spk-desc" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Opsional" />
          </div>
        </CardContent>
      </Card>

      <Card className="border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="text-xs font-black uppercase tracking-wide">Indikator</CardTitle>
            <CardDescription className={weightsValid ? "text-green-700" : "text-red-600"}>
              Total bobot: {(weightSum * 100).toFixed(0)}% {weightsValid ? "✓" : "(harus 100%)"}
            </CardDescription>
          </div>
          <Button type="button" variant="outline" size="sm" className="cursor-pointer" onClick={addIndicator}>
            <Plus className="mr-1.5 h-3.5 w-3.5" /> Tambah Indikator
          </Button>
        </CardHeader>
        <CardContent className="space-y-3">
          {draft.indicators.map((ind, idx) => (
            <div key={idx} className="grid grid-cols-12 items-end gap-2 border-2 border-foreground p-3">
              <div className="col-span-12 sm:col-span-4">
                <label className="text-[10px] font-black uppercase text-muted-foreground" htmlFor={`a-label-${idx}`}>Nama</label>
                <Input id={`a-label-${idx}`} value={ind.label} onChange={(e) => updateIndicator(idx, { label: e.target.value })} />
              </div>
              <div className="col-span-6 sm:col-span-3">
                <label className="text-[10px] font-black uppercase text-muted-foreground" htmlFor={`a-type-${idx}`}>Tipe</label>
                <select
                  id={`a-type-${idx}`}
                  value={ind.type}
                  onChange={(e) => updateIndicator(idx, { type: e.target.value as SpkMetricType })}
                  className="h-9 w-full border-2 border-foreground bg-background px-2 text-sm font-bold"
                >
                  {METRIC_TYPES.map((t) => <option key={t} value={t}>{SPK_METRIC_TYPE_LABELS[t]}</option>)}
                </select>
              </div>
              <div className="col-span-3 sm:col-span-2">
                <label className="text-[10px] font-black uppercase text-muted-foreground" htmlFor={`a-weight-${idx}`}>Bobot</label>
                <Input id={`a-weight-${idx}`} type="number" min={0} max={1} step={0.05} value={ind.weight} onChange={(e) => updateIndicator(idx, { weight: Number(e.target.value) })} />
              </div>
              <div className="col-span-3 sm:col-span-2">
                <label className="text-[10px] font-black uppercase text-muted-foreground" htmlFor={`a-target-${idx}`}>Target</label>
                <Input id={`a-target-${idx}`} type="number" min={0} step={0.05} value={ind.target} onChange={(e) => updateIndicator(idx, { target: Number(e.target.value) })} />
              </div>
              <div className="col-span-12 flex justify-end sm:col-span-1">
                <Button type="button" variant="outline" size="icon" className="cursor-pointer text-red-600" onClick={() => removeIndicator(idx)} aria-label={`Hapus ${ind.label}`}>
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button type="button" className="cursor-pointer" onClick={handleSave} disabled={!weightsValid || save.isPending}>
          <Save className="mr-2 h-4 w-4" /> {save.isPending ? "Menyimpan…" : "Simpan Model Default"}
        </Button>
      </div>
    </div>
  );
}
