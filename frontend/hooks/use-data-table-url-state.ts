"use client";

import { useCallback, useMemo } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  parseTableParams,
  buildTableParams,
  updateTableState,
  FILTER_PREFIX,
  type TableState,
  type TableDefaults,
  type SortOrder,
} from "@/lib/table-params";

export type { TableState, TableDefaults, SortOrder };

export interface UseDataTableUrlState extends TableState {
  /** Replace page. */
  setPage: (page: number) => void;
  /** Replace per-page (resets to page 1). */
  setPerPage: (perPage: number) => void;
  /** Toggle/set sort column (resets to page 1). */
  setSort: (sort: string, order?: SortOrder) => void;
  /** Set search text (resets to page 1). */
  setSearch: (q: string) => void;
  /** Set a single filter value; empty string removes it (resets to page 1). */
  setFilter: (key: string, value: string) => void;
  /** Remove all filters and search. */
  reset: () => void;
}

/**
 * Two-way binding between table state and the URL query string.
 *
 * Reads initial state from the current URL and writes changes back via
 * `router.replace` with `{ scroll: false }` so the page does not jump and no
 * full reload happens. State survives refresh and is bookmarkable/shareable.
 */
export function useDataTableUrlState(defaults: TableDefaults = {}): UseDataTableUrlState {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const state = useMemo(
    () => parseTableParams(searchParams, defaults),
    // Recompute whenever the query string changes.
    [searchParams, defaults.sort, defaults.order, defaults.perPage]
  );

  const commit = useCallback(
    (next: TableState) => {
      const current = new URLSearchParams(searchParams.toString());
      current.delete("page");
      current.delete("per_page");
      current.delete("sort");
      current.delete("order");
      current.delete("q");
      Array.from(current.keys()).forEach((k) => {
        if (k.startsWith(FILTER_PREFIX)) current.delete(k);
      });
      const tableParams = buildTableParams(next, defaults);
      tableParams.forEach((v, k) => current.set(k, v));
      const qs = current.toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    },
    [router, pathname, defaults, searchParams]
  );

  const setPage = useCallback(
    (page: number) => commit(updateTableState(state, { page }, { resetPage: false })),
    [commit, state]
  );

  const setPerPage = useCallback(
    (perPage: number) => commit(updateTableState(state, { perPage })),
    [commit, state]
  );

  const setSort = useCallback(
    (sort: string, order?: SortOrder) => {
      // Toggle direction when clicking the same column without an explicit order.
      const nextOrder: SortOrder =
        order ?? (state.sort === sort && state.order === "asc" ? "desc" : "asc");
      commit(updateTableState(state, { sort, order: nextOrder }));
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

  const reset = useCallback(() => {
    commit({
      page: 1,
      perPage: defaults.perPage ?? state.perPage,
      sort: defaults.sort ?? "",
      order: defaults.order ?? "desc",
      q: "",
      filters: {},
    });
  }, [commit, defaults, state.perPage]);

  return { ...state, setPage, setPerPage, setSort, setSearch, setFilter, reset };
}
