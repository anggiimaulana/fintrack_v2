"use client";

import { Suspense, useState, useMemo, useCallback } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { Plus, Download, Camera, ArrowDownLeft, ArrowUpRight, PiggyBank, Wallet, Pencil, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { RupiahInput } from "@/components/shared/RupiahInput";
import { VoiceInput } from "@/components/ai/VoiceInput";
import { ReceiptScanner } from "@/components/ai/ReceiptScanner";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { DateRangeFilter, type DateRangeValue, computePresetDates } from "@/components/shared/DateRangeFilter";
import { InlineAiAssistant } from "@/components/ai/InlineAiAssistant";
import { useTransactions, useTransactionStats, useCreateTransaction, useUpdateTransaction, useDeleteTransaction } from "@/hooks/useTransactions";
import { formatRupiah } from "@/lib/format";
import { DataTable } from "@/components/data-table/data-table";
import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { transactionColumns, getCategoryName, type TransactionItem } from "@/components/transactions/TransactionColumns";
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
  const [detailTx, setDetailTx] = useState<TransactionItem | null>(null);

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
    const p = new URLSearchParams(typeof window !== "undefined" ? window.location.search : "");
    if (val.preset !== "custom") {
      p.set("preset", val.preset);
      p.delete("start");
      p.delete("end");
    } else {
      p.delete("preset");
      if (val.startDate) p.set("start", val.startDate);
      if (val.endDate) p.set("end", val.endDate);
    }
    const qs = p.toString();
    router.replace(qs ? `/user/transactions?${qs}` : `/user/transactions`, { scroll: false });
  }, [router]);

  const { data: txData, isLoading } = useTransactions({
    type: filterType as "income" | "expense" | "savings" | "all" | undefined,
    page,
    perPage: table.perPage,
    search,
    dateFrom: dateFilter.startDate,
    dateTo: dateFilter.endDate,
    sort: table.sort || undefined,
    order: table.order,
  });
  const { data: statsData, isLoading: isStatsLoading } = useTransactionStats({
    startDate: dateFilter.startDate,
    endDate: dateFilter.endDate,
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

  const categoryMap = useMemo(() => {
    const map = new Map<string, { name: string; color?: string; icon?: string }>();
    incomeCategories?.forEach((c) => map.set(c.id, { name: c.name, color: c.color, icon: c.icon }));
    expenseCategories?.forEach((c) => map.set(c.id, { name: c.name, color: c.color, icon: c.icon }));
    savingsCategories?.forEach((c) => map.set(c.id, { name: c.name, color: c.color, icon: c.icon }));
    return map;
  }, [incomeCategories, expenseCategories, savingsCategories]);

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
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4 flex flex-col justify-between gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Pemasukan</span>
            <div className="w-8 h-8 border-2 border-foreground flex items-center justify-center bg-emerald-100 text-emerald-800">
              <ArrowDownLeft className="w-4 h-4" />
            </div>
          </div>
          <div>
            <div className="text-lg sm:text-xl font-black font-mono text-foreground">
              {isStatsLoading ? "..." : formatRupiah(statsData?.total_income || 0)}
            </div>
            <p className="text-[11px] font-bold text-muted-foreground mt-0.5">
              {isStatsLoading ? "memuat..." : `${statsData?.income_count || 0} transaksi`}
            </p>
          </div>
        </div>

        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4 flex flex-col justify-between gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Pengeluaran</span>
            <div className="w-8 h-8 border-2 border-foreground flex items-center justify-center bg-rose-100 text-rose-800">
              <ArrowUpRight className="w-4 h-4" />
            </div>
          </div>
          <div>
            <div className="text-lg sm:text-xl font-black font-mono text-foreground">
              {isStatsLoading ? "..." : formatRupiah(statsData?.total_expense || 0)}
            </div>
            <p className="text-[11px] font-bold text-muted-foreground mt-0.5">
              {isStatsLoading ? "memuat..." : `${statsData?.expense_count || 0} transaksi`}
            </p>
          </div>
        </div>

        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4 flex flex-col justify-between gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Tabungan</span>
            <div className="w-8 h-8 border-2 border-foreground flex items-center justify-center bg-blue-100 text-blue-800">
              <PiggyBank className="w-4 h-4" />
            </div>
          </div>
          <div>
            <div className="text-lg sm:text-xl font-black font-mono text-foreground">
              {isStatsLoading ? "..." : formatRupiah(statsData?.total_savings || 0)}
            </div>
            <p className="text-[11px] font-bold text-muted-foreground mt-0.5">
              {isStatsLoading ? "memuat..." : `${statsData?.savings_count || 0} transaksi`}
            </p>
          </div>
        </div>

        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4 flex flex-col justify-between gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Sisa Bersih</span>
            <div className="w-8 h-8 border-2 border-foreground flex items-center justify-center bg-amber-100 text-amber-800">
              <Wallet className="w-4 h-4" />
            </div>
          </div>
          <div>
            <div className={`text-lg sm:text-xl font-black font-mono ${(statsData?.net_balance || 0) >= 0 ? "text-foreground" : "text-rose-600"}`}>
              {isStatsLoading ? "..." : formatRupiah(statsData?.net_balance || 0)}
            </div>
            <p className="text-[11px] font-bold text-muted-foreground mt-0.5">
              {isStatsLoading ? "memuat..." : "Pemasukan - Pengeluaran"}
            </p>
          </div>
        </div>
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
            categoryMap,
            onDetail: setDetailTx,
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
      <Dialog open={!!detailTx} onOpenChange={(open) => { if (!open) setDetailTx(null); }}>
        <DialogContent className="sm:max-w-lg border-2 border-foreground shadow-[6px_6px_0px_0px] shadow-foreground">
          <DialogHeader className="pr-8">
            <DialogTitle className="font-black uppercase tracking-wide flex items-center gap-2.5">
              <span>Detail Transaksi</span>
              {detailTx && (
                <span className={`text-[10px] px-2 py-0.5 border-2 border-foreground ${
                  detailTx.type === "income" ? "bg-emerald-100 text-emerald-800" : detailTx.type === "savings" ? "bg-blue-100 text-blue-800" : "bg-rose-100 text-rose-800"
                }`}>
                  {detailTx.type === "income" ? "Pemasukan" : detailTx.type === "savings" ? "Tabungan" : "Pengeluaran"}
                </span>
              )}
            </DialogTitle>
          </DialogHeader>

          {detailTx && (
            <div className="py-2 space-y-4">
              <div className="border-2 border-foreground bg-muted p-4 flex flex-col items-center justify-center gap-1 text-center">
                <span className="text-xs font-bold uppercase tracking-wide text-muted-foreground">Nominal Transaksi</span>
                <span className={`text-2xl sm:text-3xl font-black font-mono ${
                  detailTx.type === "income" ? "text-emerald-700" : detailTx.type === "savings" ? "text-blue-700" : "text-foreground"
                }`}>
                  {detailTx.type === "income" ? "+" : detailTx.type === "savings" ? "→" : "-"}
                  {formatRupiah(detailTx.amount)}
                </span>
              </div>

              <div className="grid grid-cols-2 gap-3 text-xs">
                <div className="border-2 border-foreground p-3 bg-card space-y-1">
                  <p className="font-bold uppercase tracking-wider text-muted-foreground text-[10px]">Kategori</p>
                  <p className="font-black text-sm text-foreground">
                    {getCategoryName(detailTx, categoryMap)}
                  </p>
                </div>

                <div className="border-2 border-foreground p-3 bg-card space-y-1">
                  <p className="font-bold uppercase tracking-wider text-muted-foreground text-[10px]">Tanggal & Waktu</p>
                  <p className="font-black text-sm text-foreground">
                    {new Date(detailTx.transaction_date || detailTx.date || "").toLocaleDateString("id-ID", {
                      day: "numeric",
                      month: "short",
                      year: "numeric",
                    })}
                    {detailTx.created_at && ` · ${new Date(detailTx.created_at).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" })} WIB`}
                  </p>
                </div>

                <div className="col-span-2 border-2 border-foreground p-3 bg-card space-y-1">
                  <p className="font-bold uppercase tracking-wider text-muted-foreground text-[10px]">Deskripsi / Nama</p>
                  <p className="font-bold text-sm text-foreground">
                    {detailTx.description || "-"}
                  </p>
                </div>

                {detailTx.notes && (
                  <div className="col-span-2 border-2 border-foreground p-3 bg-card space-y-1">
                    <p className="font-bold uppercase tracking-wider text-muted-foreground text-[10px]">Catatan</p>
                    <p className="font-bold text-sm text-muted-foreground">
                      {detailTx.notes}
                    </p>
                  </div>
                )}

                <div className="border-2 border-foreground p-3 bg-card space-y-1">
                  <p className="font-bold uppercase tracking-wider text-muted-foreground text-[10px]">Sumber Input</p>
                  <p className="font-black text-xs uppercase text-foreground">
                    {detailTx.source || "manual"}
                  </p>
                </div>

                <div className="border-2 border-foreground p-3 bg-card space-y-1">
                  <p className="font-bold uppercase tracking-wider text-muted-foreground text-[10px]">ID Transaksi</p>
                  <p className="font-mono text-[10px] text-muted-foreground truncate" title={detailTx.id}>
                    {detailTx.id}
                  </p>
                </div>
              </div>

              <div className="flex gap-2 pt-2">
                <Button
                  variant="outline"
                  className="flex-1 cursor-pointer"
                  onClick={() => setDetailTx(null)}
                >
                  Tutup
                </Button>
                <Button
                  className="border-2 border-blue-600 bg-blue-600 hover:bg-blue-700 text-white cursor-pointer"
                  onClick={() => {
                    const tx = detailTx;
                    setDetailTx(null);
                    openEditDialog(tx);
                  }}
                >
                  <Pencil className="w-3.5 h-3.5 mr-1.5" />
                  Ubah
                </Button>
                <Button
                  className="border-2 border-red-600 bg-red-600 hover:bg-red-700 text-white cursor-pointer"
                  onClick={() => {
                    const id = detailTx.id;
                    setDetailTx(null);
                    handleDeleteClick(id);
                  }}
                >
                  <Trash2 className="w-3.5 h-3.5 mr-1.5" />
                  Hapus
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
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
