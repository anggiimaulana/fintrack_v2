"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Trash2, Clock } from "lucide-react";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";

/** A transaction row as returned by GET /transactions. */
export interface TransactionItem {
  id: string;
  type: "income" | "expense" | "savings";
  amount: number;
  category_name?: string;
  expense_category_id?: string | null;
  income_category_id?: string | null;
  savings_category_id?: string | null;
  description?: string | null;
  transaction_date?: string;
  date?: string;
  created_at?: string;
}

function formatAmount(tx: TransactionItem): string {
  const formatted = new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
  }).format(tx.amount);
  const prefix = tx.type === "savings" ? "→" : tx.type === "income" ? "+" : "-";
  return `${prefix}${formatted}`;
}

function amountClass(tx: TransactionItem): string {
  if (tx.type === "savings") return "text-amber-700";
  if (tx.type === "income") return "text-green-700";
  return "text-foreground";
}

export interface TransactionColumnsOptions {
  /** 1-based page number, used for absolute row numbering. */
  page: number;
  perPage: number;
  onEdit: (tx: TransactionItem) => void;
  onDelete: (id: string) => void;
}

/**
 * Column definitions for the transactions table.
 *
 * Sorting is delegated to the server: the accessor keys match the API's sort
 * field names so `sort` in the URL maps straight through.
 */
export function transactionColumns({
  page,
  perPage,
  onEdit,
  onDelete,
}: TransactionColumnsOptions): ColumnDef<TransactionItem, unknown>[] {
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
      id: "description",
      accessorFn: (tx) => tx.description || tx.category_name || "Transaksi",
      // Sort keys match the API's accepted `sortBy` values.
      header: (ctx) => (
        <DataTableColumnHeader
          title="Deskripsi"
          columnId="date"
          sortable
          currentSort={(ctx as { sort?: string }).sort}
          currentOrder={(ctx as { order?: "asc" | "desc" }).order}
          onSort={(ctx as { onSort?: (id: string, order: "asc" | "desc") => void }).onSort}
        />
      ),
      cell: ({ row }) => {
        const tx = row.original;
        const when = new Date((tx.transaction_date || tx.date) as string);
        return (
          <div className="min-w-0">
            <p className="truncate text-sm font-bold text-foreground">
              {tx.description || tx.category_name || "Transaksi"}
            </p>
            <div className="flex flex-wrap items-center gap-1.5 text-xs font-bold text-muted-foreground">
              <span>{tx.type === "savings" ? "Tabungan" : tx.category_name}</span>
              <span aria-hidden="true">&middot;</span>
              <span>{when.toLocaleDateString("id-ID")}</span>
              {tx.created_at && (
                <>
                  <span aria-hidden="true">&middot;</span>
                  <Clock className="h-3 w-3" aria-hidden="true" />
                  <span>
                    {new Date(tx.created_at).toLocaleTimeString("id-ID", {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </span>
                </>
              )}
            </div>
          </div>
        );
      },
    },
    {
      id: "amount",
      accessorFn: (tx) => tx.amount,
      header: (ctx) => (
        <DataTableColumnHeader
          title="Jumlah"
          columnId="amount"
          sortable
          currentSort={(ctx as { sort?: string }).sort}
          currentOrder={(ctx as { order?: "asc" | "desc" }).order}
          onSort={(ctx as { onSort?: (id: string, order: "asc" | "desc") => void }).onSort}
          className="justify-end"
        />
      ),
      cell: ({ row }) => (
        <span className={`font-mono text-sm font-black ${amountClass(row.original)}`}>
          {formatAmount(row.original)}
        </span>
      ),
      meta: { align: "right" },
    },
    {
      id: "actions",
      header: () => <span className="block text-center">Aksi</span>,
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex items-center justify-center gap-1">
          <button
            type="button"
            onClick={() => onEdit(row.original)}
            aria-label="Ubah transaksi"
            className="cursor-pointer border-2 border-transparent p-1.5 text-muted-foreground transition-all hover:border-foreground hover:bg-primary/10 hover:text-primary-strong"
          >
            <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={() => onDelete(row.original.id)}
            aria-label="Hapus transaksi"
            className="cursor-pointer border-2 border-transparent p-1.5 text-muted-foreground transition-all hover:border-foreground hover:bg-red-100 hover:text-red-700"
          >
            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        </div>
      ),
    },
  ];
}
