"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Trash2, Clock, Eye } from "lucide-react";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";

/** A transaction row as returned by GET /transactions. */
export interface TransactionItem {
  id: string;
  type: "income" | "expense" | "savings";
  amount: number;
  category_name?: string;
  category?: string;
  subcategory_name?: string;
  subcategory?: string;
  expense_category_id?: string | null;
  income_category_id?: string | null;
  savings_category_id?: string | null;
  description?: string | null;
  notes?: string | null;
  transaction_date?: string;
  date?: string;
  created_at?: string;
  source?: "manual" | "ai" | "voice" | "receipt";
}

export function getCategoryName(
  tx: TransactionItem,
  categoryMap?: Map<string, { name: string; color?: string; icon?: string }>
): string {
  const catId = tx.expense_category_id || tx.income_category_id || tx.savings_category_id;
  if (catId && categoryMap?.has(catId)) {
    return categoryMap.get(catId)!.name;
  }
  if (tx.category_name) return tx.category_name;
  if (tx.category) return tx.category;
  if (tx.type === "savings") return "Tabungan";
  if (tx.type === "income") return "Pemasukan";
  return "Pengeluaran";
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
  page?: number;
  perPage?: number;
  categoryMap?: Map<string, { name: string; color?: string; icon?: string }>;
  onDetail?: (tx: TransactionItem) => void;
  onEdit?: (tx: TransactionItem) => void;
  onDelete?: (id: string) => void;
  hideActions?: boolean;
}
/**
 * Column definitions for the transactions table.
 *
 * Sorting is delegated to the server: the accessor keys match the API's sort
 * field names so `sort` in the URL maps straight through.
 */
export function transactionColumns({
  page = 1,
  perPage = 10,
  categoryMap,
  onDetail,
  onEdit,
  onDelete,
  hideActions = false,
}: TransactionColumnsOptions): ColumnDef<TransactionItem, unknown>[] {
  const cols: ColumnDef<TransactionItem, unknown>[] = [
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
          columnId="description"
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
              {tx.description || getCategoryName(tx, categoryMap) || "Transaksi"}
            </p>
            <div className="flex flex-wrap items-center gap-1.5 text-xs font-bold text-muted-foreground">
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
      id: "type",
      header: (ctx) => (
        <DataTableColumnHeader
          title="Tipe"
          columnId="type"
          sortable
          currentSort={(ctx as { sort?: string }).sort}
          currentOrder={(ctx as { order?: "asc" | "desc" }).order}
          onSort={(ctx as { onSort?: (id: string, order: "asc" | "desc") => void }).onSort}
        />
      ),
      cell: ({ row }) => {
        const txType = row.original.type;
        const typeConfig =
          txType === "savings"
            ? { label: "Tabungan", style: "bg-blue-100 text-blue-900 border-blue-900/40" }
            : txType === "income"
            ? { label: "Pemasukan", style: "bg-emerald-100 text-emerald-900 border-emerald-900/40" }
            : { label: "Pengeluaran", style: "bg-rose-100 text-rose-900 border-rose-900/40" };
        return (
          <span className={`inline-block border-2 rounded-md px-2 py-0.5 text-xs font-black uppercase tracking-wider ${typeConfig.style}`}>
            {typeConfig.label}
          </span>
        );
      },
    },
    {
      id: "category",
      header: (ctx) => (
        <DataTableColumnHeader
          title="Kategori"
          columnId="category"
          sortable
          currentSort={(ctx as { sort?: string }).sort}
          currentOrder={(ctx as { order?: "asc" | "desc" }).order}
          onSort={(ctx as { onSort?: (id: string, order: "asc" | "desc") => void }).onSort}
        />
      ),
      cell: ({ row }) => {
        const catName = getCategoryName(row.original, categoryMap);
        return (
          <span className="inline-block border-2 rounded-md border-foreground bg-muted px-2.5 py-0.5 text-xs font-bold text-foreground">
            {catName}
          </span>
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
  ];

  if (!hideActions && (onEdit || onDelete || onDetail)) {
    cols.push({
      id: "actions",
      header: () => <span className="block text-center">Aksi</span>,
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex items-center justify-center gap-1.5">
          {onDetail && (
            <button
              type="button"
              onClick={() => onDetail(row.original)}
              aria-label="Detail transaksi"
              title="Detail transaksi"
              className="cursor-pointer border-2 rounded-md border-foreground bg-card p-1.5 text-foreground transition-all hover:bg-muted shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[1px] active:translate-y-[1px]"
            >
              <Eye className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
          )}
          {onEdit && (
            <button
              type="button"
              onClick={() => onEdit(row.original)}
              aria-label="Ubah transaksi"
              title="Ubah transaksi"
              className="cursor-pointer border-2 rounded-md border-blue-600 bg-blue-50 p-1.5 text-blue-600 transition-all hover:bg-blue-600 hover:text-white shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[1px] active:translate-y-[1px]"
            >
              <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
          )}
          {onDelete && (
            <button
              type="button"
              onClick={() => onDelete(row.original.id)}
              aria-label="Hapus transaksi"
              title="Hapus transaksi"
              className="cursor-pointer border-2 rounded-md border-red-600 bg-red-50 p-1.5 text-red-600 transition-all hover:bg-red-600 hover:text-white shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[1px] active:translate-y-[1px]"
            >
              <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
          )}
        </div>
      ),
    });
  }

  return cols;
}
