"use client";

import * as React from "react";
import Link from "next/link";
import { Gauge, ArrowRight, Zap, Brain, Sparkles, Clock, AlertTriangle } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useEntitlements } from "@/hooks/use-entitlements";

interface AiQuotaDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AiQuotaDialog({ open, onOpenChange }: AiQuotaDialogProps) {
  const { planName, period, usage } = useEntitlements();

  const tokenUsage = usage?.["ai_tokens_per_cycle"];
  const requestUsage = usage?.["ai_requests_per_cycle"];

  const tokenUsed = tokenUsage?.used ?? 0;
  const tokenLimit = tokenUsage?.limit ?? 0;
  const tokenRemaining = tokenUsage?.unlimited ? -1 : Math.max(0, tokenUsage?.remaining ?? 0);
  const tokenPct =
    tokenUsage?.unlimited || tokenLimit <= 0
      ? 0
      : Math.min(100, Math.round((tokenUsed / tokenLimit) * 100));

  const reqUsed = requestUsage?.used ?? 0;
  const reqLimit = requestUsage?.limit ?? 0;
  const reqRemaining = requestUsage?.unlimited ? -1 : Math.max(0, requestUsage?.remaining ?? 0);
  const reqPct =
    requestUsage?.unlimited || reqLimit <= 0
      ? 0
      : Math.min(100, Math.round((reqUsed / reqLimit) * 100));

  const periodEndFormatted = period?.end
    ? new Date(period.end).toLocaleDateString("id-ID", {
        day: "numeric",
        month: "long",
        year: "numeric",
      })
    : null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="border-2 rounded-md border-foreground bg-card shadow-[6px_6px_0px_0px] shadow-foreground sm:max-w-md max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base font-black uppercase tracking-wide">
            <Gauge className="w-5 h-5 text-primary" /> Informasi Kuota AI
          </DialogTitle>
          <DialogDescription className="text-xs font-bold text-muted-foreground">
            Status sisa kuota dan pemakaian fitur AI pada siklus aktif Anda.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 my-2">
          {/* Plan & Cycle header */}
          <div className="flex items-center justify-between border-2 rounded-md border-foreground bg-muted p-3">
            <div>
              <p className="text-[10px] font-black uppercase text-muted-foreground">Paket Saat Ini</p>
              <div className="flex items-center gap-2 mt-0.5">
                <span className="text-sm font-black text-foreground">Paket {planName || "Pro"}</span>
                <Badge variant="outline" className="text-[10px] uppercase font-black bg-primary/10 text-primary border-primary">
                  Aktif
                </Badge>
              </div>
            </div>
            {periodEndFormatted && (
              <div className="text-right">
                <p className="text-[10px] font-black uppercase text-muted-foreground">Reset Siklus</p>
                <p className="text-xs font-bold font-mono text-muted-foreground mt-0.5 flex items-center justify-end gap-1">
                  <Clock className="w-3 h-3" /> {periodEndFormatted}
                </p>
              </div>
            )}
          </div>

          {/* AI Tokens / Kuota */}
          <div className="border-2 rounded-md border-foreground bg-card p-3.5 space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-black uppercase tracking-wide flex items-center gap-1.5 text-foreground">
                <Sparkles className="w-3.5 h-3.5 text-primary" /> Kuota Token AI
              </span>
              <span className="font-mono text-xs font-black">
                {tokenUsage?.unlimited
                  ? "Unlimited"
                  : `${tokenRemaining.toLocaleString("id-ID")} tersisa`}
              </span>
            </div>

            {!tokenUsage?.unlimited ? (
              <>
                <div className="h-2.5 w-full border-2 rounded-md border-foreground bg-muted overflow-hidden">
                  <div
                    className={`h-full transition-all duration-300 ${
                      tokenRemaining <= 0
                        ? "bg-destructive"
                        : tokenPct >= 80
                          ? "bg-amber-500"
                          : "bg-primary"
                    }`}
                    style={{ width: `${tokenPct}%` }}
                  />
                </div>
                <div className="flex justify-between text-[11px] font-bold text-muted-foreground">
                  <span>Terpakai {tokenUsed.toLocaleString("id-ID")} ({tokenPct}%)</span>
                  <span>Batas {tokenLimit.toLocaleString("id-ID")} kuota</span>
                </div>
              </>
            ) : (
              <p className="text-xs font-bold text-muted-foreground">
                Paket Anda memiliki akses kuota token tanpa batas.
              </p>
            )}

            {tokenRemaining <= 0 && !tokenUsage?.unlimited && (
              <div className="flex items-center gap-1.5 p-2 border rounded-md border-destructive bg-destructive/10 text-destructive text-[11px] font-bold mt-2">
                <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
                <span>Kuota token telah habis untuk siklus ini.</span>
              </div>
            )}
          </div>

          {/* AI Requests (if present) */}
          {requestUsage && (
            <div className="border-2 rounded-md border-foreground bg-card p-3.5 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs font-black uppercase tracking-wide flex items-center gap-1.5 text-foreground">
                  <Gauge className="w-3.5 h-3.5 text-primary" /> Permintaan AI
                </span>
                <span className="font-mono text-xs font-black">
                  {requestUsage.unlimited
                    ? "Unlimited"
                    : `${reqRemaining.toLocaleString("id-ID")} tersisa`}
                </span>
              </div>

              {!requestUsage.unlimited ? (
                <>
                  <div className="h-2.5 w-full border-2 rounded-md border-foreground bg-muted overflow-hidden">
                    <div
                      className={`h-full transition-all duration-300 ${
                        reqRemaining <= 0
                          ? "bg-destructive"
                          : reqPct >= 80
                            ? "bg-amber-500"
                            : "bg-primary"
                      }`}
                      style={{ width: `${reqPct}%` }}
                    />
                  </div>
                  <div className="flex justify-between text-[11px] font-bold text-muted-foreground">
                    <span>Terpakai {reqUsed.toLocaleString("id-ID")} ({reqPct}%)</span>
                    <span>Batas {reqLimit.toLocaleString("id-ID")} permintaan</span>
                  </div>
                </>
              ) : (
                <p className="text-xs font-bold text-muted-foreground">
                  Bebas melakukan permintaan AI tanpa batas.
                </p>
              )}
            </div>
          )}

          {/* AI Mode Info */}
          <div className="border-2 rounded-md border-foreground/30 bg-muted/40 p-3 space-y-1.5 text-[11px] font-bold text-muted-foreground">
            <p className="font-black text-foreground uppercase tracking-wide text-[10px]">Konsumsi Mode AI</p>
            <p className="flex items-start gap-1.5">
              <Zap className="w-3.5 h-3.5 text-primary shrink-0 mt-0.5" />
              <span><strong>Mode Cepat</strong>: Dirancang untuk respon cepat dan hemat kuota token.</span>
            </p>
            <p className="flex items-start gap-1.5">
              <Brain className="w-3.5 h-3.5 text-primary shrink-0 mt-0.5" />
              <span><strong>Mode Penalaran</strong>: Analisis mendalam bertahap, menggunakan lebih banyak token per jawaban.</span>
            </p>
          </div>
        </div>

        <div className="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 pt-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            className="cursor-pointer font-bold"
          >
            Tutup
          </Button>
          <Button asChild className="cursor-pointer font-bold shadow-[2px_2px_0px_0px] shadow-foreground">
            <Link href="/user/usage">
              Lihat Detail Penggunaan <ArrowRight className="w-4 h-4 ml-1.5" />
            </Link>
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
