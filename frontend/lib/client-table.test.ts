import { describe, it, expect } from "vitest";
import {
  sortRows,
  searchRows,
  filterRows,
  paginate,
  totalPages,
  applyTableQuery,
  type TableQuery,
} from "./client-table";

const rows = [
  { category_name: "Makanan", total: 300, type: "expense" },
  { category_name: "Transportasi", total: 100, type: "expense" },
  { category_name: "Gaji", total: 500, type: "income" },
  { category_name: "Café", total: 50, type: "expense" },
];

describe("sortRows", () => {
  it("sorts numbers ascending and descending", () => {
    expect(sortRows(rows, "total", "asc").map((r) => r.total)).toEqual([50, 100, 300, 500]);
    expect(sortRows(rows, "total", "desc").map((r) => r.total)).toEqual([500, 300, 100, 50]);
  });

  it("sorts strings with locale awareness", () => {
    const names = sortRows(rows, "category_name", "asc").map((r) => r.category_name);
    expect(names).toEqual(["Café", "Gaji", "Makanan", "Transportasi"]);
  });

  it("does not mutate the input", () => {
    const before = rows.map((r) => r.total);
    sortRows(rows, "total", "desc");
    expect(rows.map((r) => r.total)).toEqual(before);
  });

  it("returns a copy when no sort is given", () => {
    const out = sortRows(rows, "", "asc");
    expect(out).toEqual(rows);
    expect(out).not.toBe(rows);
  });

  it("pushes null/undefined values last", () => {
    const withNull = [{ v: 2 }, { v: null }, { v: 1 }];
    expect(sortRows(withNull, "v", "asc").map((r) => r.v)).toEqual([1, 2, null]);
  });
});

describe("searchRows", () => {
  it("matches case-insensitively", () => {
    expect(searchRows(rows, ["category_name"], "makan").map((r) => r.category_name)).toEqual(["Makanan"]);
  });

  it("matches across diacritics", () => {
    // "cafe" should find "Café".
    expect(searchRows(rows, ["category_name"], "cafe").map((r) => r.category_name)).toEqual(["Café"]);
  });

  it("searches multiple keys", () => {
    const out = searchRows(rows, ["category_name", "type"], "income");
    expect(out).toHaveLength(1);
    expect(out[0].category_name).toBe("Gaji");
  });

  it("returns all rows for an empty query", () => {
    expect(searchRows(rows, ["category_name"], "  ")).toHaveLength(rows.length);
  });
});

describe("filterRows", () => {
  it("applies exact-match filters", () => {
    expect(filterRows(rows, { type: "expense" })).toHaveLength(3);
  });

  it("ignores empty filters", () => {
    expect(filterRows(rows, { type: "" })).toHaveLength(rows.length);
    expect(filterRows(rows, {})).toHaveLength(rows.length);
  });

  it("combines multiple filters with AND", () => {
    expect(filterRows(rows, { type: "expense", category_name: "Makanan" })).toHaveLength(1);
  });
});

describe("paginate / totalPages", () => {
  it("slices the requested page", () => {
    expect(paginate([1, 2, 3, 4, 5], 1, 2)).toEqual([1, 2]);
    expect(paginate([1, 2, 3, 4, 5], 2, 2)).toEqual([3, 4]);
    expect(paginate([1, 2, 3, 4, 5], 3, 2)).toEqual([5]);
  });

  it("clamps a page below 1", () => {
    expect(paginate([1, 2, 3], 0, 2)).toEqual([1, 2]);
  });

  it("computes page counts", () => {
    expect(totalPages(0, 20)).toBe(1);
    expect(totalPages(20, 20)).toBe(1);
    expect(totalPages(21, 20)).toBe(2);
    expect(totalPages(100, 20)).toBe(5);
  });
});

describe("applyTableQuery", () => {
  const base: TableQuery = { page: 1, perPage: 20, sort: "", order: "desc", q: "", filters: {} };

  it("runs search → filter → sort → paginate", () => {
    const { page, total } = applyTableQuery(
      rows,
      { ...base, q: "an", filters: { type: "expense" }, sort: "total", order: "asc" },
      ["category_name"]
    );
    // "an" matches Makanan, Transportasi; filtered to expense keeps both;
    // sorted by total asc => Transportasi(100), Makanan(300).
    expect(total).toBe(2);
    expect(page.map((r) => r.category_name)).toEqual(["Transportasi", "Makanan"]);
  });

  it("reports the total before pagination", () => {
    const { page, total } = applyTableQuery(rows, { ...base, perPage: 2 }, ["category_name"]);
    expect(total).toBe(4);
    expect(page).toHaveLength(2);
  });
});
