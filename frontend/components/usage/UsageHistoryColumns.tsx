"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ArrowLeftRight, Sparkles, ScanLine, Mic, Coins } from "lucide-react";
import { SortableHeader } from "@/components/data-table/data-table-column-header";
import type { UsageEvent } from "@/hooks/use-usage";
import { formatRupiah } from "@/lib/format";

/** A colored icon + label per consumption kind, so a row reads at a glance. */
function kindMeta(kind: string): { icon: React.ElementType; label: string; className: string } {
  switch (kind) {
    case "transaction":
      return { icon: ArrowLeftRight, label: "Transaksi", className: "bg-blue-100 text-blue-800 border-blue-800/40" };
    case "ai_token":
      return { icon: Coins, label: "Token AI", className: "bg-violet-100 text-violet-800 border-violet-800/40" };
    case "ai_request":
      return { icon: Sparkles, label: "Permintaan AI", className: "bg-amber-100 text-amber-800 border-amber-800/40" };
    case "ocr":
      return { icon: ScanLine, label: "Pindai Struk", className: "bg-emerald-100 text-emerald-800 border-emerald-800/40" };
    case "voice":
      return { icon: Mic, label: "Input Suara", className: "bg-rose-100 text-rose-800 border-rose-800/40" };
    default:
      return { icon: Sparkles, label: kind, className: "bg-muted text-foreground border-foreground/40" };
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

function formatAmount(amount: number, unit: string): string {
  const n = new Intl.NumberFormat("id-ID").format(amount);
  return unit ? `${n} ${unit}` : n;
}

export interface UsageHistoryColumnsOptions {
  /** 1-based page number, used for absolute row numbering. */
  page?: number;
  perPage?: number;
}

/**
 * Column definitions for the usage-history ledger. Sorting is server-side: the
 * column ids match the API's accepted `sort` values (created_at | kind | amount
 * | cost_idr).
 */
export function usageHistoryColumns({
  page = 1,
  perPage = 20,
}: UsageHistoryColumnsOptions = {}): ColumnDef<UsageEvent, unknown>[] {
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
      id: "kind",
      accessorFn: (e) => e.kind,
      header: (ctx) => <SortableHeader title="Jenis" columnId="kind" ctx={ctx} />,
      cell: ({ row }) => {
        const meta = kindMeta(row.original.kind);
        const Icon = meta.icon;
        return (
          <span className={`inline-flex items-center gap-1.5 border-2 rounded-md px-2 py-0.5 text-xs font-black uppercase tracking-wider ${meta.className}`}>
            <Icon className="h-3.5 w-3.5" aria-hidden="true" />
            {row.original.kind_label || meta.label}
          </span>
        );
      },
    },
    {
      id: "description",
      header: () => <span className="font-black uppercase tracking-wide text-xs">Keterangan</span>,
      enableSorting: false,
      cell: ({ row }) => (
        <span className="block max-w-[280px] truncate text-sm font-bold text-foreground" title={row.original.description}>
          {row.original.description || "—"}
        </span>
      ),
    },
    {
      id: "amount",
      accessorFn: (e) => e.amount,
      header: (ctx) => <SortableHeader title="Jumlah" columnId="amount" ctx={ctx} className="justify-end" />,
      cell: ({ row }) => (
        <span className="font-mono text-sm font-black text-foreground">
          {formatAmount(row.original.amount, row.original.unit)}
        </span>
      ),
      meta: { align: "right" },
    },
    {
      id: "cost_idr",
      accessorFn: (e) => e.cost_idr,
      header: (ctx) => <SortableHeader title="Biaya" columnId="cost_idr" ctx={ctx} className="justify-end" />,
      cell: ({ row }) =>
        row.original.cost_idr > 0 ? (
          <span className="font-mono text-sm font-bold text-foreground">{formatRupiah(row.original.cost_idr)}</span>
        ) : (
          <span className="text-xs font-bold text-muted-foreground">—</span>
        ),
      meta: { align: "right" },
    },
    {
      id: "created_at",
      accessorFn: (e) => e.created_at,
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
  ];
}
