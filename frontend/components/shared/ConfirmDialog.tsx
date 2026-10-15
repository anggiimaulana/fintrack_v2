"use client";

import { AlertTriangle } from "lucide-react";

interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: "danger" | "primary";
  loading?: boolean;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  title,
  message,
  confirmText = "Ya, Hapus",
  cancelText = "Batal",
  variant = "danger",
  loading = false,
}: ConfirmDialogProps) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      {/* Backdrop: a sibling behind the dialog, so a click inside the dialog
          never reaches it and no stopPropagation plumbing is needed. */}
      <div
        role="presentation"
        aria-hidden="true"
        className="absolute inset-0 bg-black/60"
        onClick={() => !loading && onOpenChange(false)}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Konfirmasi tindakan"
        className="relative border-2 rounded-md border-foreground bg-card shadow-[6px_6px_0px_0px] shadow-foreground w-full max-w-md"
      >
        <div className="p-6 text-center">
          <div className={`w-14 h-14 border-2 rounded-md border-foreground flex items-center justify-center mx-auto mb-4 ${
            variant === "danger" ? "bg-red-100" : "bg-primary/10"
          }`}>
            <AlertTriangle className={`w-7 h-7 ${variant === "danger" ? "text-red-600" : "text-primary"}`} />
          </div>
          <h3 className="text-base font-black uppercase tracking-wide text-foreground mb-2">{title}</h3>
          <p className="text-sm font-bold text-muted-foreground">{message}</p>
        </div>
        <div className="flex border-t-2 border-foreground">
          <button
            onClick={() => !loading && onOpenChange(false)}
            disabled={loading}
            className="flex-1 py-3.5 text-sm font-bold text-muted-foreground hover:text-foreground hover:bg-muted border-r-2 border-foreground transition-colors disabled:opacity-50 cursor-pointer"
          >
            {cancelText}
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={`flex-1 py-3.5 text-sm font-bold text-white transition-colors disabled:opacity-50 cursor-pointer ${
              variant === "danger" ? "bg-red-600 hover:bg-red-700" : "bg-primary hover:bg-primary/90"
            }`}
          >
            {loading ? "Memproses..." : confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}
