"use client";

import * as React from "react";
import { toast } from "sonner";
import { Banknote, Plus, Pencil, RotateCw, Save, Loader2, LineChart } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ErrorState } from "@/components/shared/ErrorState";
import {
  useAdminCurrencies,
  useCurrencyRates,
  useUpsertCurrency,
  useAddCurrencyRate,
  type AdminCurrency,
  type CurrencyPayload,
  type RateRow,
} from "@/hooks/useSuperAdminCurrency";

function apiErrorMessage(err: unknown): string | null {
  if (typeof err !== "object" || err === null || !("response" in err)) return null;
  const { response } = err as { response?: { data?: { message?: unknown } } };
  const message = response?.data?.message;
  return typeof message === "string" ? message : null;
}

interface CurrencyForm {
  code: string;
  name: string;
  symbol: string;
  locale: string;
  decimal_digits: string;
  sort_order: string;
  is_active: boolean;
}

const EMPTY_FORM: CurrencyForm = {
  code: "",
  name: "",
  symbol: "",
  locale: "id-ID",
  decimal_digits: "0",
  sort_order: "100",
  is_active: false,
};

function toForm(c: AdminCurrency): CurrencyForm {
  return {
    code: c.code,
    name: c.name,
    symbol: c.symbol,
    locale: c.locale,
    decimal_digits: String(c.decimal_digits ?? 0),
    sort_order: String(c.sort_order ?? 0),
    is_active: c.is_active,
  };
}

export default function CurrencyPage() {
  const { data: currencies, isLoading, isError, refetch } = useAdminCurrencies();
  const upsert = useUpsertCurrency();

  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<AdminCurrency | null>(null);
  const [form, setForm] = React.useState<CurrencyForm>(EMPTY_FORM);

  const [rateTarget, setRateTarget] = React.useState<AdminCurrency | null>(null);
  const [rateForm, setRateForm] = React.useState({ rate_to_base: "", effective_at: "", note: "" });

  const openCreate = () => {
    setEditing(null);
    setForm(EMPTY_FORM);
    setDialogOpen(true);
  };

  const openEdit = (c: AdminCurrency) => {
    setEditing(c);
    setForm(toForm(c));
    setDialogOpen(true);
  };

  const openRate = (c: AdminCurrency) => {
    setRateTarget(c);
    setRateForm({ rate_to_base: "", effective_at: new Date().toISOString().slice(0, 10), note: "" });
  };

  const handleSave = async () => {
    if (!/^[A-Za-z]{3}$/.test(form.code.trim())) {
      toast.error("Kode mata uang harus 3 huruf (ISO 4217), mis. JPY.");
      return;
    }
    if (!form.name.trim()) {
      toast.error("Nama mata uang wajib diisi.");
      return;
    }
    const payload: CurrencyPayload = {
      code: form.code.trim().toUpperCase(),
      name: form.name.trim(),
      symbol: form.symbol.trim() || undefined,
      locale: form.locale.trim() || undefined,
      decimal_digits: Number(form.decimal_digits) || 0,
      sort_order: Number(form.sort_order) || 0,
      is_active: form.is_active,
    };
    try {
      await upsert.mutateAsync(payload);
      toast.success("Mata uang disimpan.");
      setDialogOpen(false);
    } catch (err) {
      toast.error(apiErrorMessage(err) || "Gagal menyimpan mata uang.");
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-black uppercase tracking-wide flex items-center gap-2">
            <Banknote className="w-6 h-6 text-primary" /> Mata Uang &amp; Kurs
          </h1>
          <p className="text-sm font-bold text-muted-foreground mt-1">
            Katalog mata uang dinamis dan riwayat kurs. IDR tetap mata uang dasar akuntansi.
            Menambah mata uang tidak memerlukan deploy.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => refetch()} aria-label="Muat ulang">
            <RotateCw className="w-4 h-4" />
          </Button>
          <Button onClick={openCreate}>
            <Plus className="w-4 h-4 mr-2" /> Mata Uang Baru
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2].map((i) => (
            <Skeleton key={i} className="h-24 w-full" />
          ))}
        </div>
      ) : isError ? (
        <ErrorState title="Gagal memuat katalog mata uang" onRetry={() => refetch()} />
      ) : !currencies || currencies.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm font-bold text-muted-foreground">
            Belum ada mata uang.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {currencies.map((c) => (
            <Card key={c.code} className={c.is_base ? "border-primary border-2" : "border-2 border-foreground"}>
              <CardHeader>
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      {c.symbol} {c.name} ({c.code})
                      {c.is_base && <Badge>Base</Badge>}
                      {c.is_active ? (
                        <Badge variant="secondary">Aktif</Badge>
                      ) : (
                        <Badge variant="outline">Nonaktif</Badge>
                      )}
                    </CardTitle>
                    <CardDescription className="mt-1">
                      Lokal {c.locale} · {c.decimal_digits} desimal · urutan {c.sort_order} · sumber kurs {c.rate_source}
                      {c.rate_to_base ? ` · kurs sekarang 1 ${c.code} = ${c.rate_to_base} IDR` : " · belum ada kurs"}
                    </CardDescription>
                  </div>
                  <div className="flex gap-2 shrink-0">
                    <Button size="sm" variant="outline" onClick={() => openRate(c)}>
                      <LineChart className="w-4 h-4 mr-1" /> Kurs
                    </Button>
                    {!c.is_base && (
                      <Button size="sm" variant="outline" onClick={() => openEdit(c)} aria-label="Ubah">
                        <Pencil className="w-4 h-4" />
                      </Button>
                    )}
                  </div>
                </div>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}

      {/* Currency create/edit */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg border-2 rounded-md border-foreground">
          <DialogHeader>
            <DialogTitle>{editing ? "Ubah Mata Uang" : "Mata Uang Baru"}</DialogTitle>
            <DialogDescription>
              Mata uang baru hanya bisa diaktifkan setelah memiliki minimal satu kurs.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="cur-code">Kode (ISO 4217)</Label>
                <Input
                  id="cur-code"
                  value={form.code}
                  maxLength={3}
                  disabled={Boolean(editing)}
                  onChange={(e) => setForm((s) => ({ ...s, code: e.target.value.toUpperCase() }))}
                  placeholder="JPY"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="cur-symbol">Simbol</Label>
                <Input
                  id="cur-symbol"
                  value={form.symbol}
                  onChange={(e) => setForm((s) => ({ ...s, symbol: e.target.value }))}
                  placeholder="¥"
                />
              </div>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="cur-name">Nama</Label>
              <Input
                id="cur-name"
                value={form.name}
                onChange={(e) => setForm((s) => ({ ...s, name: e.target.value }))}
                placeholder="Yen Jepang"
              />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="cur-locale">Locale</Label>
                <Input
                  id="cur-locale"
                  value={form.locale}
                  onChange={(e) => setForm((s) => ({ ...s, locale: e.target.value }))}
                  placeholder="ja-JP"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="cur-digits">Desimal</Label>
                <Input
                  id="cur-digits"
                  type="number"
                  min={0}
                  max={4}
                  value={form.decimal_digits}
                  onChange={(e) => setForm((s) => ({ ...s, decimal_digits: e.target.value }))}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="cur-order">Urutan</Label>
                <Input
                  id="cur-order"
                  type="number"
                  value={form.sort_order}
                  onChange={(e) => setForm((s) => ({ ...s, sort_order: e.target.value }))}
                />
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm font-bold">
              <input
                type="checkbox"
                checked={form.is_active}
                onChange={(e) => setForm((s) => ({ ...s, is_active: e.target.checked }))}
              />
              Aktif (dapat dipilih pengguna)
            </label>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Batal
            </Button>
            <Button onClick={handleSave} disabled={upsert.isPending}>
              {upsert.isPending ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Save className="w-4 h-4 mr-2" />}
              Simpan
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Rates */}
      {rateTarget && (
        <RateDialog
          currency={rateTarget}
          form={rateForm}
          setForm={setRateForm}
          onClose={() => setRateTarget(null)}
        />
      )}
    </div>
  );
}

function RateDialog({
  currency,
  form,
  setForm,
  onClose,
}: {
  currency: AdminCurrency;
  form: { rate_to_base: string; effective_at: string; note: string };
  setForm: React.Dispatch<React.SetStateAction<{ rate_to_base: string; effective_at: string; note: string }>>;
  onClose: () => void;
}) {
  const { data: rates, isLoading } = useCurrencyRates(currency.code);
  const addRate = useAddCurrencyRate();

  const submit = async () => {
    const rate = Number(form.rate_to_base);
    if (!(rate > 0)) {
      toast.error("Kurs harus lebih dari 0.");
      return;
    }
    const payload: RateRow = {
      currency_code: currency.code,
      rate_to_base: rate,
      effective_at: form.effective_at ? new Date(form.effective_at).toISOString() : new Date().toISOString(),
      note: form.note || undefined,
    };
    try {
      await addRate.mutateAsync(payload);
      toast.success("Kurs ditambahkan.");
      setForm((s) => ({ ...s, rate_to_base: "", note: "" }));
    } catch (err) {
      toast.error(apiErrorMessage(err) || "Gagal menambahkan kurs.");
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-lg border-2 rounded-md border-foreground">
        <DialogHeader>
          <DialogTitle>Kurs {currency.code}</DialogTitle>
          <DialogDescription>
            Kurs bersifat append-only: koreksi dilakukan dengan menambah baris baru, bukan mengubah
            yang lama. 1 {currency.code} = … IDR.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="rate-value">Kurs ke IDR</Label>
              <Input
                id="rate-value"
                type="number"
                step="0.0001"
                min={0}
                value={form.rate_to_base}
                onChange={(e) => setForm((s) => ({ ...s, rate_to_base: e.target.value }))}
                placeholder="110.5"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="rate-date">Berlaku sejak</Label>
              <Input
                id="rate-date"
                type="date"
                value={form.effective_at}
                onChange={(e) => setForm((s) => ({ ...s, effective_at: e.target.value }))}
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="rate-note">Catatan (opsional)</Label>
            <Input
              id="rate-note"
              value={form.note}
              onChange={(e) => setForm((s) => ({ ...s, note: e.target.value }))}
              placeholder="Sumber kurs BI, 19 Okt 2026"
            />
          </div>

          <div className="border-2 rounded-md border-foreground/20 p-3">
            <p className="text-xs font-black uppercase text-muted-foreground mb-2">Riwayat Kurs</p>
            {isLoading ? (
              <Skeleton className="h-10 w-full" />
            ) : !rates || rates.length === 0 ? (
              <p className="text-xs font-bold text-muted-foreground">Belum ada kurs.</p>
            ) : (
              <ul className="space-y-1 max-h-48 overflow-y-auto">
                {rates.map((r) => (
                  <li key={r.id ?? `${r.effective_at}-${r.rate_to_base}`} className="flex justify-between text-xs font-bold">
                    <span>{new Date(r.effective_at).toLocaleDateString("id-ID")}</span>
                    <span className="font-mono">{r.rate_to_base} IDR</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Tutup
          </Button>
          <Button onClick={submit} disabled={addRate.isPending}>
            {addRate.isPending ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Plus className="w-4 h-4 mr-2" />}
            Tambah Kurs
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
