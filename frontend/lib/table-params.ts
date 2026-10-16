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

/** One column + direction inside a (possibly multi-column) sort. */
export interface SortSpec {
  column: string;
  order: SortOrder;
}

export interface TableState {
  page: number;
  perPage: number;
  /** Primary sort column (mirror of `sorts[0]`), kept for single-sort callers. */
  sort: string;
  /** Primary sort direction (mirror of `sorts[0]`). */
  order: SortOrder;
  /** Full ordered sort list. Empty when no sort is active. */
  sorts: SortSpec[];
  q: string;
  filters: Record<string, string>;
}

export interface TableDefaults {
  sort?: string;
  order?: SortOrder;
  perPage?: number;
  /**
   * Namespace for this table's URL params, so several independent tables can
   * coexist on one page (e.g. the subscription tabs). When set, params become
   * `<prefix>page`, `<prefix>sort`, `<prefix>filter_status`, … Backend requests
   * are unaffected — the page reads this state and maps it to API params.
   */
  prefix?: string;
  /** Debounce delay in ms for filter, sort, and search commits (default 300). */
  debounceMs?: number;
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

/** Effective filter prefix for a table (namespaced when a prefix is set). */
export function filterPrefix(prefix?: string): string {
  return prefix ? `${prefix}_${FILTER_PREFIX}` : FILTER_PREFIX;
}

/** Effective URL param name for a base key (namespaced when a prefix is set). */
export function paramName(base: string, prefix?: string): string {
  return prefix ? `${prefix}_${base}` : base;
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

  const fPrefix = filterPrefix(defaults.prefix);
  const filters: Record<string, string> = {};
  params.forEach?.((value, key) => {
    if (key.startsWith(fPrefix) && value !== "") {
      filters[key.slice(fPrefix.length)] = value;
    }
  });

  const rawSort = get(paramName("sort", defaults.prefix)) ?? "";
  const rawOrder = get(paramName("order", defaults.prefix)) ?? "";
  const sorts = parseSortList(rawSort, rawOrder, defaults);

  return {
    page: toPositiveInt(get(paramName("page", defaults.prefix)), DEFAULT_PAGE),
    perPage: clampPerPage(
      toPositiveInt(
        get(paramName("per_page", defaults.prefix)),
        defaults.perPage ?? DEFAULT_PER_PAGE
      )
    ),
    sort: sorts[0]?.column ?? "",
    order: sorts[0]?.order ?? normalizeOrder(null, defaults.order ?? "desc"),
    sorts,
    q: (get(paramName("q", defaults.prefix)) ?? "").trim(),
    filters,
  };
}

/**
 * Parse a comma-separated `sort=a,b` / `order=desc,asc` pair into a sort list.
 * When the URL carries no sort, the endpoint default (if any) is used.
 */
export function parseSortList(
  rawSort: string,
  rawOrder: string,
  defaults: TableDefaults = {}
): SortSpec[] {
  const cols = rawSort
    .split(",")
    .map((c) => c.trim())
    .filter(Boolean);
  if (cols.length === 0) {
    if (defaults.sort) {
      return [{ column: defaults.sort, order: defaults.order ?? "desc" }];
    }
    return [];
  }
  const orders = rawOrder.split(",").map((o) => o.trim());
  const fallback = defaults.order ?? "desc";
  return cols.map((column, i) => {
    const raw = orders[i] ?? orders[orders.length - 1] ?? "";
    return { column, order: normalizeOrder(raw || null, fallback) };
  });
}

/**
 * Serialise table state to URLSearchParams. Default/empty values are omitted to
 * keep URLs clean and shareable.
 */
export function buildTableParams(state: TableState, defaults: TableDefaults = {}): URLSearchParams {
  const p = new URLSearchParams();
  const name = (base: string) => paramName(base, defaults.prefix);

  if (state.page !== DEFAULT_PAGE) p.set(name("page"), String(state.page));
  if (state.perPage !== (defaults.perPage ?? DEFAULT_PER_PAGE)) {
    p.set(name("per_page"), String(state.perPage));
  }

  // Multi-sort: only emit `sort`/`order` when the list differs from defaults.
  const sorts = state.sorts ?? [];
  if (sorts.length > 0) {
    const isDefault =
      sorts.length === 1 &&
      sorts[0].column === (defaults.sort ?? "") &&
      sorts[0].order === (defaults.order ?? "desc");
    if (!isDefault) {
      p.set(name("sort"), sorts.map((s) => s.column).join(","));
      p.set(name("order"), sorts.map((s) => s.order).join(","));
    }
  }

  if (state.q) p.set(name("q"), state.q);

  const fPrefix = filterPrefix(defaults.prefix);
  for (const [k, v] of Object.entries(state.filters)) {
    if (v !== "") p.set(fPrefix + k, v);
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

  // Keep `sorts` and the `sort`/`order` mirrors consistent: an explicit `sorts`
  // patch wins; otherwise a `sort`/`order` patch collapses to a single column.
  let nextSorts = patch.sorts ?? state.sorts ?? [];
  if (patch.sorts === undefined && (patch.sort !== undefined || patch.order !== undefined)) {
    const column = patch.sort ?? state.sort;
    const order = patch.order ?? state.order;
    nextSorts = column ? [{ column, order }] : [];
  }

  const next: TableState = {
    ...state,
    ...patch,
    filters: nextFilters,
    sorts: nextSorts,
    sort: nextSorts[0]?.column ?? "",
    order: nextSorts[0]?.order ?? "desc",
  };
  const resetPage = opts.resetPage ?? Object.keys(patch).some((k) => k !== "page");
  if (resetPage) next.page = 1;
  return next;
}

/** Structural type for Next's ReadonlyURLSearchParams (avoids a hard import). */
interface ReadonlyURLSearchParamsLike {
  get(key: string): string | null;
  forEach?(callback: (value: string, key: string) => void): void;
}
