import { format } from "date-fns";
import { id as idLocale } from "date-fns/locale";

const rupiahFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

/** Format a number as Indonesian Rupiah, e.g. 1000 -> "Rp1.000". */
export function formatRupiah(value: number | string | null | undefined): string {
  const n = typeof value === "string" ? Number(value) : value;
  if (n == null || Number.isNaN(n)) return "Rp0";
  return rupiahFormatter.format(n);
}

/** Format a date as "d MMM yyyy" in Indonesian, e.g. "15 Apr 2026". */
export function formatTanggal(value: string | Date | null | undefined): string {
  if (!value) return "-";
  const d = typeof value === "string" ? new Date(value) : value;
  if (Number.isNaN(d.getTime())) return "-";
  return format(d, "d MMM yyyy", { locale: idLocale });
}
