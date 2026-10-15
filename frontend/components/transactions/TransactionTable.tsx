"use client";

import { useMemo } from "react";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { DataTable } from "@/components/data-table/data-table";
import { transactionColumns, type TransactionItem } from "./TransactionColumns";

export interface TransactionTableProps {
  transactions: TransactionItem[];
  isLoading?: boolean;
  categoryMap?: Map<string, { name: string; color?: string; icon?: string }>;
}

export function TransactionTable({ transactions, isLoading, categoryMap }: TransactionTableProps) {
  const columns = useMemo(
    () =>
      transactionColumns({
        page: 1,
        perPage: 10,
        categoryMap,
        hideActions: true,
      }),
    [categoryMap]
  );

  return (
    <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground flex flex-col overflow-hidden">
      <div className="px-5 py-4 border-b-2 border-foreground flex justify-between items-center bg-card">
        <h3 className="text-xs font-black uppercase tracking-wide text-foreground">Transaksi Terbaru</h3>
        <Link href="/user/transactions" className="text-xs font-black text-primary hover:text-primary/80 flex items-center gap-1 cursor-pointer uppercase tracking-wide">
          Lihat Semua <ArrowRight className="w-3 h-3" />
        </Link>
      </div>

      <DataTable
        columns={columns}
        data={transactions.slice(0, 10)}
        isLoading={isLoading}
        emptyMessage="Belum ada transaksi bulan ini."
        skeletonColumns={5}
      />
    </div>
  );
}
