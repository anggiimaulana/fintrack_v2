"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { useTheme } from "@/providers/theme-provider";
import { useCurrency } from "@/providers/currency-provider";
import { useSetDisplayCurrency } from "@/hooks/useCurrencies";
import { isPriced } from "@/lib/currency";
import { cn } from "@/lib/utils";

const THEME_OPTIONS: { value: "light" | "dark" | "system"; label: string }[] = [
  { value: "light", label: "Terang" },
  { value: "dark", label: "Gelap" },
  { value: "system", label: "Sistem" },
];

/** Prefer the backend's own message over axios's generic one. */
function apiErrorMessage(err: unknown, fallback: string): string {
  const e = err as { response?: { data?: { message?: unknown } } } | null;
  const message = e?.response?.data?.message;
  return typeof message === "string" && message ? message : fallback;
}

/**
 * Appearance (theme) and language. Theme drives the theme provider directly so
 * the choice takes effect immediately and the active option is visibly
 * highlighted. Language is fixed to Indonesian until an i18n layer exists.
 *
 * Currency is the user's recording/display currency: it is stored server-side
 * (users.display_currency) and drives every financial view. The list comes from
 * the Super Admin catalog, so adding a currency needs no deploy.
 */
export function AppearanceSettings() {
  const { theme, setTheme } = useTheme();
  const { currencies, displayCurrency, isLoading: currencyLoading } = useCurrency();
  const { mutate: setCurrency, isPending: savingCurrency } = useSetDisplayCurrency();
  const [mounted, setMounted] = useState(false);

  // Only currencies with a usable rate are selectable; an unpriced option would
  // relabel IDR amounts without converting them.
  const selectableCurrencies = currencies.filter((c) => c.is_base || isPriced(c));

  // The stored theme is only known after hydration; render the buttons as
  // inactive until then to avoid a server/client mismatch.
  useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);

  const onCurrencyChange = (code: string) => {
    if (code === displayCurrency.code) return;
    setCurrency(code, {
      onSuccess: () => toast.success("Mata uang berhasil diperbarui"),
      onError: (err) => toast.error(apiErrorMessage(err, "Gagal memperbarui mata uang")),
    });
  };

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-3">
          <p className="text-xs font-bold uppercase tracking-wide text-foreground">Tema</p>
          <div className="flex gap-3">
            {THEME_OPTIONS.map((option) => {
              const isActive = mounted && theme === option.value;
              return (
                <button
                  key={option.value}
                  type="button"
                  aria-pressed={isActive}
                  onClick={() => setTheme(option.value)}
                  className={cn(
                    "flex-1 py-2.5 text-sm font-bold border-2 rounded-md border-foreground transition-all cursor-pointer",
                    isActive
                      ? "bg-primary text-primary-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                      : "bg-muted text-muted-foreground hover:text-foreground"
                  )}
                >
                  {option.label}
                </button>
              );
            })}
          </div>
        </div>

        <div className="space-y-3">
          <label htmlFor="settings-language" className="text-xs font-bold uppercase tracking-wide text-foreground">
            Bahasa &amp; Regional
          </label>
          {/* Only Indonesian ships today, so the field is a fixed, disabled value
              rather than a picker that would pretend English works. */}
          <select
            id="settings-language"
            value="id"
            disabled
            className="w-full border-2 rounded-md border-foreground bg-muted px-4 py-2.5 text-sm font-bold text-muted-foreground cursor-not-allowed focus:outline-none"
          >
            <option value="id">Indonesia (ID)</option>
          </select>
          <p className="text-[11px] font-bold text-muted-foreground">
            Saat ini baru tersedia Bahasa Indonesia. Dukungan bahasa lain segera hadir.
          </p>
        </div>
      </div>

      <div className="space-y-3">
        <label htmlFor="settings-currency" className="text-xs font-bold uppercase tracking-wide text-foreground">
          Mata Uang Pencatatan
        </label>
        <select
          id="settings-currency"
          value={displayCurrency.code}
          disabled={currencyLoading || savingCurrency}
          onChange={(e) => onCurrencyChange(e.target.value)}
          className="w-full md:w-1/2 border-2 rounded-md border-foreground bg-background px-4 py-2.5 text-sm font-bold text-foreground focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
        >
          {selectableCurrencies.map((c) => (
            <option key={c.code} value={c.code}>
              {c.name} ({c.code}) — {c.symbol}
            </option>
          ))}
        </select>
        <p className="text-[11px] font-bold text-muted-foreground">
          Pilihan ini berlaku untuk dashboard, transaksi, laporan, dan anggaran. Tagihan, kuota AI,
          dan lisensi tetap dalam Rupiah (IDR).
        </p>
      </div>
    </div>
  );
}
