"use client";

import { Suspense, useState, useMemo, useCallback } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { Plus, Download, Camera } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { RupiahInput } from "@/components/shared/RupiahInput";
import { VoiceInput } from "@/components/ai/VoiceInput";
import { ReceiptScanner } from "@/components/ai/ReceiptScanner";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { DateRangeFilter, type DateRangeValue, computePresetDates } from "@/components/shared/DateRangeFilter";
import { InlineAiAssistant } from "@/components/ai/InlineAiAssistant";
import { useTransactions, useCreateTransaction, useUpdateTransaction, useDeleteTransaction } from "@/hooks/useTransactions";
import { DataTable } from "@/components/data-table/data-table";
import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { transactionColumns, type TransactionItem } from "@/components/transactions/TransactionColumns";
import { useDataTableUrlState } from "@/hooks/use-data-table-url-state";
import { useAdminCategories } from "@/hooks/useAdminCategories";
import { useEntitlements } from "@/hooks/use-entitlements";
import { notifyEntitlementError } from "@/lib/entitlement-errors";
import api from "@/lib/api";


function TransactionsContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const now = useMemo(() => new Date(), []);

  // List state lives in the URL so it survives refresh and is shareable
  // (TASK-P07-011 AC). The date range keeps its own params (preset/start/end) so
  // the two namespaces do not collide.
  const table = useDataTableUrlState({ perPage: 20 });
  const page = table.page;
  const search = table.q;
  const filterType = table.filters.type || "all";
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingTx, setEditingTx] = useState<TransactionItem | null>(null);
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
    const presetType = initialPreset as import('@/components/shared/DateRangeFilter').PresetKey;
    const d = computePresetDates(presetType, now);
    return { preset: presetType, startDate: d.start, endDate: d.end, compareStart: "", compareEnd: "", enableCompare: false };
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
    type: filterType as "income" | "expense" | "savings" | "all" | undefined,
    page,
    perPage: table.perPage,
    search,
    dateFrom: dateFilter.startDate,
    dateTo: dateFilter.endDate,
    sortBy: (table.sort || undefined) as "date" | "amount" | "category" | undefined,
    sortOrder: table.order,
  });
  const { mutateAsync: createTransaction, isPending: isCreating } = useCreateTransaction();
  const { mutateAsync: updateTransaction, isPending: isUpdating } = useUpdateTransaction();
  const { mutateAsync: deleteTransaction, isPending: isDeleting } = useDeleteTransaction();
  // UX mirror only: the backend gate is authoritative (see lib/entitlement-errors).
  const { hasFeature } = useEntitlements();

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

  const openEditDialog = (tx: TransactionItem) => {
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
      const payload: Record<string, unknown> = {
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
    } catch (err) {
      // A plan-gate refusal (402 limit_exceeded) explains itself and links to the
      // subscription page; only fall back to the generic message otherwise.
      if (!notifyEntitlementError(err)) toast.error("Gagal menyimpan transaksi.");
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
          <Button
            variant="outline"
            className="text-muted-foreground cursor-pointer disabled:cursor-not-allowed"
            disabled={!hasFeature("reports_export")}
            title={hasFeature("reports_export") ? undefined : "Ekspor laporan membutuhkan paket Pro atau lebih tinggi."}
            onClick={() => toast.info("Ekspor laporan belum tersedia pada paket Anda.")}
          >
            <Download className="w-4 h-4 mr-2" />
            Export CSV
          </Button>
          <Button
            variant="outline"
            className="text-amber-600 border-amber-600 hover:bg-amber-50 cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
            disabled={!hasFeature("ocr_receipt")}
            title={hasFeature("ocr_receipt") ? undefined : "Pindai struk membutuhkan paket Pro atau lebih tinggi."}
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
                  <label htmlFor="tx-form-amount" className="text-xs font-bold uppercase tracking-wide text-foreground">Jumlah</label>
                  <RupiahInput id="tx-form-amount" value={formAmount} onChange={(v) => setFormAmount(v || 0)} />
                </div>

                <div className="space-y-2">
                  <label htmlFor="tx-form-category" className="text-xs font-bold uppercase tracking-wide text-foreground">Kategori</label>
                  <select
                    id="tx-form-category"
                    value={formCategoryId}
                    onChange={(e) => setFormCategoryId(e.target.value)}
                    className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
                  >
                    <option value="">Pilih kategori</option>
                    {categories.map((cat: { id: string; name: string }) => (
                      <option key={cat.id} value={cat.id}>{cat.name}</option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <label htmlFor="tx-form-date" className="text-xs font-bold uppercase tracking-wide text-foreground">Tanggal</label>
                  <input
                    id="tx-form-date"
                    type="date"
                    value={formDate}
                    onChange={(e) => setFormDate(e.target.value)}
                    className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
                  />
                </div>

                <div className="space-y-2">
                  <label htmlFor="tx-form-notes" className="text-xs font-bold uppercase tracking-wide text-foreground">Catatan (Opsional)</label>
                  <textarea
                    id="tx-form-notes"
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
                onClick={() => table.setFilter("type", f.key === "all" ? "" : f.key)}
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
            <DataTableToolbar
              search={search}
              onSearch={(q) => table.setSearch(q)}
              placeholder="Cari transaksi..."
            />
        </div>

        <DataTable
          columns={transactionColumns({
            page,
            perPage: table.perPage,
            onEdit: openEditDialog,
            onDelete: handleDeleteClick,
          })}
          data={(transactions || []) as TransactionItem[]}
          isLoading={isLoading}
          emptyMessage="Tidak ada transaksi ditemukan."
          sort={table.sort}
          order={table.order}
          onSort={table.setSort}
        />

        <DataTablePagination
          page={page}
          perPage={table.perPage}
          total={txData?.total ?? 0}
          totalPages={Math.max(1, Math.ceil((txData?.total ?? 0) / table.perPage))}
          onPageChange={table.setPage}
          onPerPageChange={table.setPerPage}
        />
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
