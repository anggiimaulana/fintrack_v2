"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { TransactionRow, TransactionData } from "./TransactionRow";
import Link from "next/link";
import { ArrowRight } from "lucide-react";

interface TransactionTableProps {
  transactions: TransactionData[];
  isLoading?: boolean;
}

export function TransactionTable({ transactions, isLoading }: TransactionTableProps) {
  return (
    <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground flex flex-col overflow-hidden">
      <div className="px-5 py-4 border-b-2 border-foreground flex justify-between items-center bg-card">
        <h3 className="text-xs font-black uppercase tracking-wide text-foreground">Transaksi Terbaru</h3>
        <Link href="/user/transactions" className="text-xs font-black text-primary hover:text-primary/80 flex items-center gap-1 cursor-pointer uppercase tracking-wide">
          Lihat Semua <ArrowRight className="w-3 h-3" />
        </Link>
      </div>
      
      <div className="divide-y-2 divide-foreground">
        {isLoading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 p-4">
              <Skeleton className="h-10 w-10 border-2 border-foreground" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-4 w-48 border-2 border-foreground" />
                <Skeleton className="h-3 w-24 border-2 border-foreground" />
              </div>
              <Skeleton className="h-5 w-24 border-2 border-foreground" />
            </div>
          ))
        ) : transactions.length > 0 ? (
          transactions.map((trx) => (
            <TransactionRow key={trx.id} transaction={trx} />
          ))
        ) : (
          <div className="p-8 text-center text-sm font-bold text-muted-foreground">
            Belum ada transaksi bulan ini.
          </div>
        )}
      </div>
    </div>
  );
}
