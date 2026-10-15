"use client";

import { useState } from "react";
import { History, Trash2, Mic, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import {
  useVoiceHistory,
  useDeleteVoiceHistory,
  useClearVoiceHistory,
} from "@/hooks/useVoiceHistory";
import { useVoiceConfig } from "@/hooks/useVoiceConfig";

/** Human label for a transcription mode. */
function modeLabel(mode: string): string {
  return mode === "server" ? "AI premium" : "Dasar";
}

function formatWhen(iso: string): string {
  const ms = Date.parse(iso);
  if (Number.isNaN(ms)) return "";
  return new Date(ms).toLocaleString("id-ID", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/**
 * VoiceHistoryPanel shows the user's transcription history (Q-04). It lives
 * inside the chat now that the standalone voice page was merged here (Q-07).
 */
export function VoiceHistoryPanel() {
  const [open, setOpen] = useState(false);
  const [confirmClear, setConfirmClear] = useState(false);
  const { data, isLoading } = useVoiceHistory({ page: 1, perPage: 50, enabled: open });
  const del = useDeleteVoiceHistory();
  const clear = useClearVoiceHistory();
  const { config } = useVoiceConfig();

  const items = data?.data ?? [];
  const retention = config?.history_retention_days;

  return (
    <>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setOpen(true)}
        title="Riwayat suara"
        aria-label="Riwayat suara"
        className="text-muted-foreground hover:text-foreground"
      >
        <History className="w-4 h-4" />
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <History className="w-5 h-5" /> Riwayat Suara
            </DialogTitle>
          </DialogHeader>

          <p className="text-xs font-bold text-muted-foreground">
            {retention ? `Riwayat disimpan ${retention} hari terakhir.` : "Riwayat transkripsi Anda."}
          </p>

          <div className="max-h-96 overflow-y-auto border-2 rounded-md border-foreground divide-y-2 divide-foreground/10">
            {isLoading ? (
              <div className="p-4 text-sm font-bold text-muted-foreground">Memuat…</div>
            ) : items.length === 0 ? (
              <div className="p-6 text-center text-sm font-bold text-muted-foreground">
                Belum ada riwayat suara.
              </div>
            ) : (
              items.map((it) => (
                <div key={it.id} className="flex items-start gap-2 p-3">
                  <span
                    className="mt-0.5 shrink-0"
                    title={modeLabel(it.mode)}
                    aria-label={modeLabel(it.mode)}
                  >
                    {it.mode === "server" ? (
                      <Sparkles className="w-3.5 h-3.5 text-primary" />
                    ) : (
                      <Mic className="w-3.5 h-3.5 text-muted-foreground" />
                    )}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-bold text-foreground break-words">{it.transcript}</p>
                    <p className="text-[10px] font-bold text-muted-foreground mt-0.5">
                      {formatWhen(it.created_at)} · {modeLabel(it.mode)} · {it.language}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => del.mutate(it.id)}
                    aria-label="Hapus riwayat"
                    title="Hapus"
                    disabled={clear.isPending || del.isPending}
                    className="p-1 text-muted-foreground hover:text-red-500 cursor-pointer shrink-0 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              ))
            )}
          </div>

          {items.length > 0 && (
            <div className="flex justify-end">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setConfirmClear(true)}
                disabled={clear.isPending}
              >
                <Trash2 className="w-4 h-4 mr-1.5" /> Bersihkan semua
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={confirmClear}
        onOpenChange={setConfirmClear}
        onConfirm={async () => {
          await clear.mutateAsync();
          setConfirmClear(false);
        }}
        title="Bersihkan Riwayat Suara"
        message="Semua riwayat transkripsi suara akan dihapus. Tindakan ini tidak dapat dibatalkan."
        confirmText="Ya, Bersihkan"
        variant="danger"
        loading={clear.isPending}
      />
    </>
  );
}
