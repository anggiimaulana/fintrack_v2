"use client";

import { useState, useRef, useEffect } from "react";
import { Camera, X, Loader2, Upload, ImageIcon, Receipt, ShoppingCart, Sparkles, AlertCircle, CheckCircle, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useMutation } from "@tanstack/react-query";
import type { TransactionData } from "@/components/transactions/TransactionRow";
import { useAdminCategories } from "@/hooks/useAdminCategories";
import { useCreateTransaction } from "@/hooks/useTransactions";
import { useInvalidateQueries } from "@/lib/cache-invalidation";
import api from "@/lib/api";

interface ReceiptItem {
  name: string;
  quantity: number;
  unit_price: number;
  total: number;
}

interface ReceiptResult {
  success: boolean;
  message?: string;
  merchant?: string;
  date?: string;
  total?: number;
  category?: string;
  type?: string;
  items?: ReceiptItem[];
  raw_text?: string;
}

interface ReceiptScannerProps {
  open: boolean;
  onClose: () => void;
}

type ScannerStep = "upload" | "preview" | "analyzing" | "saving" | "success" | "error";

export function ReceiptScanner({ open, onClose }: ReceiptScannerProps) {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [result, setResult] = useState<ReceiptResult | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [step, setStep] = useState<ScannerStep>("upload");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const validTypes = ["image/jpeg", "image/png", "image/webp", "image/heic"];
  const maxSize = 2 * 1024 * 1024;

  async function compressImage(file: File): Promise<File> {
    return new Promise((resolve, reject) => {
      const img = new Image();
      img.onload = () => {
        let w = img.width, h = img.height;
        const maxDim = 1920;
        if (w > maxDim || h > maxDim) {
          if (w > h) { h = Math.round(h * maxDim / w); w = maxDim; }
          else { w = Math.round(w * maxDim / h); h = maxDim; }
        }
        const canvas = document.createElement("canvas");
        canvas.width = w; canvas.height = h;
        const ctx = canvas.getContext("2d");
        if (!ctx) { resolve(file); return; }
        ctx.drawImage(img, 0, 0, w, h);
        canvas.toBlob((blob) => {
          if (!blob) { resolve(file); return; }
          resolve(new File([blob], file.name.replace(/\.[^.]+$/, ".webp"), { type: "image/webp" }));
        }, "image/webp", 0.8);
      };
      img.onerror = () => resolve(file);
      const reader = new FileReader();
      reader.onload = (e) => { img.src = e.target?.result as string; };
      reader.onerror = () => resolve(file);
      reader.readAsDataURL(file);
    });
  }

  const { categories: incomeCategories } = useAdminCategories("income");
  const { categories: expenseCategories } = useAdminCategories("expense");
  const { categories: savingsCategories } = useAdminCategories("savings");

  const { invalidateTransactions } = useInvalidateQueries();
  const { mutateAsync: createTransaction } = useCreateTransaction();

  const resetAll = () => {
    setFile(null);
    setPreview(null);
    setResult(null);
    setError(null);
    setStep("upload");
    if (fileInputRef.current) fileInputRef.current.value = "";
  };
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    if (!open) resetAll();
  }, [open]);


  const handleClose = () => {
    if (step === "analyzing" || step === "saving") return;
    resetAll();
    onClose();
  };

  const validateFile = (f: File) => {
    if (!validTypes.includes(f.type)) {
      setError("Format file tidak didukung. Gunakan JPG, PNG, WEBP, HEIC, atau PDF.");
      setStep("error");
      return false;
    }
    if (f.size > maxSize) {
      setError("Ukuran file maksimal 10MB.");
      setStep("error");
      return false;
    }
    return true;
  };

  const handleFileSelect = async (f: File) => {
    if (!validateFile(f)) return;
    setError(null);
    setResult(null);
    const compressed = await compressImage(f);
    setFile(compressed);
    setPreview(URL.createObjectURL(compressed));
    setStep("preview");
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    handleFileSelect(f);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const f = e.dataTransfer.files?.[0];
    if (!f) return;
    handleFileSelect(f);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = () => setDragOver(false);

  const doAutoSave = async (data: ReceiptResult) => {
    const detectedType = (data.type as "income" | "expense" | "savings") || "expense";
    const catList = detectedType === "income" ? (incomeCategories || []) : detectedType === "savings" ? (savingsCategories || []) : (expenseCategories || []);
    // Try exact match, then fallback for "Lainnya" vs "Lain-lain" mismatch
    let matched = catList.find((c: { name: string; id: string }) => c.name?.toLowerCase() === (data.category || "").toLowerCase());
    if (!matched) {
      matched = catList.find((c: { name: string; id: string }) =>
        c.name?.toLowerCase().includes((data.category || "").toLowerCase()) ||
        (data.category || "").toLowerCase().includes(c.name?.toLowerCase())
      );
    }
    // Final fallback: use the generic category for the type
    if (!matched) {
      const fallbackNames = detectedType === "expense" ? ["lain-lain", "lainnya"] : ["lainnya"];
      matched = catList.find((c: { name: string; id: string }) => fallbackNames.some(f => c.name?.toLowerCase() === f));
    }
    const categoryId = matched?.id || "";
    const amount = data.total || 0;

    if (!amount || !categoryId) {
      throw new Error(!amount ? "Jumlah tidak terbaca" : `Kategori "${data.category}" tidak ditemukan`);
    }

    const txDate = data.date
      ? new Date(data.date + "T00:00:00").toISOString()
      : new Date().toISOString();

    const categoryField = detectedType === "income" ? "income_category_id" : detectedType === "savings" ? "savings_category_id" : "expense_category_id";
    await createTransaction({
      type: detectedType as "income" | "expense",
      amount,
      [categoryField]: categoryId,
      description: data.merchant || undefined,
      transaction_date: txDate,
    } as unknown as Partial<TransactionData>);
  };

  const analyzeMutation = useMutation({
    mutationFn: async (f: File) => {
      const formData = new FormData();
      formData.append("image", f);
      const res = await api.post("/ai/receipt", formData, { timeout: 30000 });
      const data = res.data?.data as ReceiptResult;
      if (!data.success) throw new Error(data.message || "Gagal menganalisis struk.");
      return data;
    },
    onSuccess: async (data) => {
      setResult(data);
      setStep("saving");
      try {
        await doAutoSave(data);
        invalidateTransactions();
        setStep("success");
      } catch (saveErr: unknown) {
        setError(saveErr instanceof Error ? saveErr.message : "Gagal menyimpan transaksi.");
        setStep("error");
      }
    },
    onError: (err: unknown) => {
      setResult(null);
      const msg = err instanceof Error ? err.message : "Gagal menganalisis struk.";
      setError(msg);
      setStep("error");
    },
  });

  const handleStartAnalyze = () => {
    if (!file) return;
    setError(null);
    setResult(null);
    setStep("analyzing");
    analyzeMutation.mutate(file);
  };

  const formatRupiah = (num: number) => new Intl.NumberFormat("id-ID").format(num);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={handleClose}>
      <div
        className="border-2 border-foreground bg-card shadow-[6px_6px_0px_0px] shadow-foreground w-full max-w-4xl max-h-[90vh] flex flex-col overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b-2 border-foreground bg-muted shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 border-2 border-foreground bg-primary/10 flex items-center justify-center">
              <Camera className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-sm font-black uppercase tracking-wide text-foreground">
                {step === "success" ? "Transaksi Tersimpan" : "Scan Struk"}
              </h3>
              <p className="text-xs font-bold text-muted-foreground">
                {step === "success"
                  ? "Data struk berhasil dicatat sebagai transaksi"
                  : "Upload foto struk untuk catat transaksi otomatis"}
              </p>
            </div>
          </div>
          <button onClick={handleClose} className="p-1.5 text-muted-foreground hover:text-foreground hover:bg-muted-foreground/10 border-2 border-transparent hover:border-foreground transition-all cursor-pointer" disabled={step === "analyzing" || step === "saving"}>
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto">
          {/* ── STEP: Upload ── */}
          {step === "upload" && (
            <div
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onClick={() => fileInputRef.current?.click()}
              className={`p-12 text-center cursor-pointer transition-all ${
                dragOver ? "bg-primary/5 scale-[1.01]" : "hover:bg-muted/30"
              }`}
            >
              <div className="w-16 h-16 border-2 border-foreground bg-muted flex items-center justify-center mx-auto mb-4">
                <Upload className="w-7 h-7 text-muted-foreground" />
              </div>
              <p className="text-base font-black text-foreground mb-2">Klik atau tarik file ke sini</p>
              <p className="text-sm font-bold text-muted-foreground mb-1">JPG, PNG, WEBP, HEIC - Maks 2MB (kompres otomatis)</p>
              <p className="text-xs text-muted-foreground/60 font-bold">Struk akan otomatis diklasifikasikan sebagai Pemasukan / Pengeluaran / Tabungan</p>
              <input
                type="file"
                accept="image/jpeg,image/png,image/webp,image/heic"
                className="hidden"
                ref={fileInputRef}
                onChange={handleFileChange}
              />
              <Button variant="outline" size="sm" className="mt-5 pointer-events-none">
                <Camera className="w-4 h-4 mr-2" /> Pilih File
              </Button>
            </div>
          )}

          {/* ── STEP: Preview (before analyze) ── */}
          {step === "preview" && file && (
            <div className="flex flex-col lg:flex-row divide-y-2 lg:divide-y-0 lg:divide-x-2 divide-foreground min-h-[300px]">
              <div className="lg:w-2/5 p-5 flex flex-col items-center justify-center">
                <div className="w-full border-2 border-foreground bg-muted overflow-hidden mb-3">
                  {preview ? (
                    <img src={preview} alt="Preview" className="w-full max-h-[260px] object-contain" />
                  ) : null}
                </div>
                <p className="text-xs font-bold text-muted-foreground text-center truncate w-full">{file.name} ({(file.size / 1024 / 1024).toFixed(1)} MB)</p>
                <Button variant="ghost" size="sm" className="mt-2 text-muted-foreground hover:text-foreground" onClick={() => { resetAll(); setTimeout(() => fileInputRef.current?.click(), 50); }}>
                  <Upload className="w-3.5 h-3.5 mr-1.5" /> Ganti File
                </Button>
              </div>
              <div className="lg:w-3/5 p-5 flex flex-col items-center justify-center py-8 text-center">
                <Receipt className="w-12 h-12 text-muted-foreground mb-4" />
                <p className="text-sm font-bold text-foreground mb-1">Siap untuk analisis</p>
                <p className="text-xs font-bold text-muted-foreground mb-6 max-w-xs">
                  Klik tombol di bawah untuk mulai mengekstrak data struk menggunakan AI.
                </p>
                <Button onClick={handleStartAnalyze} size="lg" className="shadow-[3px_3px_0px_0px] shadow-foreground">
                  <Camera className="w-4 h-4 mr-2" /> Analisis Struk
                </Button>
              </div>
            </div>
          )}

          {/* ── STEP: Analyzing ── */}
          {step === "analyzing" && (
            <div className="flex flex-col items-center justify-center py-12">
              <Loader2 className="w-10 h-10 animate-spin text-primary mb-4" />
              <p className="text-sm font-bold text-muted-foreground">Menganalisis struk...</p>
              <p className="text-xs text-muted-foreground/60 font-bold mt-1">AI sedang mengekstrak informasi</p>
            </div>
          )}

          {/* ── STEP: Saving ── */}
          {step === "saving" && (
            <div className="flex flex-col items-center justify-center py-12">
              <Loader2 className="w-10 h-10 animate-spin text-primary mb-4" />
              <p className="text-sm font-bold text-muted-foreground">Menyimpan transaksi...</p>
              <p className="text-xs text-muted-foreground/60 font-bold mt-1">Data struk sedang dicatat</p>
            </div>
          )}

          {/* ── STEP: Error ── */}
          {step === "error" && (
            <div className="p-6">
              <div className="border-2 border-red-500 bg-red-50 p-4 flex items-start gap-3">
                <AlertCircle className="w-5 h-5 text-red-600 shrink-0 mt-0.5" />
                <div>
                  <p className="text-xs font-bold text-red-700">Gagal</p>
                  <p className="text-xs text-red-600 mt-1 whitespace-pre-wrap">{error}</p>
                </div>
              </div>

              {/* Show result data even on save failure */}
              {result && !result.success && result.raw_text && (
                <div className="mt-4 border-2 border-foreground p-3">
                  <p className="text-xs font-black uppercase text-muted-foreground mb-2">Hasil mentah dari AI:</p>
                  <pre className="text-xs font-mono text-muted-foreground whitespace-pre-wrap break-words max-h-40 overflow-y-auto">{result.raw_text}</pre>
                </div>
              )}

              <div className="flex gap-2 mt-4">
                {file && (
                  <Button variant="outline" size="sm" onClick={handleStartAnalyze}>
                    <RotateCcw className="w-3.5 h-3.5 mr-1.5" /> Coba Lagi
                  </Button>
                )}
                <Button variant="outline" size="sm" onClick={() => { resetAll(); setTimeout(() => fileInputRef.current?.click(), 50); }}>
                  <Upload className="w-3.5 h-3.5 mr-1.5" /> Upload Ulang
                </Button>
              </div>
            </div>
          )}

          {/* ── STEP: Success ── */}
          {step === "success" && result && (
            <div className="p-6">
              <div className="flex items-center gap-3 px-4 py-3 border-2 border-green-500 bg-green-50 mb-5">
                <CheckCircle className="w-5 h-5 text-green-600 shrink-0" />
                <div>
                  <p className="text-sm font-bold text-green-700">Transaksi berhasil disimpan!</p>
                  <p className="text-xs text-green-600">Data struk telah otomatis dicatat.</p>
                </div>
              </div>

              {result.type && (
                <div className="flex items-center gap-2 px-3 py-2 border-2 border-primary/30 bg-primary/5 mb-4">
                  <Sparkles className="w-4 h-4 text-primary shrink-0" />
                  <span className="text-xs font-bold text-primary uppercase">
                    Terdeteksi sebagai: {
                      result.type === "income" ? "PEMASUKAN" :
                      result.type === "savings" ? "TABUNGAN" : "PENGELUARAN"
                    }
                  </span>
                </div>
              )}

              <div className="grid grid-cols-2 gap-3 mb-4">
                <div className="border-2 border-foreground p-3">
                  <p className="text-xs font-black uppercase text-muted-foreground mb-1">Merchant</p>
                  <p className="text-sm font-bold">{result.merchant || "-"}</p>
                </div>
                <div className="border-2 border-foreground p-3">
                  <p className="text-xs font-black uppercase text-muted-foreground mb-1">Tanggal</p>
                  <p className="text-sm font-bold">{result.date || "-"}</p>
                </div>
                <div className="border-2 border-foreground p-3">
                  <p className="text-xs font-black uppercase text-muted-foreground mb-1">Kategori</p>
                  <p className="text-sm font-bold">{result.category || "-"}</p>
                </div>
                <div className="border-2 border-foreground p-3">
                  <p className="text-xs font-black uppercase text-muted-foreground mb-1">Total</p>
                  <p className="text-sm font-black text-green-700">Rp {formatRupiah(result.total || 0)}</p>
                </div>
              </div>

              {result.items && result.items.length > 0 && (
                <div className="mb-4">
                  <p className="text-xs font-black uppercase tracking-wide text-muted-foreground mb-2 flex items-center gap-1">
                    <ShoppingCart className="w-3.5 h-3.5" /> Item
                  </p>
                  <div className="border-2 border-foreground divide-y-2 divide-foreground max-h-28 overflow-y-auto">
                    {result.items.map((item, i) => (
                      <div key={i} className="px-3 py-1.5 text-xs font-bold flex items-center justify-between">
                        <span className="truncate flex-1">{item.name}</span>
                        <span className="text-muted-foreground mx-2">{item.quantity}x</span>
                        <span className="text-right w-20">Rp {formatRupiah(item.total)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-5 py-3 border-t-2 border-foreground bg-muted shrink-0">
          <Button variant="outline" size="sm" onClick={handleClose} disabled={step === "analyzing" || step === "saving"}>
            {step === "success" ? "Tutup" : "Batal"}
          </Button>
          {step === "success" && (
            <Button variant="outline" size="sm" onClick={() => { resetAll(); }}>
              <RotateCcw className="w-3.5 h-3.5 mr-1.5" /> Analisis Ulang
            </Button>
          )}
          {step === "preview" && (
            <Button size="sm" onClick={handleStartAnalyze} className="shadow-[2px_2px_0px_0px] shadow-foreground">
              <Camera className="w-4 h-4 mr-2" /> Analisis Struk
            </Button>
          )}
          {step === "error" && file && (
            <Button size="sm" onClick={handleStartAnalyze} className="shadow-[2px_2px_0px_0px] shadow-foreground">
              <RotateCcw className="w-4 h-4 mr-2" /> Coba Lagi
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
