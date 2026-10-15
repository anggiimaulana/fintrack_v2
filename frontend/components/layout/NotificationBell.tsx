"use client";

import { useRouter } from "next/navigation";
import { Bell, Check, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useNotificationInbox, type NotificationItem } from "@/hooks/useNotifications";

/** relativeTime renders a compact "baru saja / 5m / 2j / 3h" style label. */
function relativeTime(iso: string, now: number = Date.now()): string {
  const ms = Date.parse(iso);
  if (Number.isNaN(ms)) return "";
  const secs = Math.max(0, Math.floor((now - ms) / 1000));
  if (secs < 60) return "baru saja";
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m lalu`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}j lalu`;
  return `${Math.floor(hours / 24)}h lalu`;
}

/**
 * The header bell. Reads the shared notification cache (no polling); freshness
 * is driven entirely by the single SSE stream subscribed once in
 * `NotificationRealtimeProvider`. Clicking a row marks it read optimistically and
 * navigates to its href.
 */
export function NotificationBell() {
  const router = useRouter();
  const { items, unread, markAllRead, markRead, clear } = useNotificationInbox();

  const openNotification = (n: NotificationItem) => {
    markRead(n.id);
    if (n.href) router.push(n.href);
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={unread > 0 ? `Notifikasi, ${unread} belum dibaca` : "Notifikasi"}
          className="relative p-2 text-muted-foreground hover:text-foreground cursor-pointer"
        >
          <Bell className="w-5 h-5" />
          {unread > 0 && (
            <span className="absolute top-0.5 right-0.5 min-w-4 h-4 px-1 flex items-center justify-center rounded-full bg-red-500 text-white text-[10px] font-black border-2 border-card">
              {unread > 9 ? "9+" : unread}
            </span>
          )}
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent
        align="end"
        className="w-80 p-0 border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground bg-card"
      >
        <div className="flex items-center justify-between px-3 py-2 border-b-2 border-foreground">
          <span className="text-sm font-black uppercase tracking-wide">Notifikasi</span>
          {items.length > 0 && (
            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={() => markAllRead()}
                aria-label="Tandai semua dibaca"
                title="Tandai semua dibaca"
                className="p-1 text-muted-foreground hover:text-foreground cursor-pointer"
              >
                <Check className="w-4 h-4" />
              </button>
              <button
                type="button"
                onClick={() => clear()}
                aria-label="Bersihkan notifikasi"
                title="Bersihkan"
                className="p-1 text-muted-foreground hover:text-red-500 cursor-pointer"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          )}
        </div>

        <div className="max-h-80 overflow-y-auto">
          {items.length === 0 ? (
            <div className="px-4 py-8 text-center">
              <p className="text-sm font-black uppercase tracking-wide text-foreground">Tidak ada notifikasi baru</p>
              <p className="mt-1 text-xs font-bold text-muted-foreground">
                Pemberitahuan pembayaran, perubahan paket, dan reset kuota akan muncul di sini.
              </p>
            </div>
          ) : (
            items.map((n) => (
              <button
                key={n.id}
                type="button"
                onClick={() => openNotification(n)}
                className={`w-full text-left px-3 py-2.5 border-b-2 border-foreground/10 last:border-b-0 transition-colors cursor-pointer hover:bg-muted ${
                  n.read_at ? "" : "bg-primary/5"
                }`}
              >
                <div className="flex items-start gap-2">
                  <span className={`mt-1.5 w-2 h-2 shrink-0 rounded-full ${n.read_at ? "bg-transparent" : "bg-primary"}`} aria-hidden="true" />
                  <div className="min-w-0">
                    <p className="text-sm font-bold text-foreground truncate">{n.title}</p>
                    <p className="text-xs font-medium text-muted-foreground line-clamp-2">{n.body}</p>
                    <p className="text-[10px] font-bold text-muted-foreground/70 mt-0.5">{relativeTime(n.created_at)}</p>
                  </div>
                </div>
              </button>
            ))
          )}
        </div>

        {items.length > 0 && (
          <div className="px-3 py-2 border-t-2 border-foreground">
            <Button
              variant="outline"
              size="sm"
              className="w-full"
              onClick={() => router.push("/user/notifications")}
            >
              Lihat semua notifikasi
            </Button>
          </div>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
