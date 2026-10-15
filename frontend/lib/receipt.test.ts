import { describe, it, expect } from "vitest";
import {
  normalizeReceiptType,
  matchCategory,
  buildReceiptTransaction,
  receiptTypeLabel,
  formatRupiahIDR,
  type ReceiptResult,
} from "./receipt";
import type { AdminCategory } from "@/hooks/useAdminCategories";

const cat = (id: string, name: string): AdminCategory => ({ id, name, icon: "circle", color: "#000", is_active: true });

const lists = {
  expense: [cat("e1", "Transportasi"), cat("e2", "Makanan & Minuman"), cat("e9", "Lain-lain")],
  income: [cat("i1", "Gaji"), cat("i9", "Lainnya")],
  savings: [cat("s1", "Pendidikan"), cat("s2", "Pensiun"), cat("s9", "Lainnya")],
};

describe("normalizeReceiptType", () => {
  it("maps english and indonesian type names", () => {
    expect(normalizeReceiptType("income")).toBe("income");
    expect(normalizeReceiptType("PEMASUKAN")).toBe("income");
    expect(normalizeReceiptType("savings")).toBe("savings");
    expect(normalizeReceiptType("tabungan")).toBe("savings");
    expect(normalizeReceiptType("expense")).toBe("expense");
    expect(normalizeReceiptType("pengeluaran")).toBe("expense");
  });

  it("defaults unknown/missing to expense", () => {
    expect(normalizeReceiptType(undefined)).toBe("expense");
    expect(normalizeReceiptType("")).toBe("expense");
    expect(normalizeReceiptType("nonsense")).toBe("expense");
  });
});

describe("matchCategory", () => {
  it("matches exactly (case-insensitive)", () => {
    expect(matchCategory("transportasi", "expense", lists)?.id).toBe("e1");
  });

  it("matches by substring into the db name", () => {
    expect(matchCategory("Makanan", "expense", lists)?.id).toBe("e2");
  });

  it("uses the list for the transaction type", () => {
    expect(matchCategory("Pendidikan", "savings", lists)?.id).toBe("s1");
    expect(matchCategory("Gaji", "income", lists)?.id).toBe("i1");
  });

  it("falls back to the generic bucket for unknown names", () => {
    expect(matchCategory("Zzz", "expense", lists)?.id).toBe("e9");
  });

  it("returns undefined when the type has no categories", () => {
    expect(matchCategory("Gaji", "income", {})).toBeUndefined();
  });
});

describe("buildReceiptTransaction", () => {
  it("builds a savings payload with category id, description and date", () => {
    const data: ReceiptResult = {
      success: true,
      type: "savings",
      category: "Pensiun",
      total: 851000,
      description: "Tabungan pendidikan anak",
      date: "2026-06-14",
    };
    const { payload, category } = buildReceiptTransaction(data, lists);
    expect(category.id).toBe("s2");
    expect(payload.type).toBe("savings");
    expect(payload.amount).toBe(851000);
    expect(payload.savings_category_id).toBe("s2");
    expect(payload.description).toBe("Tabungan pendidikan anak");
    expect(payload.source).toBe("receipt");
    // The date is anchored at UTC midnight so the calendar day is preserved.
    expect((payload.transaction_date as string).startsWith("2026-06-14")).toBe(true);
  });

  it("falls back to merchant when description is missing", () => {
    const { payload } = buildReceiptTransaction(
      { success: true, type: "expense", category: "Transportasi", total: 239000, merchant: "MRT" },
      lists
    );
    expect(payload.description).toBe("MRT");
    expect(payload.expense_category_id).toBe("e1");
  });

  it("throws when the amount is missing", () => {
    expect(() => buildReceiptTransaction({ success: true, type: "expense", category: "Transportasi" }, lists)).toThrow(
      /Jumlah tidak terbaca/
    );
  });

  it("throws when no category matches and there is no generic bucket", () => {
    expect(() =>
      buildReceiptTransaction({ success: true, type: "income", category: "Zzz", total: 1000 }, { income: [cat("i1", "Gaji")] })
    ).toThrow(/tidak ditemukan/);
  });
});

describe("labels and formatting", () => {
  it("labels each type", () => {
    expect(receiptTypeLabel("income")).toBe("Pemasukan");
    expect(receiptTypeLabel("savings")).toBe("Tabungan");
    expect(receiptTypeLabel("expense")).toBe("Pengeluaran");
  });

  it("formats rupiah without decimals", () => {
    expect(formatRupiahIDR(851000)).toMatch(/851\.000/);
    expect(formatRupiahIDR(0)).toMatch(/0/);
  });
});
