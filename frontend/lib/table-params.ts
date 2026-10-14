/**
 * Pure helpers for DataTable URL state.
 *
 * Kept free of React/Next so the parsing/serialisation logic can be unit-tested
 * in isolation. The React hook (`hooks/use-data-table-url-state.ts`) is a thin
 * wrapper around these functions.
 *
 * URL contract (matches the backend `pkg/pagination` parser):
 *   ?page=2&per_page=20&sort=created_at&order=desc&q=text&filter_status=active
 */

export const DEFAULT_PAGE = 1;
export const DEFAULT_PER_PAGE = 20;
export const MIN_PER_PAGE = 10;
export const MAX_PER_PAGE = 100;
export const FILTER_PREFIX = "filter_";

export type SortOrder = "asc" | "desc";

export interface TableState {
  page: number;
  perPage: number;
  sort: string;
  order: SortOrder;
  q: string;
  filters: Record<string, string>;
}

export interface TableDefaults {
  sort?: string;
  order?: SortOrder;
  perPage?: number;
}

/** Clamp a per-page value into the allowed range. */
export function clampPerPage(n: number): number {
  if (!Number.isFinite(n) || n <= 0) return DEFAULT_PER_PAGE;
  if (n < MIN_PER_PAGE) return MIN_PER_PAGE;
  if (n > MAX_PER_PAGE) return MAX_PER_PAGE;
  return Math.floor(n);
}

/** Parse a positive integer or fall back. */
function toPositiveInt(v: string | null, fallback: number): number {
  if (!v) return fallback;
  const n = Number.parseInt(v, 10);
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

/** Normalise order to "asc" | "desc". */
export function normalizeOrder(v: string | null, fallback: SortOrder = "desc"): SortOrder {
  if (!v) return fallback;
  return v.toLowerCase() === "asc" ? "asc" : "desc";
}

/**
 * Parse table state from URL search params, applying safe defaults.
 */
export function parseTableParams(
  params: URLSearchParams | ReadonlyURLSearchParamsLike,
  defaults: TableDefaults = {}
): TableState {
  const get = (k: string): string | null =>
    typeof (params as URLSearchParams).get === "function"
      ? (params as URLSearchParams).get(k)
      : null;

  const filters: Record<string, string> = {};
  params.forEach?.((value, key) => {
    if (key.startsWith(FILTER_PREFIX) && value !== "") {
      filters[key.slice(FILTER_PREFIX.length)] = value;
    }
  });

  return {
    page: toPositiveInt(get("page"), DEFAULT_PAGE),
    perPage: clampPerPage(toPositiveInt(get("per_page"), defaults.perPage ?? DEFAULT_PER_PAGE)),
    sort: get("sort") ?? defaults.sort ?? "",
    order: normalizeOrder(get("order"), defaults.order ?? "desc"),
    q: (get("q") ?? "").trim(),
    filters,
  };
}

/**
 * Serialise table state to URLSearchParams. Default/empty values are omitted to
 * keep URLs clean and shareable.
 */
export function buildTableParams(state: TableState, defaults: TableDefaults = {}): URLSearchParams {
  const p = new URLSearchParams();

  if (state.page !== DEFAULT_PAGE) p.set("page", String(state.page));
  if (state.perPage !== (defaults.perPage ?? DEFAULT_PER_PAGE)) {
    p.set("per_page", String(state.perPage));
  }
  if (state.sort) p.set("sort", state.sort);
  if (state.order && state.order !== (defaults.order ?? "desc")) p.set("order", state.order);
  if (state.q) p.set("q", state.q);

  for (const [k, v] of Object.entries(state.filters)) {
    if (v !== "") p.set(FILTER_PREFIX + k, v);
  }

  return p;
}

/**
 * Apply a partial update to table state. Changing anything other than `page`
 * resets the page to 1 (standard table behaviour).
 */
export function updateTableState(
  state: TableState,
  patch: Partial<TableState>,
  opts: { resetPage?: boolean } = {}
): TableState {
  let nextFilters = state.filters;
  if (patch.filters !== undefined) {
    nextFilters = { ...state.filters, ...patch.filters };
    for (const [k, v] of Object.entries(nextFilters)) {
      if (v === "" || v === undefined) {
        delete nextFilters[k];
      }
    }
  }
  const next: TableState = { ...state, ...patch, filters: nextFilters };
  const resetPage = opts.resetPage ?? Object.keys(patch).some((k) => k !== "page");
  if (resetPage) next.page = 1;
  return next;
}

/** Structural type for Next's ReadonlyURLSearchParams (avoids a hard import). */
interface ReadonlyURLSearchParamsLike {
  get(key: string): string | null;
  forEach?(callback: (value: string, key: string) => void): void;
}
