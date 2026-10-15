"use client";

import { Loader2 } from "lucide-react";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { useNotificationPreferences, useSendTestEmail, type NotificationPreferences } from "@/hooks/useNotifications";

const NOTIFICATION_ITEMS: { key: keyof NotificationPreferences; label: string; desc: string }[] = [
  { key: "transaction_email", label: "Email Notifikasi Transaksi", desc: "Dapatkan notifikasi email setiap ada transaksi baru" },
  { key: "category_reminder", label: "Pengingat Kategori", desc: "Pengingat jika pengeluaran mendekati batas kategori" },
  { key: "weekly_report", label: "Laporan Mingguan", desc: "Ringkasan keuangan mingguan via email" },
];

/**
 * Notification preferences, persisted server-side. Each toggle really controls
 * email delivery — the backend writes every event to the in-app inbox and only
 * emails when the matching preference is on. A test button verifies delivery.
 */
export function NotificationSettings() {
  const { preferences, isLoading, isSaving, save } = useNotificationPreferences();
  const sendTest = useSendTestEmail();

  if (isLoading || !preferences) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-12 border-2 rounded-md border-foreground" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-1">
      {NOTIFICATION_ITEMS.map((item) => (
        <div key={item.key} className="flex items-center justify-between gap-4 py-2.5">
          <div className="min-w-0">
            <p className="text-sm font-bold text-foreground">{item.label}</p>
            <p className="text-xs font-bold text-muted-foreground">{item.desc}</p>
          </div>
          <Switch
            aria-label={item.label}
            disabled={isSaving}
            checked={preferences[item.key]}
            onCheckedChange={(checked) => save({ ...preferences, [item.key]: checked })}
          />
        </div>
      ))}

      <div className="flex flex-col gap-2 pt-3 border-t-2 border-foreground/10 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-[11px] font-bold text-muted-foreground">
          Notifikasi dalam aplikasi selalu tercatat; email mengikuti preferensi di atas.
        </p>
        <button
          type="button"
          onClick={() => sendTest.mutate()}
          disabled={sendTest.isPending}
          className="shrink-0 text-xs font-bold text-primary hover:text-primary/80 flex items-center gap-1.5 cursor-pointer disabled:cursor-not-allowed disabled:opacity-60"
        >
          {sendTest.isPending && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
          Kirim email uji
        </button>
      </div>
    </div>
  );
}
