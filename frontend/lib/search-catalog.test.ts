import { describe, it, expect } from "vitest";
import { searchEntries, SEARCH_ENTRIES } from "./search-catalog";

describe("searchEntries", () => {
  it("returns nothing for an empty query", () => {
    expect(searchEntries("")).toEqual([]);
    expect(searchEntries("   ")).toEqual([]);
  });

  it("matches a module by its label", () => {
    const results = searchEntries("transaksi");
    expect(results.length).toBeGreaterThan(0);
    expect(results[0].href).toBe("/user/transactions");
  });

  it("is case-insensitive", () => {
    expect(searchEntries("PROFIL")[0].href).toBe("/user/profile");
    expect(searchEntries("profil")[0].href).toBe("/user/profile");
  });

  it("finds the merged profile/settings page by either term", () => {
    // Profile and settings are one page; both queries must reach it.
    expect(searchEntries("profil")[0].href).toBe("/user/profile");
    expect(searchEntries("pengaturan")[0].href).toBe("/user/profile");
  });

  it("matches by keyword (synonym), not only the label", () => {
    // "unduh" is a keyword of the export/report features, not a label.
    const hrefs = searchEntries("unduh").map((e) => e.href);
    expect(hrefs.some((h) => h.includes("/user/reports"))).toBe(true);
  });

  it("matches by English synonym", () => {
    // "subscription" is a keyword of the Langganan module.
    const results = searchEntries("subscription");
    expect(results.some((e) => e.href === "/user/settings/subscription")).toBe(true);
  });

  it("ranks a label prefix above a keyword match", () => {
    // "profil" is a label prefix; nothing else should outrank it.
    const results = searchEntries("profil");
    expect(results[0].label).toBe("Profil & Pengaturan");
  });

  it("finds features as well as modules", () => {
    const results = searchEntries("ocr");
    expect(results.some((e) => e.group === "Fitur")).toBe(true);
  });

  it("returns an empty list when nothing matches", () => {
    expect(searchEntries("zzzznotathing")).toEqual([]);
  });

  it("every catalog entry has a unique id and an href", () => {
    const ids = SEARCH_ENTRIES.map((e) => e.id);
    expect(new Set(ids).size).toBe(ids.length);
    for (const e of SEARCH_ENTRIES) {
      expect(e.href.startsWith("/")).toBe(true);
      expect(e.keywords.length).toBeGreaterThan(0);
    }
  });
});
