"use client";

import { useCallback, useMemo } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  parseTableParams,
  buildTableParams,
  updateTableState,
  filterPrefix,
  paramName,
  type TableState,
  type TableDefaults,
  type SortOrder,
  type SortSpec,
} from "@/lib/table-params";

export type { TableState, TableDefaults, SortOrder, SortSpec };

export interface UseDataTableUrlState extends TableState {
  /** Replace page. */
  setPage: (page: number) => void;
  /** Replace per-page (resets to page 1). */
  setPerPage: (perPage: number) => void;
  /**
   * Set the sort column. Without an explicit order the direction toggles when
   * re-clicking the active column. When `additive` is true (shift-click) the
   * column is appended/toggled in a multi-column sort instead of replacing it.
   */
  setSort: (sort: string, order?: SortOrder, opts?: { additive?: boolean }) => void;
  /** Replace the entire multi-sort list (resets to page 1). */
  setSorts: (sorts: SortSpec[]) => void;
  /** Set search text (resets to page 1). */
  setSearch: (q: string) => void;
  /** Set a single filter value; empty string removes it (resets to page 1). */
  setFilter: (key: string, value: string) => void;
  /** Set several filter values at once; empty string removes (resets to page 1). */
  setFilters: (filters: Record<string, string>) => void;
  /** Remove all filters and search. */
  reset: () => void;
}

/**
 * Two-way binding between table state and the URL query string.
 *
 * Reads initial state from the current URL and writes changes back via
 * `router.replace` with `{ scroll: false }` so the page does not jump and no
 * full reload happens. State survives refresh and is bookmarkable/shareable.
 *
 * `defaults.prefix` namespaces every param so multiple independent tables can
 * live on one page (e.g. the subscription tabs).
 */
export function useDataTableUrlState(defaults: TableDefaults = {}): UseDataTableUrlState {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const prefix = defaults.prefix;

  const state = useMemo(
    () => parseTableParams(searchParams, defaults),
    // Recompute whenever the query string changes.
    [searchParams, defaults.sort, defaults.order, defaults.perPage, defaults.prefix]
  );

  const commit = useCallback(
    (next: TableState) => {
      const current = new URLSearchParams(searchParams.toString());
      const name = (base: string) => paramName(base, prefix);
      current.delete(name("page"));
      current.delete(name("per_page"));
      current.delete(name("sort"));
      current.delete(name("order"));
      current.delete(name("q"));
      const fPrefix = filterPrefix(prefix);
      Array.from(current.keys()).forEach((k) => {
        if (k.startsWith(fPrefix)) current.delete(k);
      });
      const tableParams = buildTableParams(next, defaults);
      tableParams.forEach((v, k) => current.set(k, v));
      const qs = current.toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    },
    [router, pathname, defaults, searchParams, prefix]
  );

  const setPage = useCallback(
    (page: number) => commit(updateTableState(state, { page }, { resetPage: false })),
    [commit, state]
  );

  const setPerPage = useCallback(
    (perPage: number) => commit(updateTableState(state, { perPage })),
    [commit, state]
  );

  const setSorts = useCallback(
    (sorts: SortSpec[]) => commit(updateTableState(state, { sorts })),
    [commit, state]
  );

  const setSort = useCallback(
    (sort: string, order?: SortOrder, opts?: { additive?: boolean }) => {
      const active = state.sorts ?? [];

      if (opts?.additive) {
        // Shift-click: toggle the column inside the multi-sort list.
        const idx = active.findIndex((s) => s.column === sort);
        let nextSorts: SortSpec[];
        if (idx === -1) {
          nextSorts = [...active, { column: sort, order: order ?? "asc" }];
        } else if (order) {
          nextSorts = active.map((s, i) => (i === idx ? { column: s.column, order } : s));
        } else {
          // Flip the existing column's direction, then move it to the end.
          const flipped: SortSpec = {
            column: sort,
            order: active[idx].order === "asc" ? "desc" : "asc",
          };
          nextSorts = [...active.slice(0, idx), ...active.slice(idx + 1), flipped];
        }
        commit(updateTableState(state, { sorts: nextSorts }));
        return;
      }

      // Plain click: replace the whole sort with this single column, toggling
      // direction when the column is already the active primary sort.
      const nextOrder: SortOrder =
        order ?? (state.sort === sort && state.order === "asc" ? "desc" : "asc");
      commit(updateTableState(state, { sorts: [{ column: sort, order: nextOrder }] }));
    },
    [commit, state]
  );

  const setSearch = useCallback(
    (q: string) => commit(updateTableState(state, { q })),
    [commit, state]
  );

  const setFilter = useCallback(
    (key: string, value: string) => {
      commit(updateTableState(state, { filters: { [key]: value } }));
    },
    [commit, state]
  );

  const setFilters = useCallback(
    (filters: Record<string, string>) => {
      commit(updateTableState(state, { filters }));
    },
    [commit, state]
  );

  const reset = useCallback(() => {
    commit({
      page: 1,
      perPage: defaults.perPage ?? state.perPage,
      sort: defaults.sort ?? "",
      order: defaults.order ?? "desc",
      sorts: defaults.sort
        ? [{ column: defaults.sort, order: defaults.order ?? "desc" }]
        : [],
      q: "",
      filters: {},
    });
  }, [commit, defaults, state.perPage]);

  return {
    ...state,
    setPage,
    setPerPage,
    setSort,
    setSorts,
    setSearch,
    setFilter,
    setFilters,
    reset,
  };
}
