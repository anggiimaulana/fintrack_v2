"use client";

import * as React from "react";
import { AlertTriangle, CalendarClock, CheckCircle2 } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * Remaining-time formatting shared by the countdown and its tests. Kept as a
 * pure function so the wording is verified without a clock.
 */
export function formatRemaining(ms: number): string {
  if (ms <= 0) return "sudah lewat";
  const totalMinutes = Math.floor(ms / 60_000);
  const days = Math.floor(totalMinutes / (60 * 24));
  const hours = Math.floor((totalMinutes % (60 * 24)) / 60);
  const minutes = totalMinutes % 60;
  if (days > 0) return `${days} hari ${hours} jam`;
  if (hours > 0) return `${hours} jam ${minutes} menit`;
  return `${minutes} menit`;
}

export type LifecycleTone = "ok" | "warn" | "danger";

/**
 * The countdown a subscriber needs to act on: how long the current cycle has
 * left, or — once it has ended — how long the grace window remains before the
 * plan lapses. Returns null when there is nothing to show (no end date, or a
 * terminal subscription that is no longer counted down).
 */
export function lifecycleCountdown(
  status: string,
  periodEnd: string | undefined,
  now: number
): { tone: LifecycleTone; label: string; detail: string } | null {
  if (!periodEnd) return null;
  const end = Date.parse(periodEnd);
  if (Number.isNaN(end)) return null;

  const terminal = status === "expired" || status === "canceled";
  if (terminal) return null;

  const remaining = end - now;
  if (remaining > 0) {
    const days = Math.ceil(remaining / 86_400_000);
    return {
      tone: days <= 3 ? "warn" : "ok",
      label: days <= 3 ? `Berakhir dalam ${formatRemaining(remaining)}` : "Siklus berjalan",
      detail: "Perpanjang sebelum tanggal berakhir agar paket tidak turun.",
    };
  }

  // Past the cycle end: the subscription is inside the grace window.
  return {
    tone: "danger",
    label: `Masa tenggang — ${formatRemaining(Math.abs(remaining))} tersisa`,
    detail:
      status === "past_due"
        ? "Pembayaran belum diterima. Paket akan turun otomatis setelah masa tenggang."
        : "Siklus berakhir. Bayar tagihan agar paket tetap aktif.",
  };
}

/**
 * A 60-second ticking clock exposed as an external store, so the component
 * reads time without a setState-in-effect (which the lint rules forbid) and the
 * server snapshot stays null — the server has no clock the client must agree
 * with, so the first client render is a neutral placeholder.
 */
const CLOCK_INTERVAL_MS = 60_000;
const clockListeners = new Set<() => void>();
let clockTimer: number | null = null;

function subscribeClock(onChange: () => void) {
  clockListeners.add(onChange);
  if (clockTimer === null) {
    clockTimer = window.setInterval(() => {
      for (const listener of clockListeners) listener();
    }, CLOCK_INTERVAL_MS);
  }
  return () => {
    clockListeners.delete(onChange);
    if (clockListeners.size === 0 && clockTimer !== null) {
      window.clearInterval(clockTimer);
      clockTimer = null;
    }
  };
}

function getClientNow(): number {
  return Date.now();
}

function getServerNow(): number | null {
  return null;
}

interface LifecycleCountdownProps {
  status: string;
  periodEnd?: string;
  cancelAtPeriodEnd?: boolean;
  className?: string;
}

/**
 * Shows a live countdown for the subscription cycle / grace window. The clock is
 * client-only (the server snapshot is null) so the server and the first client
 * render agree — the same discipline used elsewhere in this app to avoid
 * hydration mismatches.
 */
export function LifecycleCountdown({ status, periodEnd, cancelAtPeriodEnd, className }: LifecycleCountdownProps) {
  const now = React.useSyncExternalStore(subscribeClock, getClientNow, getServerNow);

  // Until mounted, render a neutral placeholder (no clock-dependent text).
  if (now === null) {
    return (
      <p className={cn("flex items-center gap-1.5 text-xs font-bold text-muted-foreground", className)}>
        <CalendarClock className="h-3.5 w-3.5" aria-hidden="true" />
        Memuat status langganan…
      </p>
    );
  }

  const info = lifecycleCountdown(status, periodEnd, now);
  if (!info) {
    if (status === "expired") {
      return (
        <p className={cn("flex items-center gap-1.5 text-xs font-bold text-muted-foreground", className)}>
          <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
          Langganan berakhir. Pilih paket untuk mengaktifkan kembali.
        </p>
      );
    }
    if (status === "canceled") {
      return (
        <p className={cn("flex items-center gap-1.5 text-xs font-bold text-muted-foreground", className)}>
          <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
          Langganan dibatalkan.
        </p>
      );
    }
    return null;
  }

  const toneClass: Record<LifecycleTone, string> = {
    ok: "text-muted-foreground",
    warn: "text-amber-700 dark:text-amber-400",
    danger: "text-destructive",
  };
  const Icon = info.tone === "ok" ? CheckCircle2 : AlertTriangle;

  return (
    <div className={cn("space-y-0.5", className)}>
      <p className={cn("flex items-center gap-1.5 text-xs font-bold", toneClass[info.tone])}>
        <Icon className="h-3.5 w-3.5" aria-hidden="true" />
        {info.label}
        {cancelAtPeriodEnd && <span className="font-normal">· tidak diperpanjang</span>}
      </p>
      <p className="text-[11px] font-bold text-muted-foreground">{info.detail}</p>
    </div>
  );
}
