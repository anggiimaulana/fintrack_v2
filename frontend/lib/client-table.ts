/**
 * Pure client-side table operations for datasets that arrive in one payload
 * (e.g. the report's category list). The transactions page delegates this to
 * the server because its dataset is large; a report has tens of rows, so doing
 * it in the browser avoids a round trip per keystroke.
 *
 * All functions are pure so the table's behaviour is unit-testable without
 * rendering, and so the URL remains the single source of truth for the state.
 */

export type SortOrder = "asc" | "desc";

export interface TableQuery {
  page: number;
  perPage: number;
  sort: string;
  order: SortOrder;
  q: string;
  filters: Record<string, string>;
}

/** The minimal row shape the table operations need. */
export type Row = Record<string, unknown>;

/** Compares two cell values for sorting; numbers before strings, nulls last. */
function compareValues(a: unknown, b: unknown): number {
  if (a === b) return 0;
  if (a === null || a === undefined) return 1;
  if (b === null || b === undefined) return -1;
  if (typeof a === "number" && typeof b === "number") return a - b;
  return String(a).localeCompare(String(b), "id");
}

/**
 * sortRows returns a new array sorted by `sort` in `order`. An empty `sort`
 * leaves the order untouched (the source order is meaningful).
 */
export function sortRows<T extends Row>(rows: T[], sort: string, order: SortOrder): T[] {
  if (!sort) return [...rows];
  const dir = order === "asc" ? 1 : -1;
  return [...rows].sort((a, b) => compareValues(a[sort], b[sort]) * dir);
}

/**
 * searchRows keeps rows where any of `keys` contains the query
 * (case-insensitive, accent-insensitive).
 */
export function searchRows<T extends Row>(rows: T[], keys: string[], q: string): T[] {
  const needle = normalize(q);
  if (!needle) return rows;
  return rows.filter((row) =>
    keys.some((k) => normalize(String(row[k] ?? "")).includes(needle))
  );
}

/**
 * filterRows applies exact-match filters (empty values are ignored, so removing
 * a filter is just clearing its value).
 */
export function filterRows<T extends Row>(rows: T[], filters: Record<string, string>): T[] {
  const active = Object.entries(filters).filter(([, v]) => v !== "" && v !== undefined);
  if (active.length === 0) return rows;
  return rows.filter((row) => active.every(([k, v]) => String(row[k] ?? "") === v));
}

/** paginate slices a page out of the rows, clamped to valid bounds. */
export function paginate<T>(rows: T[], page: number, perPage: number): T[] {
  const size = perPage > 0 ? perPage : 20;
  const start = Math.max(0, (Math.max(1, page) - 1) * size);
  return rows.slice(start, start + size);
}

/** totalPages returns the page count (at least 1). */
export function totalPages(total: number, perPage: number): number {
  const size = perPage > 0 ? perPage : 20;
  return Math.max(1, Math.ceil(total / size));
}

/**
 * applyTableQuery runs search → filter → sort → paginate in that order and
 * returns both the page slice and the pre-pagination count (for the footer).
 * Returning the count avoids the caller recomputing the pipeline.
 */
export function applyTableQuery<T extends Row>(
  rows: T[],
  query: TableQuery,
  searchKeys: string[]
): { page: T[]; total: number } {
  let out = searchRows(rows, searchKeys, query.q);
  out = filterRows(out, query.filters);
  out = sortRows(out, query.sort, query.order);
  const total = out.length;
  return { page: paginate(out, query.page, query.perPage), total };
}

/** normalize lowercases and strips diacritics for tolerant matching. */
function normalize(s: string): string {
  return s.normalize("NFD").replace(/\p{Diacritic}/gu, "").toLowerCase().trim();
}
