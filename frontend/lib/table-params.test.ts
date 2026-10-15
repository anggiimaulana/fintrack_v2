import { describe, it, expect } from "vitest";
import {
  parseTableParams,
  buildTableParams,
  updateTableState,
  parseSortList,
  clampPerPage,
  normalizeOrder,
  paramName,
  filterPrefix,
  DEFAULT_PAGE,
  DEFAULT_PER_PAGE,
  type TableState,
} from "./table-params";

describe("parseTableParams", () => {
  it("returns safe defaults for empty params", () => {
    const s = parseTableParams(new URLSearchParams(""));
    expect(s.page).toBe(1);
    expect(s.perPage).toBe(20);
    expect(s.order).toBe("desc");
    expect(s.sort).toBe("");
    expect(s.q).toBe("");
    expect(s.filters).toEqual({});
  });

  it("parses page, per_page, sort, order, q", () => {
    const s = parseTableParams(
      new URLSearchParams("page=3&per_page=50&sort=amount&order=asc&q=makan")
    );
    expect(s.page).toBe(3);
    expect(s.perPage).toBe(50);
    expect(s.sort).toBe("amount");
    expect(s.order).toBe("asc");
    expect(s.q).toBe("makan");
  });

  it("clamps per_page and rejects invalid page", () => {
    expect(parseTableParams(new URLSearchParams("per_page=5")).perPage).toBe(10);
    expect(parseTableParams(new URLSearchParams("per_page=500")).perPage).toBe(100);
    expect(parseTableParams(new URLSearchParams("page=0")).page).toBe(1);
    expect(parseTableParams(new URLSearchParams("page=abc")).page).toBe(1);
  });

  it("parses filter_* params", () => {
    const s = parseTableParams(
      new URLSearchParams("filter_status=active&filter_type=expense")
    );
    expect(s.filters).toEqual({ status: "active", type: "expense" });
  });

  it("applies provided defaults", () => {
    const s = parseTableParams(new URLSearchParams(""), { sort: "created_at", perPage: 30 });
    expect(s.sort).toBe("created_at");
    expect(s.perPage).toBe(30);
  });
});

describe("buildTableParams", () => {
  it("omits default values for clean URLs", () => {
    const state: TableState = {
      page: DEFAULT_PAGE,
      perPage: DEFAULT_PER_PAGE,
      sort: "",
      order: "desc",
      sorts: [],
      q: "",
      filters: {},
    };
    expect(buildTableParams(state).toString()).toBe("");
  });

  it("serialises non-default state and filters", () => {
    const state: TableState = {
      page: 2,
      perPage: 50,
      sort: "amount",
      order: "asc",
      sorts: [{ column: "amount", order: "asc" }],
      q: "kopi",
      filters: { status: "active" },
    };
    const qs = buildTableParams(state).toString();
    const parsed = new URLSearchParams(qs);
    expect(parsed.get("page")).toBe("2");
    expect(parsed.get("per_page")).toBe("50");
    expect(parsed.get("sort")).toBe("amount");
    expect(parsed.get("order")).toBe("asc");
    expect(parsed.get("q")).toBe("kopi");
    expect(parsed.get("filter_status")).toBe("active");
  });

  it("round-trips through parse", () => {
    const state: TableState = {
      page: 4,
      perPage: 20,
      sort: "created_at",
      order: "desc",
      sorts: [{ column: "created_at", order: "desc" }],
      q: "abc",
      filters: { type: "income" },
    };
    const back = parseTableParams(buildTableParams(state));
    expect(back.page).toBe(4);
    expect(back.perPage).toBe(20);
    expect(back.sort).toBe("created_at");
    expect(back.order).toBe("desc");
    expect(back.q).toBe("abc");
    expect(back.filters).toEqual({ type: "income" });
  });

  it("serialises and round-trips a multi-column sort", () => {
    const state: TableState = {
      page: 1,
      perPage: 20,
      sort: "amount",
      order: "desc",
      sorts: [
        { column: "amount", order: "desc" },
        { column: "created_at", order: "asc" },
      ],
      q: "",
      filters: {},
    };
    const qs = buildTableParams(state).toString();
    const parsed = new URLSearchParams(qs);
    expect(parsed.get("sort")).toBe("amount,created_at");
    expect(parsed.get("order")).toBe("desc,asc");

    const back = parseTableParams(parsed);
    expect(back.sorts).toEqual([
      { column: "amount", order: "desc" },
      { column: "created_at", order: "asc" },
    ]);
    expect(back.sort).toBe("amount");
    expect(back.order).toBe("desc");
  });

  it("namespaces params when a prefix is set", () => {
    const state: TableState = {
      page: 2,
      perPage: 50,
      sort: "created_at",
      order: "asc",
      sorts: [{ column: "created_at", order: "asc" }],
      q: "x",
      filters: { status: "paid" },
    };
    const parsed = buildTableParams(state, { prefix: "payment" });
    expect(parsed.get("payment_page")).toBe("2");
    expect(parsed.get("payment_per_page")).toBe("50");
    expect(parsed.get("payment_sort")).toBe("created_at");
    expect(parsed.get("payment_order")).toBe("asc");
    expect(parsed.get("payment_q")).toBe("x");
    expect(parsed.get("payment_filter_status")).toBe("paid");

    const back = parseTableParams(parsed, { prefix: "payment" });
    expect(back.page).toBe(2);
    expect(back.filters).toEqual({ status: "paid" });
  });
});

describe("parseSortList", () => {
  it("parses a comma list with per-column directions", () => {
    expect(parseSortList("a,b", "asc,desc")).toEqual([
      { column: "a", order: "asc" },
      { column: "b", order: "desc" },
    ]);
  });

  it("reuses the last direction when fewer are given", () => {
    expect(parseSortList("a,b", "asc")).toEqual([
      { column: "a", order: "asc" },
      { column: "b", order: "asc" },
    ]);
  });

  it("falls back to the default when the list is empty", () => {
    expect(parseSortList("", "", { sort: "created_at", order: "desc" })).toEqual([
      { column: "created_at", order: "desc" },
    ]);
    expect(parseSortList("", "")).toEqual([]);
  });
});

describe("param helpers", () => {
  it("paramName / filterPrefix namespace correctly", () => {
    expect(paramName("page")).toBe("page");
    expect(paramName("page", "payment")).toBe("payment_page");
    expect(filterPrefix()).toBe("filter_");
    expect(filterPrefix("payment")).toBe("payment_filter_");
  });
});

describe("updateTableState", () => {
  const base: TableState = {
    page: 3,
    perPage: 20,
    sort: "created_at",
    order: "desc",
    sorts: [{ column: "created_at", order: "desc" }],
    q: "",
    filters: {},
  };

  it("changing page keeps other fields", () => {
    const next = updateTableState(base, { page: 5 });
    expect(next.page).toBe(5);
    expect(next.sort).toBe("created_at");
  });

  it("changing a non-page field resets page to 1", () => {
    expect(updateTableState(base, { sort: "amount" }).page).toBe(1);
    expect(updateTableState(base, { q: "x" }).page).toBe(1);
    expect(updateTableState(base, { perPage: 50 }).page).toBe(1);
  });

  it("keeps sorts in sync when patching sort/order", () => {
    const next = updateTableState(base, { sort: "amount", order: "asc" });
    expect(next.sorts).toEqual([{ column: "amount", order: "asc" }]);
    expect(next.sort).toBe("amount");
    expect(next.order).toBe("asc");
  });

  it("accepts an explicit multi-sort list", () => {
    const next = updateTableState(base, {
      sorts: [
        { column: "amount", order: "desc" },
        { column: "created_at", order: "asc" },
      ],
    });
    expect(next.sorts).toHaveLength(2);
    expect(next.sort).toBe("amount");
    expect(next.order).toBe("desc");
  });

  it("merges filters", () => {
    const withFilters = updateTableState(base, { filters: { status: "active" } });
    const next = updateTableState(withFilters, { filters: { type: "expense" } });
    expect(next.filters).toEqual({ status: "active", type: "expense" });
  });

  it("clears a filter when set to empty string", () => {
    const withFilters = updateTableState(base, { filters: { status: "active", type: "expense" } });
    const next = updateTableState(withFilters, { filters: { type: "" } });
    expect(next.filters).toEqual({ status: "active" });
  });
});

describe("helpers", () => {
  it("clampPerPage", () => {
    expect(clampPerPage(0)).toBe(20);
    expect(clampPerPage(NaN)).toBe(20);
    expect(clampPerPage(5)).toBe(10);
    expect(clampPerPage(1000)).toBe(100);
    expect(clampPerPage(42)).toBe(42);
  });

  it("normalizeOrder", () => {
    expect(normalizeOrder("ASC")).toBe("asc");
    expect(normalizeOrder("desc")).toBe("desc");
    expect(normalizeOrder(null)).toBe("desc");
    expect(normalizeOrder("weird")).toBe("desc");
  });
});
