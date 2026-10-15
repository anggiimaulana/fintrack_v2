"use client";

import * as React from "react";
import { toast } from "sonner";
import { Plus, Trash2, RotateCcw } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  useResetMySpkModel,
  useSaveMySpkModel,
  SPK_METRIC_TYPE_LABELS,
  type SpkMetricIndicator,
  type SpkMetricModel,
  type SpkMetricType,
} from "@/hooks/useSpkMetrics";

const METRIC_TYPES = Object.keys(SPK_METRIC_TYPE_LABELS) as SpkMetricType[];

interface SpkMetricEditorProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  current: SpkMetricModel;
  isOverride: boolean;
}

/** Deep-copies a model so edits do not mutate the query cache. */
function cloneModel(m: SpkMetricModel): SpkMetricModel {
  return {
    indicators: m.indicators.map((i) => ({ ...i, categories: i.categories ? [...i.categories] : undefined })),
    status_thresholds: m.status_thresholds.map((t) => ({ ...t })),
    source: m.source,
    name: m.name,
  };
}

/**
 * Editor for the signed-in user's SPK metric model. Saving stores a personal
 * override; "Kembalikan ke Default" deletes it so the super-admin default
 * applies again.
 */
export function SpkMetricEditor({ open, onOpenChange, current, isOverride }: SpkMetricEditorProps) {
  const [draft, setDraft] = React.useState<SpkMetricModel>(() => cloneModel(current));
  const save = useSaveMySpkModel();
  const reset = useResetMySpkModel();

  // Reset the draft whenever the dialog opens so it always starts from the
  // saved model (never a stale edit from a previous open).
  const handleOpenChange = (next: boolean) => {
    if (next) setDraft(cloneModel(current));
    onOpenChange(next);
  };

  const weightSum = draft.indicators.reduce((s, i) => s + (Number(i.weight) || 0), 0);
  const weightsValid = Math.abs(weightSum - 1) < 1e-6;

  const updateIndicator = (idx: number, patch: Partial<SpkMetricIndicator>) => {
    setDraft((d) => ({
      ...d,
      indicators: d.indicators.map((ind, i) => (i === idx ? { ...ind, ...patch } : ind)),
    }));
  };

  const addIndicator = () => {
    setDraft((d) => ({
      ...d,
      indicators: [
        ...d.indicators,
        { key: `indikator_${d.indicators.length + 1}`, label: "Indikator Baru", weight: 0, target: 1, type: "consistency" },
      ],
    }));
  };

  const removeIndicator = (idx: number) => {
    setDraft((d) => ({ ...d, indicators: d.indicators.filter((_, i) => i !== idx) }));
  };

  const handleSave = async () => {
    if (!weightsValid) {
      toast.error("Total bobot harus tepat 1.0 (100%)");
      return;
    }
    try {
      await save.mutateAsync({ indicators: draft.indicators, status_thresholds: draft.status_thresholds });
      toast.success("Metrik SPK Anda disimpan");
      onOpenChange(false);
    } catch (e) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message ?? "Gagal menyimpan metrik";
      toast.error(msg);
    }
  };

  const handleReset = async () => {
    try {
      await reset.mutateAsync();
      toast.success("Dikembalikan ke metrik default");
      onOpenChange(false);
    } catch {
      toast.error("Gagal mengembalikan ke default");
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Atur Metrik SPK</DialogTitle>
          <DialogDescription>
            Sesuaikan indikator dan bobot penilaian. Perubahan berlaku untuk perhitungan skor Anda berikutnya dan
            menimpa metrik default.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className={`text-xs font-black uppercase tracking-wide ${weightsValid ? "text-green-700" : "text-red-600"}`}>
              Total bobot: {(weightSum * 100).toFixed(0)}% {weightsValid ? "✓" : "(harus 100%)"}
            </span>
            <Button type="button" variant="outline" size="sm" className="cursor-pointer" onClick={addIndicator}>
              <Plus className="mr-1.5 h-3.5 w-3.5" /> Tambah
            </Button>
          </div>

          <div className="space-y-3">
            {draft.indicators.map((ind, idx) => (
              <div key={idx} className="grid grid-cols-12 items-end gap-2 border-2 rounded-md border-foreground p-3">
                <div className="col-span-12 sm:col-span-4">
                  <label className="text-[10px] font-black uppercase text-muted-foreground" htmlFor={`ind-label-${idx}`}>
                    Nama
                  </label>
                  <Input
                    id={`ind-label-${idx}`}
                    value={ind.label}
                    onChange={(e) => updateIndicator(idx, { label: e.target.value })}
                  />
                </div>
                <div className="col-span-6 sm:col-span-3">
                  <label className="text-[10px] font-black uppercase text-muted-foreground" htmlFor={`ind-type-${idx}`}>
                    Tipe
                  </label>
                  <select
                    id={`ind-type-${idx}`}
                    value={ind.type}
                    onChange={(e) => updateIndicator(idx, { type: e.target.value as SpkMetricType })}
                    className="h-9 w-full border-2 rounded-md border-foreground bg-background px-2 text-sm font-bold"
                  >
                    {METRIC_TYPES.map((t) => (
                      <option key={t} value={t}>{SPK_METRIC_TYPE_LABELS[t]}</option>
                    ))}
                  </select>
                </div>
                <div className="col-span-3 sm:col-span-2">
                  <label className="text-[10px] font-black uppercase text-muted-foreground" htmlFor={`ind-weight-${idx}`}>
                    Bobot
                  </label>
                  <Input
                    id={`ind-weight-${idx}`}
                    type="number"
                    min={0}
                    max={1}
                    step={0.05}
                    value={ind.weight}
                    onChange={(e) => updateIndicator(idx, { weight: Number(e.target.value) })}
                  />
                </div>
                <div className="col-span-3 sm:col-span-2">
                  <label className="text-[10px] font-black uppercase text-muted-foreground" htmlFor={`ind-target-${idx}`}>
                    Target
                  </label>
                  <Input
                    id={`ind-target-${idx}`}
                    type="number"
                    min={0}
                    step={0.05}
                    value={ind.target}
                    onChange={(e) => updateIndicator(idx, { target: Number(e.target.value) })}
                  />
                </div>
                <div className="col-span-12 flex justify-end sm:col-span-1">
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="cursor-pointer text-red-600"
                    onClick={() => removeIndicator(idx)}
                    aria-label={`Hapus ${ind.label}`}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>

        <DialogFooter className="flex-col gap-2 sm:flex-row sm:justify-between">
          {isOverride && (
            <Button
              type="button"
              variant="outline"
              className="cursor-pointer"
              onClick={handleReset}
              disabled={reset.isPending}
            >
              <RotateCcw className="mr-1.5 h-3.5 w-3.5" /> Kembalikan ke Default
            </Button>
          )}
          <div className="flex gap-2">
            <Button type="button" variant="outline" className="cursor-pointer" onClick={() => onOpenChange(false)}>
              Batal
            </Button>
            <Button type="button" className="cursor-pointer" onClick={handleSave} disabled={!weightsValid || save.isPending}>
              {save.isPending ? "Menyimpan…" : "Simpan"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
