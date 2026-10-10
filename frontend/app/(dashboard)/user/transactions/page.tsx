"use client";

import { Suspense, useState, useMemo, useCallback } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { Plus, Download, Search, ChevronLeft, ChevronRight, Mic, Camera, Sparkles, Pencil, Trash2, X, Clock, Send, ArrowUp } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { RupiahInput } from "@/components/ui/rupiah-input";
import { VoiceInput } from "@/components/ai/VoiceInput";
import { ReceiptScanner } from "@/components/ai/ReceiptScanner";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { DateRangeFilter, type DateRangeValue, computePresetDates } from "@/components/shared/DateRangeFilter";
import { InlineAiAssistant } from "@/components/ai/InlineAiAssistant";
import { useTransactions, useCreateTransaction, useUpdateTransaction, useDeleteTransaction } from "@/hooks/useTransactions";
import { useAdminCategories } from "@/hooks/useAdminCategories";
import api from "@/lib/api";

interface TransactionRowProps {
  tx: any;
  index: number;
  page: number;
  perPage: number;
  onEdit: (tx: any) => void;
  onDelete: (id: string) => void;
}

function TransactionRowData({ tx, index, page, perPage, onEdit, onDelete }: TransactionRowProps) {
  const isIncome = tx.type === "income";
  const isSavings = tx.type === "savings";
  const formattedAmount = new Intl.NumberFormat("id-ID", {
    style: "currency", currency: "IDR", minimumFractionDigits: 0,
  }).format(tx.amount);

  const rowColor = isSavings ? "text-amber-600" : isIncome ? "text-green-700" : "text-foreground";
  const prefix = isSavings ? "→" : isIncome ? "+" : "-";

  return (
    <div className="flex items-center gap-3 px-5 py-3 border-b-2 border-foreground last:border-b-0 hover:bg-muted/50 transition-colors group">
      <span className="text-xs font-mono font-bold text-muted-foreground w-8 shrink-0">{(page - 1) * perPage + index + 1}</span>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-bold text-foreground truncate">{tx.description || tx.category_name || "Transaksi"}</p>
        <div className="flex items-center gap-1.5 text-xs font-bold text-muted-foreground flex-wrap">
          <span>{isSavings ? "Tabungan" : tx.category_name}</span>
          <span>&middot;</span>
          <span>{new Date(tx.transaction_date || tx.date).toLocaleDateString("id-ID")}</span>
          {tx.created_at && (
            <>
              <span>&middot;</span>
              <Clock className="w-3 h-3" />
              <span>{new Date(tx.created_at).toLocaleDateString("id-ID", { hour: "2-digit", minute: "2-digit" })}</span>
            </>
          )}
        </div>
      </div>
      <span className={`text-sm font-black font-mono shrink-0 min-w-[100px] text-right ${rowColor}`}>
        {prefix}{formattedAmount}
      </span>
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        <button onClick={() => onEdit(tx)} className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 border-2 border-transparent hover:border-foreground cursor-pointer transition-all" title="Edit">
          <Pencil className="w-3.5 h-3.5" />
        </button>
        <button onClick={() => onDelete(tx.id)} className="p-1.5 text-muted-foreground hover:text-red-600 hover:bg-red-100 border-2 border-transparent hover:border-foreground cursor-pointer transition-all" title="Hapus">
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}

function TransactionsContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const now = useMemo(() => new Date(), []);
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;

  const [page, setPage] = useState(1);
  const [filterType, setFilterType] = useState<string>("all");
  const [search, setSearch] = useState("");
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingTx, setEditingTx] = useState<any | null>(null);
  const [smartInputText, setSmartInputText] = useState("");
  const [scannerOpen, setScannerOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  // Date filter from URL
  const initialPreset = searchParams.get("preset") || "this_month";
  const initialStart = searchParams.get("start") || "";
  const initialEnd = searchParams.get("end") || "";
  const [dateFilter, setDateFilterState] = useState<DateRangeValue>(() => {
    if (initialStart && initialEnd) {
      return { preset: "custom", startDate: initialStart, endDate: initialEnd, compareStart: "", compareEnd: "", enableCompare: false };
    }
    const d = computePresetDates(initialPreset as any, now);
    return { preset: initialPreset as any, startDate: d.start, endDate: d.end, compareStart: "", compareEnd: "", enableCompare: false };
  });

  const setDateFilter = useCallback((val: DateRangeValue) => {
    setDateFilterState(val);
    const p: Record<string, string> = {};
    if (val.preset !== "custom") p.preset = val.preset;
    if (val.preset === "custom" || val.startDate) { p.start = val.startDate; p.end = val.endDate; }
    const qs = new URLSearchParams(p).toString();
    router.replace(`/user/transactions?${qs}`, { scroll: false });
  }, [router]);

  const { data: txData, isLoading } = useTransactions({
    type: filterType as any,
    page,
    perPage: 20,
    search,
    dateFrom: dateFilter.startDate,
    dateTo: dateFilter.endDate,
  });
  const { mutateAsync: createTransaction, isPending: isCreating } = useCreateTransaction();
  const { mutateAsync: updateTransaction, isPending: isUpdating } = useUpdateTransaction();
  const { mutateAsync: deleteTransaction, isPending: isDeleting } = useDeleteTransaction();

  const [formType, setFormType] = useState<"expense" | "income" | "savings">("expense");
  const [formAmount, setFormAmount] = useState(0);
  const [formCategoryId, setFormCategoryId] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formDate, setFormDate] = useState(new Date().toISOString().split("T")[0]);

  const { categories: incomeCategories } = useAdminCategories("income");
  const { categories: expenseCategories } = useAdminCategories("expense");
  const { categories: savingsCategories } = useAdminCategories("savings");
  const categories = formType === "income" ? (incomeCategories || []) : formType === "expense" ? (expenseCategories || []) : (savingsCategories || []);

  const transactions = txData?.data || [];
  const totalPages = Math.ceil((txData?.total || 0) / 20);

  const openCreateDialog = () => {
    setEditingTx(null);
    setFormType("expense");
    setFormAmount(0);
    setFormCategoryId("");
    setFormDescription("");
    setFormDate(new Date().toISOString().split("T")[0]);
    setIsDialogOpen(true);
  };

  const openEditDialog = (tx: any) => {
    setEditingTx(tx);
    setFormType(tx.type);
    setFormAmount(tx.amount);
    setFormCategoryId(tx.expense_category_id || tx.income_category_id || tx.savings_category_id || "");
    setFormDescription(tx.description || "");
    setFormDate(tx.transaction_date?.split("T")[0] || new Date().toISOString().split("T")[0]);
    setIsDialogOpen(true);
  };

  const handleSubmit = async () => {
    if (!formAmount || !formCategoryId) return;
    try {
      const categoryField = formType === "income" ? "income_category_id" : formType === "expense" ? "expense_category_id" : "savings_category_id";
      const payload: any = {
        type: formType,
        amount: formAmount,
        [categoryField]: formCategoryId,
        description: formDescription || undefined,
        transaction_date: formDate + "T00:00:00Z",
      };
      if (editingTx) {
        await updateTransaction({ id: editingTx.id, ...payload });
      } else {
        await createTransaction(payload);
      }
      setFormAmount(0);
      setFormCategoryId("");
      setFormDescription("");
      setFormDate(new Date().toISOString().split("T")[0]);
      setIsDialogOpen(false);
      setEditingTx(null);
      toast.success(editingTx ? "Transaksi berhasil diperbarui" : "Transaksi berhasil disimpan");
    } catch {
      toast.error("Gagal menyimpan transaksi.");
    }
  };

  const handleDeleteClick = (id: string) => {
    setConfirmDeleteId(id);
    setConfirmOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!confirmDeleteId) return;
    try {
      await deleteTransaction(confirmDeleteId);
      toast.success("Transaksi berhasil dihapus");
      setConfirmOpen(false);
      setConfirmDeleteId(null);
    } catch {
      toast.error("Gagal menghapus transaksi.");
    }
  };

  const resetForm = () => {
    setEditingTx(null);
    setFormType("expense");
    setFormAmount(0);
    setFormCategoryId("");
    setFormDescription("");
    setFormDate(new Date().toISOString().split("T")[0]);
  };

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Daftar Transaksi</h2>
          <p className="text-xs font-bold text-muted-foreground mt-1">Kelola dan pantau semua riwayat transaksi Anda.</p>
        </div>
          <div className="flex items-center gap-3">
          <Button variant="outline" className="text-muted-foreground cursor-pointer">
            <Download className="w-4 h-4 mr-2" />
            Export CSV
          </Button>
          <Button
            variant="outline"
            className="text-amber-600 border-amber-600 hover:bg-amber-50 cursor-pointer"
            onClick={() => setScannerOpen(true)}
          >
            <Camera className="w-4 h-4 mr-2" />
            Analisis Struk
          </Button>
          <Button className="bg-primary text-primary-foreground cursor-pointer" onClick={openCreateDialog}>
            <Plus className="w-4 h-4 mr-2" />
            Tambah Transaksi
          </Button>
          <Dialog open={isDialogOpen} onOpenChange={(open) => { if (!open) resetForm(); setIsDialogOpen(open); }}>
            <DialogContent className="sm:max-w-md border-2 border-foreground">
              <DialogHeader>
                <DialogTitle className="font-black uppercase tracking-wide">{editingTx ? "Edit Transaksi" : "Tambah Transaksi Baru"}</DialogTitle>
              </DialogHeader>
              <div className="py-4 space-y-5">
                <div className="flex border-2 border-foreground p-1 bg-muted">
                  <button
                    onClick={() => { setFormType("expense"); setFormCategoryId(""); }}
                    className={`flex-1 py-2 text-sm font-bold cursor-pointer ${formType === "expense" ? "border-2 border-foreground bg-card shadow-[2px_2px_0px_0px] shadow-foreground text-foreground" : "text-muted-foreground hover:text-foreground border-0"}`}
                  >
                    Pengeluaran
                  </button>
                  <button
                    onClick={() => { setFormType("income"); setFormCategoryId(""); }}
                    className={`flex-1 py-2 text-sm font-bold cursor-pointer ${formType === "income" ? "border-2 border-foreground bg-card shadow-[2px_2px_0px_0px] shadow-foreground text-foreground" : "text-muted-foreground hover:text-foreground border-0"}`}
                  >
                    Pemasukan
                  </button>
                  <button
                    onClick={() => { setFormType("savings"); setFormCategoryId(""); }}
                    className={`flex-1 py-2 text-sm font-bold cursor-pointer ${formType === "savings" ? "border-2 border-foreground bg-card shadow-[2px_2px_0px_0px] shadow-foreground text-foreground" : "text-muted-foreground hover:text-foreground border-0"}`}
                  >
                    Tabungan
                  </button>
                </div>

                <div className="space-y-2">
                  <label className="text-xs font-bold uppercase tracking-wide text-foreground">Jumlah</label>
                  <RupiahInput value={formAmount} onChange={setFormAmount} />
                </div>

                <div className="space-y-2">
                  <label className="text-xs font-bold uppercase tracking-wide text-foreground">Kategori</label>
                  <select
                    value={formCategoryId}
                    onChange={(e) => setFormCategoryId(e.target.value)}
                    className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
                  >
                    <option value="">Pilih kategori</option>
                    {categories.map((cat: any) => (
                      <option key={cat.id} value={cat.id}>{cat.name}</option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <label className="text-xs font-bold uppercase tracking-wide text-foreground">Tanggal</label>
                  <input
                    type="date"
                    value={formDate}
                    onChange={(e) => setFormDate(e.target.value)}
                    className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-xs font-bold uppercase tracking-wide text-foreground">Catatan (Opsional)</label>
                  <textarea
                    rows={2}
                    placeholder="Makan siang bersama teman..."
                    value={formDescription}
                    onChange={(e) => setFormDescription(e.target.value)}
                    className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70 resize-none"
                  />
                </div>

                <div className="flex gap-3 pt-2">
                  <Button variant="outline" className="flex-1" onClick={() => { resetForm(); setIsDialogOpen(false); }}>Batal</Button>
                  <Button className="flex-1" onClick={handleSubmit} disabled={isCreating || isUpdating || !formAmount || !formCategoryId}>
                    {isCreating || isUpdating ? "Menyimpan..." : editingTx ? "Perbarui" : "Simpan"}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <InlineAiAssistant />

      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-3">
        <DateRangeFilter value={dateFilter} onChange={setDateFilter} showCompare={false} />
      </div>

      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground flex flex-col overflow-hidden">
        <div className="p-4 border-b-2 border-foreground flex flex-col md:flex-row gap-4 justify-between items-center bg-muted">
          <div className="flex border-2 border-foreground p-1 bg-background w-full md:w-auto">
            {[{ key: "all", label: "Semua" }, { key: "income", label: "Pemasukan" }, { key: "expense", label: "Pengeluaran" }, { key: "savings", label: "Tabungan" }].map((f) => (
              <button
                key={f.key}
                onClick={() => { setFilterType(f.key); setPage(1); }}
                className={`flex-1 md:flex-none px-4 py-1.5 text-xs font-bold cursor-pointer transition-colors ${
                  filterType === f.key
                    ? "bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground border-x-2 border-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>
          <div className="relative w-full md:w-64">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              placeholder="Cari transaksi..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
              className="w-full pl-9 pr-4 py-2 text-sm border-2 border-foreground bg-background focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
            />
          </div>
        </div>

        <div className="divide-y-0">
          <div className="grid grid-cols-[40px_1fr_140px_80px] gap-3 px-5 py-2.5 bg-muted border-b-2 border-foreground text-xs font-black uppercase tracking-wide text-muted-foreground">
            <span>#</span>
            <span>Deskripsi</span>
            <span className="text-right">Jumlah</span>
            <span className="text-center">Aksi</span>
          </div>
          {isLoading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 px-5 py-3 border-b-2 border-foreground last:border-b-0">
                <div className="w-8 h-4 bg-muted border-2 border-foreground animate-pulse" />
                <div className="flex-1 space-y-1.5">
                  <div className="h-4 w-48 bg-muted border-2 border-foreground animate-pulse" />
                  <div className="h-3 w-32 bg-muted border-2 border-foreground animate-pulse" />
                </div>
                <div className="h-5 w-28 bg-muted border-2 border-foreground animate-pulse" />
                <div className="w-16 h-5 bg-muted border-2 border-foreground animate-pulse" />
              </div>
            ))
          ) : transactions.length > 0 ? (
            transactions.map((tx: any, i: number) => (
              <TransactionRowData key={tx.id} tx={tx} index={i} page={page} perPage={12} onEdit={openEditDialog} onDelete={handleDeleteClick} />
            ))
          ) : (
            <div className="p-8 text-center text-sm font-bold text-muted-foreground">
              Tidak ada transaksi ditemukan.
            </div>
          )}
        </div>

        {totalPages > 1 && (
          <div className="p-4 border-t-2 border-foreground flex justify-between items-center bg-card">
            <span className="text-xs font-bold text-muted-foreground">
              Halaman {page} dari {totalPages} ({txData?.total} transaksi)
            </span>
            <div className="flex gap-1">
              <Button variant="outline" size="sm" onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1}>
                <ChevronLeft className="w-4 h-4" />
              </Button>
              {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
                const start = Math.max(1, Math.min(page - 2, totalPages - 4));
                const p = start + i;
                if (p > totalPages) return null;
                return (
                  <Button
                    key={p}
                    variant={p === page ? "default" : "outline"}
                    size="sm"
                    onClick={() => setPage(p)}
                    className={p === page ? "bg-primary text-primary-foreground shadow-[2px_2px_0px_0px] shadow-foreground" : ""}
                  >
                    {p}
                  </Button>
                );
              })}
              <Button variant="outline" size="sm" onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
                <ChevronRight className="w-4 h-4" />
              </Button>
            </div>
          </div>
        )}
      </div>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        onConfirm={handleDeleteConfirm}
        title="Hapus Transaksi"
        message="Yakin ingin menghapus transaksi ini? Tindakan ini tidak dapat dibatalkan."
        confirmText="Ya, Hapus"
        loading={isDeleting}
      />

      <ReceiptScanner
        open={scannerOpen}
        onClose={() => setScannerOpen(false)}
      />
    </div>
  );
}

export default function TransactionsPage() {
  return (
    <Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 border-foreground" />}>
      <TransactionsContent />
    </Suspense>
  );
}
