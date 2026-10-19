/**
 * Multi-currency helpers.
 *
 * The backend keeps IDR as the base accounting currency: every aggregate
 * (dashboard stats, reports, budgets) is expressed in IDR base units. The user's
 * chosen display currency is a presentation concern, so these helpers convert
 * base → display at format time using the catalog rate.
 *
 * The active currency is held in module state (same pattern as
 * `aiRequestTimeout` in lib/api.ts) so existing call sites can follow the user's
 * choice without threading a hook through every component. The provider sets it
 * synchronously before children render and re-renders the subtree on change.
 *
 * A small catalog registry mirrors the provider's currency list so pure helpers
 * (`formatOriginal`, `formatDisplay`) can resolve a symbol/locale by code even
 * outside React (e.g. a legacy transaction whose currency is not the active one).
 */

export type CurrencyMeta = {
  code: string;
  name: string;
  symbol: string;
  locale: string;
  decimal_digits: number;
  is_active: boolean;
  is_base: boolean;
  sort_order: number;
  rate_source: string;
  /** 1 unit of this currency = rate_to_base IDR (present for selectable currencies). */
  rate_to_base?: number | null;
  rate_at?: string | null;
};

/** IDR is the base currency and the fallback when nothing is configured. */
export const BASE_CURRENCY: CurrencyMeta = {
  code: "IDR",
  name: "Rupiah Indonesia",
  symbol: "Rp",
  locale: "id-ID",
  decimal_digits: 0,
  is_active: true,
  is_base: true,
  sort_order: 10,
  rate_source: "manual",
  rate_to_base: 1,
};

let active: CurrencyMeta = BASE_CURRENCY;
let catalog: Record<string, CurrencyMeta> = { [BASE_CURRENCY.code]: BASE_CURRENCY };

/** True when the currency reports a usable base rate. */
export function isPriced(meta: CurrencyMeta | null | undefined): boolean {
  return Boolean(meta && meta.rate_to_base && meta.rate_to_base > 0);
}

/** Publish the selectable catalog so pure helpers can resolve any code. */
export function setCurrencyCatalog(list?: CurrencyMeta[] | null): void {
  const next: Record<string, CurrencyMeta> = { [BASE_CURRENCY.code]: BASE_CURRENCY };
  for (const c of list ?? []) {
    if (c && c.code) next[c.code] = c;
  }
  catalog = next;
}

/** Resolve the metadata for a code (falls back to the active one when omitted). */
export function currencyMeta(code?: string | null): CurrencyMeta {
  if (!code) return active;
  const found = catalog[code];
  if (found) return found;
  if (code === BASE_CURRENCY.code) return BASE_CURRENCY;
  // Unknown code: never inherit the base currency's rate (that would let callers
  // treat it as priced and convert without a real quote).
  return { ...BASE_CURRENCY, code, symbol: symbolFor(code), is_base: false, is_active: false, rate_to_base: null };
}

/** Set the currency used by {@link formatMoney}/{@link toBase}. A currency with
 *  no usable rate is never made active — otherwise base amounts would be
 *  relabelled without conversion. */
export function setActiveCurrency(meta?: CurrencyMeta | null): void {
  if (meta && meta.code && (meta.code === BASE_CURRENCY.code || isPriced(meta))) {
    active = meta;
  } else {
    active = BASE_CURRENCY;
  }
}

export function getActiveCurrency(): CurrencyMeta {
  return active;
}

/** Format an amount already expressed in `meta`'s currency. */
export function formatInCurrency(value: number | string | null | undefined, meta: CurrencyMeta): string {
  const n = typeof value === "string" ? Number(value) : value;
  const digits = clampDigits(meta.decimal_digits);
  if (n == null || Number.isNaN(n)) {
    return `${meta.symbol}${(0).toLocaleString(safeLocale(meta.locale), {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    })}`;
  }
  const abs = Math.abs(n).toLocaleString(safeLocale(meta.locale), {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
  return `${n < 0 ? "-" : ""}${meta.symbol}${abs}`;
}

/** Format a value expressed in the recorded currency (no conversion). */
export function formatOriginal(value: number | string | null | undefined, code?: string | null): string {
  return formatInCurrency(value, currencyMeta(code));
}

/** Format an amount the server already converted to `code` (report endpoints). */
export function formatDisplay(value: number | string | null | undefined, code?: string | null): string {
  return formatInCurrency(value, currencyMeta(code));
}

/** Format an IDR base value in the active display currency. */
export function formatMoney(value: number | string | null | undefined): string {
  const cur = active;
  const n = typeof value === "string" ? Number(value) : value;
  if (n == null || Number.isNaN(n)) return formatInCurrency(0, cur);
  const rate = isPriced(cur) ? cur.rate_to_base! : 1;
  return formatInCurrency(n / rate, cur);
}

/** Convert an IDR base value into the active display currency (number). */
export function fromBase(base: number | null | undefined): number {
  const n = base ?? 0;
  const rate = isPriced(active) ? active.rate_to_base! : 1;
  return n / rate;
}

/** Convert a value expressed in the active display currency into IDR base. */
export function toBase(value: number | string | null | undefined): number {
  const n = typeof value === "string" ? Number(value) : value;
  if (n == null || Number.isNaN(n)) return 0;
  const rate = isPriced(active) ? active.rate_to_base! : 1;
  return n * rate;
}

function clampDigits(digits: number | undefined | null): number {
  if (digits == null || Number.isNaN(digits)) return 0;
  return Math.min(4, Math.max(0, Math.trunc(digits)));
}

function safeLocale(locale: string | undefined | null): string {
  if (!locale) return "id-ID";
  try {
    // An invalid locale throws a RangeError; fall back to the app default so a
    // bad catalog value can never crash a money render.
    new Intl.NumberFormat(locale);
    return locale;
  } catch {
    return "id-ID";
  }
}

/** Best-effort symbol for a code the catalog has not (yet) described. */
function symbolFor(code: string): string {
  switch (code) {
    case "IDR":
      return "Rp";
    case "JPY":
      return "¥";
    case "USD":
      return "$";
    case "MYR":
      return "RM";
    default:
      return `${code} `;
  }
}
