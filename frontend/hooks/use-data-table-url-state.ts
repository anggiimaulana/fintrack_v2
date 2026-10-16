"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
  /** Immediate optimistic state for instant UI controls. */
  immediate: TableState;
  /** Debounced state (300ms default) for feeding data queries. */
  debounced: TableState;
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
  const debounceMs = defaults.debounceMs ?? 300;

  const searchString = searchParams.toString();
  const parsedState = useMemo(
    () => parseTableParams(searchParams, defaults),
    [searchString, defaults.sort, defaults.order, defaults.perPage, defaults.prefix]
  );

  const [immediateState, setImmediateState] = useState<TableState>(parsedState);
  const [debouncedState, setDebouncedState] = useState<TableState>(parsedState);
  const timerRef = useRef<NodeJS.Timeout | number | null>(null);
  const prevSearchRef = useRef(searchString);

  useEffect(() => {
    if (prevSearchRef.current !== searchString) {
      prevSearchRef.current = searchString;
      setImmediateState(parsedState);
      setDebouncedState(parsedState);
    }
  }, [searchString, parsedState]);
  useEffect(() => {
    return () => clearTimeout(timerRef.current as number);
  }, []);

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
      if (qs !== searchParams.toString()) {
        router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
      }
    },
    [router, pathname, defaults, searchParams, prefix]
  );

  const scheduleCommit = useCallback(
    (next: TableState) => {
      setImmediateState(next);
      clearTimeout(timerRef.current as number);
      timerRef.current = setTimeout(() => {
        setDebouncedState(next);
        commit(next);
      }, debounceMs);
    },
    [commit, debounceMs]
  );

  const immediateCommit = useCallback(
    (next: TableState) => {
      clearTimeout(timerRef.current as number);
      setImmediateState(next);
      setDebouncedState(next);
      commit(next);
    },
    [commit]
  );

  const setPage = useCallback(
    (page: number) => immediateCommit(updateTableState(immediateState, { page }, { resetPage: false })),
    [immediateCommit, immediateState]
  );

  const setPerPage = useCallback(
    (perPage: number) => immediateCommit(updateTableState(immediateState, { perPage })),
    [immediateCommit, immediateState]
  );

  const setSorts = useCallback(
    (sorts: SortSpec[]) => scheduleCommit(updateTableState(immediateState, { sorts })),
    [scheduleCommit, immediateState]
  );

  const setSort = useCallback(
    (sort: string, order?: SortOrder, opts?: { additive?: boolean }) => {
      const active = immediateState.sorts ?? [];

      if (opts?.additive) {
        const idx = active.findIndex((s) => s.column === sort);
        let nextSorts: SortSpec[];
        if (idx === -1) {
          nextSorts = [...active, { column: sort, order: order ?? "asc" }];
        } else if (order) {
          nextSorts = active.map((s, i) => (i === idx ? { column: s.column, order } : s));
        } else {
          const flipped: SortSpec = {
            column: sort,
            order: active[idx].order === "asc" ? "desc" : "asc",
          };
          nextSorts = [...active.slice(0, idx), ...active.slice(idx + 1), flipped];
        }
        scheduleCommit(updateTableState(immediateState, { sorts: nextSorts }));
        return;
      }

      const nextOrder: SortOrder =
        order ?? (immediateState.sort === sort && immediateState.order === "asc" ? "desc" : "asc");
      scheduleCommit(updateTableState(immediateState, { sorts: [{ column: sort, order: nextOrder }] }));
    },
    [scheduleCommit, immediateState]
  );

  const setSearch = useCallback(
    (q: string) => scheduleCommit(updateTableState(immediateState, { q })),
    [scheduleCommit, immediateState]
  );

  const setFilter = useCallback(
    (key: string, value: string) => {
      scheduleCommit(updateTableState(immediateState, { filters: { [key]: value } }));
    },
    [scheduleCommit, immediateState]
  );

  const setFilters = useCallback(
    (filters: Record<string, string>) => {
      scheduleCommit(updateTableState(immediateState, { filters }));
    },
    [scheduleCommit, immediateState]
  );

  const reset = useCallback(() => {
    immediateCommit({
      page: 1,
      perPage: defaults.perPage ?? immediateState.perPage,
      sort: defaults.sort ?? "",
      order: defaults.order ?? "desc",
      sorts: defaults.sort
        ? [{ column: defaults.sort, order: defaults.order ?? "desc" }]
        : [],
      q: "",
      filters: {},
    });
  }, [immediateCommit, defaults, immediateState.perPage]);

  return {
    ...immediateState,
    immediate: immediateState,
    debounced: debouncedState,
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
