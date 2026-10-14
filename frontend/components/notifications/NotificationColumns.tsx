"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { Eye, MailOpen, Trash2, Bell, CreditCard, Package, RefreshCw, Sparkles } from "lucide-react";
import { SortableHeader } from "@/components/data-table/data-table-column-header";
import type { NotificationItem } from "@/hooks/useNotifications";

/** A colored icon + label per notification event, so the row reads at a glance. */
function eventMeta(event: string): { icon: React.ElementType; label: string; className: string } {
  switch (event) {
    case "payment_received":
      return { icon: CreditCard, label: "Pembayaran", className: "bg-emerald-100 text-emerald-800 border-emerald-800/40" };
    case "plan_updated":
      return { icon: Package, label: "Paket", className: "bg-violet-100 text-violet-800 border-violet-800/40" };
    case "quota_reset":
      return { icon: RefreshCw, label: "Kuota", className: "bg-blue-100 text-blue-800 border-blue-800/40" };
    case "manual_approval_needed":
      return { icon: Sparkles, label: "Verifikasi", className: "bg-amber-100 text-amber-800 border-amber-800/40" };
    default:
      return { icon: Bell, label: "Info", className: "bg-muted text-foreground border-foreground/40" };
  }
}

function formatWhen(iso: string): { date: string; time: string } {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return { date: "-", time: "" };
  return {
    date: d.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" }),
    time: d.toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" }),
  };
}

export interface NotificationColumnsOptions {
  /** 1-based page number, used for absolute row numbering. */
  page?: number;
  perPage?: number;
  onDetail?: (n: NotificationItem) => void;
  onMarkRead?: (n: NotificationItem) => void;
  onDelete?: (id: string) => void;
}

/**
 * Column definitions for the notifications table.
 *
 * Sorting is delegated to the server: the column ids match the API's accepted
 * `sort` values (created_at | title | event).
 */
export function notificationColumns({
  page = 1,
  perPage = 20,
  onDetail,
  onMarkRead,
  onDelete,
}: NotificationColumnsOptions): ColumnDef<NotificationItem, unknown>[] {
  return [
    {
      id: "index",
      header: "#",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="font-mono text-xs font-bold text-muted-foreground">
          {(page - 1) * perPage + row.index + 1}
        </span>
      ),
    },
    {
      id: "title",
      accessorFn: (n) => n.title,
      header: (ctx) => <SortableHeader title="Notifikasi" columnId="title" ctx={ctx} />,
      cell: ({ row }) => {
        const n = row.original;
        const unread = !n.read_at;
        return (
          <div className="flex min-w-0 items-start gap-2.5">
            <span
              className={`mt-0.5 h-2 w-2 shrink-0 rounded-full ${unread ? "bg-primary" : "bg-transparent"}`}
              aria-label={unread ? "Belum dibaca" : "Sudah dibaca"}
              title={unread ? "Belum dibaca" : "Sudah dibaca"}
            />
            <div className="min-w-0">
              <p className={`truncate text-sm ${unread ? "font-black text-foreground" : "font-bold text-muted-foreground"}`}>
                {n.title}
              </p>
              <p className="mt-0.5 line-clamp-2 text-xs font-medium text-muted-foreground">{n.body}</p>
            </div>
          </div>
        );
      },
    },
    {
      id: "event",
      accessorFn: (n) => n.event,
      header: (ctx) => <SortableHeader title="Jenis" columnId="event" ctx={ctx} />,
      cell: ({ row }) => {
        const meta = eventMeta(row.original.event);
        const Icon = meta.icon;
        return (
          <span className={`inline-flex items-center gap-1.5 border-2 px-2 py-0.5 text-xs font-black uppercase tracking-wider ${meta.className}`}>
            <Icon className="h-3.5 w-3.5" aria-hidden="true" />
            {meta.label}
          </span>
        );
      },
    },
    {
      id: "status",
      header: () => <span className="font-black uppercase tracking-wide text-xs">Status</span>,
      enableSorting: false,
      cell: ({ row }) => {
        const read = !!row.original.read_at;
        return (
          <span
            className={`inline-block border-2 px-2 py-0.5 text-xs font-black uppercase tracking-wider ${
              read
                ? "border-foreground/30 bg-muted text-muted-foreground"
                : "border-primary bg-primary/10 text-primary"
            }`}
          >
            {read ? "Dibaca" : "Belum"}
          </span>
        );
      },
    },
    {
      id: "created_at",
      accessorFn: (n) => n.created_at,
      header: (ctx) => <SortableHeader title="Waktu" columnId="created_at" ctx={ctx} />,
      cell: ({ row }) => {
        const { date, time } = formatWhen(row.original.created_at);
        return (
          <div className="text-xs font-bold text-foreground">
            <div>{date}</div>
            <div className="font-mono text-muted-foreground">{time}</div>
          </div>
        );
      },
    },
    {
      id: "actions",
      header: () => <span className="block text-center">Aksi</span>,
      enableSorting: false,
      cell: ({ row }) => {
        const n = row.original;
        return (
          <div className="flex items-center justify-center gap-1.5">
            {onDetail && (
              <button
                type="button"
                onClick={() => onDetail(n)}
                aria-label="Lihat notifikasi"
                title="Lihat"
                className="cursor-pointer border-2 border-foreground bg-card p-1.5 text-foreground transition-all hover:bg-muted shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[1px] active:translate-y-[1px]"
              >
                <Eye className="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            )}
            {onMarkRead && !n.read_at && (
              <button
                type="button"
                onClick={() => onMarkRead(n)}
                aria-label="Tandai sudah dibaca"
                title="Tandai sudah dibaca"
                className="cursor-pointer border-2 border-emerald-600 bg-emerald-50 p-1.5 text-emerald-600 transition-all hover:bg-emerald-600 hover:text-white shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[1px] active:translate-y-[1px]"
              >
                <MailOpen className="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            )}
            {onDelete && (
              <button
                type="button"
                onClick={() => onDelete(n.id)}
                aria-label="Hapus notifikasi"
                title="Hapus"
                className="cursor-pointer border-2 border-red-600 bg-red-50 p-1.5 text-red-600 transition-all hover:bg-red-600 hover:text-white shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[1px] active:translate-y-[1px]"
              >
                <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            )}
          </div>
        );
      },
    },
  ];
}

export { eventMeta };
