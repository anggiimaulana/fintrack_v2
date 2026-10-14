"use client";

import * as React from "react";
import { toast } from "sonner";
import { Plus, Trash2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useBudgets, useDeleteBudget, useUpsertBudget, type BudgetStatus } from "@/hooks/useBudgets";
import { useAdminCategories } from "@/hooks/useAdminCategories";

function formatIDR(value: number): string {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value);
}

/** Colour band for a budget's usage. */
function barClass(over: boolean, usedPct: number): string {
  if (over) return "bg-red-600";
  if (usedPct >= 80) return "bg-amber-500";
  return "bg-green-600";
}

interface BudgetPanelProps {
  period: { startDate: string; endDate: string } | undefined;
  /** Whether the plan grants `advanced_budgeting`. */
  enabled: boolean;
}

/**
 * Budget vs actual: set a monthly ceiling per expense category and see how much
 * of it is used in the selected period.
 */
export function BudgetPanel({ period, enabled }: BudgetPanelProps) {
  const { data: budgets = [], isLoading } = useBudgets(period, { enabled });
  const { categories: expenseCategories = [] } = useAdminCategories("expense");
  const upsert = useUpsertBudget();
  const remove = useDeleteBudget();

  const [categoryId, setCategoryId] = React.useState("");
  const [amount, setAmount] = React.useState("");

  if (!enabled) {
    return (
      <Card className="border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader>
          <CardTitle className="text-xs font-black uppercase tracking-wide">Anggaran vs Aktual</CardTitle>
          <CardDescription>Fitur anggaran tersedia di paket Pro ke atas.</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  const handleAdd = async () => {
    const idr = Number(amount.replace(/[^\d]/g, ""));
    if (!categoryId || !idr) {
      toast.error("Pilih kategori dan isi nominal anggaran");
      return;
    }
    try {
      await upsert.mutateAsync({ category_id: categoryId, amount_idr: idr });
      toast.success("Anggaran disimpan");
      setAmount("");
    } catch {
      toast.error("Gagal menyimpan anggaran");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await remove.mutateAsync(id);
      toast.success("Anggaran dihapus");
    } catch {
      toast.error("Gagal menghapus anggaran");
    }
  };

  const totalBudget = budgets.reduce((s, b) => s + b.amount_idr, 0);
  const totalActual = budgets.reduce((s, b) => s + b.actual, 0);

  return (
    <div className="space-y-6">
      {/* Add form */}
      <Card className="border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader>
          <CardTitle className="text-xs font-black uppercase tracking-wide">Atur Anggaran Kategori</CardTitle>
          <CardDescription>Tetapkan batas bulanan per kategori pengeluaran.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex-1">
            <label htmlFor="budget-category" className="text-[10px] font-black uppercase text-muted-foreground">
              Kategori
            </label>
            <select
              id="budget-category"
              value={categoryId}
              onChange={(e) => setCategoryId(e.target.value)}
              className="h-9 w-full border-2 border-foreground bg-background px-2 text-sm font-bold"
            >
              <option value="">Pilih kategori…</option>
              {expenseCategories.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>
          <div className="flex-1">
            <label htmlFor="budget-amount" className="text-[10px] font-black uppercase text-muted-foreground">
              Nominal (Rp)
            </label>
            <Input
              id="budget-amount"
              inputMode="numeric"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="500000"
            />
          </div>
          <Button type="button" className="cursor-pointer" onClick={handleAdd} disabled={upsert.isPending}>
            <Plus className="mr-1.5 h-4 w-4" /> Simpan
          </Button>
        </CardContent>
      </Card>

      {/* Overview */}
      {budgets.length > 0 && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="border-2 border-foreground bg-card p-4 shadow-[4px_4px_0px_0px] shadow-foreground">
            <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Total Anggaran</p>
            <p className="font-mono text-lg font-black">{formatIDR(totalBudget)}</p>
          </div>
          <div className="border-2 border-foreground bg-card p-4 shadow-[4px_4px_0px_0px] shadow-foreground">
            <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Terpakai</p>
            <p className="font-mono text-lg font-black text-red-600">{formatIDR(totalActual)}</p>
          </div>
          <div className="border-2 border-foreground bg-card p-4 shadow-[4px_4px_0px_0px] shadow-foreground">
            <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Sisa</p>
            <p className={`font-mono text-lg font-black ${totalBudget - totalActual < 0 ? "text-red-600" : "text-green-700"}`}>
              {formatIDR(totalBudget - totalActual)}
            </p>
          </div>
        </div>
      )}

      {/* Per-category bars */}
      {isLoading ? (
        <div className="h-40 animate-pulse border-2 border-foreground bg-muted" />
      ) : budgets.length === 0 ? (
        <Card className="border-2 border-foreground">
          <CardContent className="py-10 text-center text-sm font-bold text-muted-foreground">
            Belum ada anggaran. Tambahkan satu di atas untuk mulai memantau.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {budgets.map((b: BudgetStatus) => (
            <div key={b.id} className="border-2 border-foreground bg-card p-4 shadow-[4px_4px_0px_0px] shadow-foreground">
              <div className="mb-2 flex items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-black">{b.category_name}</span>
                  {b.over && (
                    <span className="border-2 border-red-600 px-1.5 py-0.5 text-[10px] font-black uppercase text-red-600">
                      Lewat {formatIDR(Math.abs(b.remaining))}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-3">
                  <span className="font-mono text-xs font-bold">
                    {formatIDR(b.actual)} / {formatIDR(b.amount_idr)}
                  </span>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="h-7 w-7 cursor-pointer text-red-600"
                    onClick={() => handleDelete(b.id)}
                    aria-label={`Hapus anggaran ${b.category_name}`}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-3 flex-1 border-2 border-foreground bg-muted">
                  <div
                    className={`h-full ${barClass(b.over, b.used_pct)}`}
                    style={{ width: `${Math.min(100, b.used_pct)}%` }}
                  />
                </div>
                <span className="w-14 text-right font-mono text-xs font-black">{b.used_pct.toFixed(0)}%</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
