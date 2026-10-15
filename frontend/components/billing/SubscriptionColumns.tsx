"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { CheckCircle2, Clock, XCircle, CreditCard, Ban } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Money } from "@/components/shared/Money";
import { SortableHeader } from "@/components/data-table/data-table-column-header";
import { PayNowDialog } from "@/components/billing/PaymentRow";
import type { Invoice, UserPayment } from "@/hooks/use-billing";
import { formatTanggal } from "@/lib/format";
import { formatCountdown, isCancellable, remainingMs, urgencyFor } from "@/lib/invoice-expiry";

// ─── Payments ─────────────────────────────────────────────────

const PAYMENT_STATUS_LABEL: Record<UserPayment["status"], string> = {
  pending: "Menunggu pembayaran",
  pending_review: "Menunggu verifikasi",
  settled: "Berhasil",
  failed: "Gagal",
  expired: "Kedaluwarsa",
  refunded: "Dikembalikan",
};

const PAYMENT_STATUS_VARIANT: Record<UserPayment["status"], "default" | "secondary" | "destructive" | "outline"> = {
  pending: "secondary",
  pending_review: "outline",
  settled: "default",
  failed: "destructive",
  expired: "destructive",
  refunded: "secondary",
};

export const PAYMENT_STATUS_OPTIONS = [
  { value: "", label: "Semua status" },
  { value: "pending", label: "Menunggu pembayaran" },
  { value: "pending_review", label: "Menunggu verifikasi" },
  { value: "settled", label: "Berhasil" },
  { value: "failed", label: "Gagal" },
  { value: "expired", label: "Kedaluwarsa" },
  { value: "refunded", label: "Dikembalikan" },
];

/** A "Bayar Sekarang" action cell that owns its own payment dialog. */
function PaymentActionCell({ payment }: { payment: UserPayment }) {
  const [open, setOpen] = React.useState(false);
  if (payment.status !== "pending") {
    return <span className="text-xs font-bold text-muted-foreground">—</span>;
  }
  return (
    <>
      <Button size="sm" className="cursor-pointer" onClick={() => setOpen(true)}>
        <CreditCard className="mr-1.5 h-3.5 w-3.5" /> Bayar
      </Button>
      <PayNowDialog payment={payment} open={open} onOpenChange={setOpen} />
    </>
  );
}

export function paymentColumns({
  page = 1,
  perPage = 20,
}: {
  page?: number;
  perPage?: number;
} = {}): ColumnDef<UserPayment, unknown>[] {
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
      id: "invoice_number",
      header: () => <span className="font-black uppercase tracking-wide text-xs">Faktur</span>,
      enableSorting: false,
      cell: ({ row }) => (
        <div className="min-w-0">
          <p className="truncate text-sm font-bold text-foreground">{row.original.plan_name || row.original.plan_code || "Langganan"}</p>
          <p className="font-mono text-xs font-bold text-muted-foreground">{row.original.invoice_number || "—"}</p>
        </div>
      ),
    },
    {
      id: "created_at",
      accessorFn: (p) => p.created_at,
      header: (ctx) => <SortableHeader title="Dibuat" columnId="created_at" ctx={ctx} />,
      cell: ({ row }) => <span className="text-xs font-bold text-muted-foreground">{formatTanggal(row.original.created_at)}</span>,
    },
    {
      id: "amount_idr",
      accessorFn: (p) => p.amount_idr,
      header: (ctx) => <SortableHeader title="Jumlah" columnId="amount_idr" ctx={ctx} className="justify-end" />,
      cell: ({ row }) => <Money value={row.original.amount_idr} className="text-sm font-black" />,
      meta: { align: "right" },
    },
    {
      id: "status",
      accessorFn: (p) => p.status,
      header: (ctx) => <SortableHeader title="Status" columnId="status" ctx={ctx} />,
      cell: ({ row }) => {
        const s = row.original.status;
        const icon =
          s === "settled" ? (
            <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
          ) : s === "pending" || s === "pending_review" ? (
            <Clock className="h-3.5 w-3.5" aria-hidden="true" />
          ) : (
            <XCircle className="h-3.5 w-3.5" aria-hidden="true" />
          );
        return (
          <Badge variant={PAYMENT_STATUS_VARIANT[s]} className="gap-1">
            {icon}
            {PAYMENT_STATUS_LABEL[s]}
          </Badge>
        );
      },
    },
    {
      id: "actions",
      header: () => <span className="block text-center">Aksi</span>,
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex justify-center">
          <PaymentActionCell payment={row.original} />
        </div>
      ),
    },
  ];
}

// ─── Invoices ─────────────────────────────────────────────────

const INVOICE_STATUS_LABEL: Record<Invoice["status"], string> = {
  draft: "Draf",
  issued: "Terbit",
  paid: "Lunas",
  past_due: "Jatuh tempo",
  voided: "Dibatalkan",
  expired: "Kedaluwarsa",
};

export const INVOICE_STATUS_OPTIONS = [
  { value: "", label: "Semua status" },
  { value: "issued", label: "Terbit" },
  { value: "paid", label: "Lunas" },
  { value: "past_due", label: "Jatuh tempo" },
  { value: "voided", label: "Dibatalkan" },
  { value: "expired", label: "Kedaluwarsa" },
];

/** A "Batalkan" action cell that owns its own confirmation dialog. */
function InvoiceActionCell({ invoice, onCancel }: { invoice: Invoice; onCancel: (id: string) => void }) {
  if (!isCancellable(invoice.status)) {
    return <span className="text-xs font-bold text-muted-foreground">—</span>;
  }
  return (
    <Button
      variant="outline"
      size="sm"
      className="cursor-pointer text-red-600"
      onClick={() => onCancel(invoice.id)}
    >
      <Ban className="mr-1.5 h-3.5 w-3.5" /> Batalkan
    </Button>
  );
}

export function invoiceColumns({
  page = 1,
  perPage = 20,
  onCancel,
}: {
  page?: number;
  perPage?: number;
  onCancel?: (id: string) => void;
} = {}): ColumnDef<Invoice, unknown>[] {
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
      id: "invoice_number",
      accessorFn: (i) => i.invoice_number,
      header: (ctx) => <SortableHeader title="No. Faktur" columnId="invoice_number" ctx={ctx} />,
      cell: ({ row }) => <span className="font-mono text-sm font-bold text-foreground">{row.original.invoice_number}</span>,
    },
    {
      id: "issued_at",
      accessorFn: (i) => i.issued_at,
      header: (ctx) => <SortableHeader title="Terbit" columnId="issued_at" ctx={ctx} />,
      cell: ({ row }) => <span className="text-xs font-bold text-muted-foreground">{formatTanggal(row.original.issued_at)}</span>,
    },
    {
      id: "due_at",
      accessorFn: (i) => i.due_at ?? "",
      header: (ctx) => <SortableHeader title="Jatuh Tempo" columnId="due_at" ctx={ctx} />,
      cell: ({ row }) => {
        const inv = row.original;
        const cancellable = isCancellable(inv.status);
        const ms = remainingMs(inv.due_at);
        const urgency = urgencyFor(ms);
        const cls = urgency === "critical" ? "text-red-600" : urgency === "warning" ? "text-amber-600" : "text-muted-foreground";
        return (
          <div className="text-xs font-bold text-muted-foreground">
            <div>{inv.due_at ? formatTanggal(inv.due_at) : "—"}</div>
            {cancellable && inv.due_at && (
              <span className={`flex items-center gap-1 ${cls}`}>
                <Clock className="h-3 w-3" aria-hidden="true" />
                {formatCountdown(ms)} lagi
              </span>
            )}
          </div>
        );
      },
    },
    {
      id: "total_idr",
      accessorFn: (i) => i.total_idr,
      header: (ctx) => <SortableHeader title="Total" columnId="total_idr" ctx={ctx} className="justify-end" />,
      cell: ({ row }) => (
        <div className="text-right">
          <Money value={row.original.total_idr} className="text-sm font-black" />
          {row.original.tax_idr > 0 && (
            <span className="block text-[10px] font-bold text-muted-foreground">
              incl. pajak <Money value={row.original.tax_idr} />
            </span>
          )}
        </div>
      ),
      meta: { align: "right" },
    },
    {
      id: "status",
      accessorFn: (i) => i.status,
      header: (ctx) => <SortableHeader title="Status" columnId="status" ctx={ctx} />,
      cell: ({ row }) => (
        <Badge
          variant={
            row.original.status === "paid"
              ? "default"
              : row.original.status === "voided" || row.original.status === "expired"
                ? "destructive"
                : "secondary"
          }
        >
          {INVOICE_STATUS_LABEL[row.original.status]}
        </Badge>
      ),
    },
    {
      id: "actions",
      header: () => <span className="block text-center">Aksi</span>,
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex justify-center">
          <InvoiceActionCell invoice={row.original} onCancel={(id) => onCancel?.(id)} />
        </div>
      ),
    },
  ];
}

// ─── Subscription lifecycle events ────────────────────────────

import type { SubscriptionEvent } from "@/hooks/use-billing";
import { Sparkles, Package, RefreshCw, Undo2, Ban as BanIcon, CalendarClock } from "lucide-react";

const EVENT_META: Record<string, { label: string; className: string }> = {
  created: { label: "Dibuat", className: "bg-blue-100 text-blue-800 border-blue-800/40" },
  upgraded: { label: "Upgrade", className: "bg-emerald-100 text-emerald-800 border-emerald-800/40" },
  downgrade_scheduled: { label: "Downgrade dijadwalkan", className: "bg-amber-100 text-amber-800 border-amber-800/40" },
  downgraded: { label: "Downgrade", className: "bg-amber-100 text-amber-800 border-amber-800/40" },
  downgrade_canceled: { label: "Downgrade dibatalkan", className: "bg-muted text-foreground border-foreground/40" },
  renewed: { label: "Diperpanjang", className: "bg-emerald-100 text-emerald-800 border-emerald-800/40" },
  canceled: { label: "Dibatalkan", className: "bg-rose-100 text-rose-800 border-rose-800/40" },
  extended: { label: "Diperpanjang manual", className: "bg-violet-100 text-violet-800 border-violet-800/40" },
  force_downgrade: { label: "Downgrade paksa", className: "bg-rose-100 text-rose-800 border-rose-800/40" },
  expired: { label: "Kedaluwarsa", className: "bg-rose-100 text-rose-800 border-rose-800/40" },
};

function eventIcon(eventType: string) {
  switch (eventType) {
    case "created":
      return Package;
    case "upgraded":
      return Sparkles;
    case "renewed":
      return RefreshCw;
    case "extended":
      return CalendarClock;
    case "downgrade_canceled":
      return Undo2;
    case "canceled":
    case "force_downgrade":
    case "expired":
      return BanIcon;
    default:
      return CalendarClock;
  }
}

function eventSummary(e: SubscriptionEvent): string {
  const meta = EVENT_META[e.event_type];
  if (e.reason) return e.reason;
  return meta?.label ?? e.event_type;
}

export function subscriptionEventColumns({
  page = 1,
  perPage = 20,
}: {
  page?: number;
  perPage?: number;
} = {}): ColumnDef<SubscriptionEvent, unknown>[] {
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
      id: "event_type",
      accessorFn: (e) => e.event_type,
      header: (ctx) => <SortableHeader title="Peristiwa" columnId="event_type" ctx={ctx} />,
      cell: ({ row }) => {
        const meta = EVENT_META[row.original.event_type] ?? {
          label: row.original.event_type,
          className: "bg-muted text-foreground border-foreground/40",
        };
        const Icon = eventIcon(row.original.event_type);
        return (
          <span className={`inline-flex items-center gap-1.5 border-2 rounded-md px-2 py-0.5 text-xs font-black uppercase tracking-wider ${meta.className}`}>
            <Icon className="h-3.5 w-3.5" aria-hidden="true" />
            {meta.label}
          </span>
        );
      },
    },
    {
      id: "summary",
      header: () => <span className="font-black uppercase tracking-wide text-xs">Keterangan</span>,
      enableSorting: false,
      cell: ({ row }) => (
        <span className="block max-w-[360px] truncate text-sm font-bold text-foreground" title={eventSummary(row.original)}>
          {eventSummary(row.original)}
        </span>
      ),
    },
    {
      id: "created_at",
      accessorFn: (e) => e.created_at,
      header: (ctx) => <SortableHeader title="Waktu" columnId="created_at" ctx={ctx} />,
      cell: ({ row }) => {
        const d = new Date(row.original.created_at);
        if (Number.isNaN(d.getTime())) return <span className="text-xs font-bold text-muted-foreground">—</span>;
        return (
          <div className="text-xs font-bold text-foreground">
            <div>{d.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" })}</div>
            <div className="font-mono text-muted-foreground">
              {d.toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" })}
            </div>
          </div>
        );
      },
    },
  ];
}
