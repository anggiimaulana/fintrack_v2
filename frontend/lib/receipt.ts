import type { AdminCategory } from "@/hooks/useAdminCategories";
import type { TransactionData } from "@/components/transactions/TransactionRow";

/**
 * Result shape returned by POST /api/v1/ai/receipt. The backend now fills
 * `description` (from the model or the merchant) and may include `time`, so
 * the scanner can prefill the same fields the transaction table/detail popup
 * show.
 */
export interface ReceiptResult {
  success: boolean;
  message?: string;
  merchant?: string;
  date?: string;
  time?: string;
  total?: number;
  category?: string;
  type?: string;
  description?: string;
  items?: { name: string; quantity: number; unit_price: number; total: number }[];
  raw_text?: string;
  transactions?: ReceiptTransaction[];
  [key: string]: unknown;
}

export interface ReceiptTransaction {
  type?: string;
  total?: number;
  category?: string;
  description?: string;
}

export type ReceiptType = "income" | "expense" | "savings";

/** Normalize the model's free-form type into the three app transaction types. */
export function normalizeReceiptType(raw: unknown): ReceiptType {
  const t = String(raw || "").toLowerCase().trim();
  if (t === "income" || t === "pemasukan") return "income";
  if (t === "savings" || t === "tabungan" || t === "saving") return "savings";
  return "expense";
}

/** The category list that applies to a given transaction type. */
export function categoriesForType(
  type: ReceiptType,
  lists: { income?: AdminCategory[]; expense?: AdminCategory[]; savings?: AdminCategory[] }
): AdminCategory[] {
  if (type === "income") return lists.income || [];
  if (type === "savings") return lists.savings || [];
  return lists.expense || [];
}

/**
 * Match a model-provided category name onto a real category row: exact match
 * (case-insensitive) first, then a substring match either way, then the generic
 * "lain" bucket. Mirrors the backend's snapCategory so client and server agree.
 */
export function matchCategory(
  name: string | undefined,
  type: ReceiptType,
  lists: { income?: AdminCategory[]; expense?: AdminCategory[]; savings?: AdminCategory[] }
): AdminCategory | undefined {
  const list = categoriesForType(type, lists);
  const want = (name || "").toLowerCase().trim();
  if (want) {
    const exact = list.find((c) => c.name?.toLowerCase() === want);
    if (exact) return exact;
    const partial = list.find((c) => {
      const n = c.name?.toLowerCase() || "";
      return n.includes(want) || want.includes(n);
    });
    if (partial) return partial;
  }
  // Generic bucket, tolerating "Lainnya" vs "Lain-lain".
  return list.find((c) => (c.name || "").toLowerCase().includes("lain"));
}

export interface ReceiptTransactionPayload {
  type: ReceiptType;
  amount: number;
  expense_category_id?: string;
  income_category_id?: string;
  savings_category_id?: string;
  description?: string;
  transaction_date: string;
  source: string;
  [key: string]: unknown;
}

/**
 * Build the transaction payload for a scanned receipt. Returns the payload plus
 * the resolved category, or an error explaining why it cannot be saved (no
 * amount / no matching category) so the caller can show a precise message.
 */
export function buildReceiptTransaction(
  data: ReceiptResult,
  lists: { income?: AdminCategory[]; expense?: AdminCategory[]; savings?: AdminCategory[] }
): { payload: ReceiptTransactionPayload; category: AdminCategory } {
  const type = normalizeReceiptType(data.type);
  const category = matchCategory(data.category, type, lists);
  const amount = Number(data.total) || 0;

  if (!amount) throw new Error("Jumlah tidak terbaca pada struk.");
  if (!category) throw new Error(`Kategori "${data.category || "-"}" tidak ditemukan.`);

  const description = (data.description || data.merchant || "").trim() || undefined;

  // transaction_date is a DATE column. Anchor the receipt's date at UTC
  // midnight so the calendar day survives the timezone conversion on the way
  // in (a local-midnight timestamp can roll back a day on a UTC server). Fall
  // back to "now" only when the model could not read a date.
  const txDate = data.date
    ? new Date(`${data.date}T00:00:00Z`).toISOString()
    : new Date().toISOString();

  const categoryField =
    type === "income" ? "income_category_id" : type === "savings" ? "savings_category_id" : "expense_category_id";

  const payload: ReceiptTransactionPayload = {
    type,
    amount,
    [categoryField]: category.id,
    description,
    transaction_date: txDate,
    source: "receipt",
  };

  return { payload, category };
}

/** Human-readable label for a transaction type (used in toasts/messages). */
export function receiptTypeLabel(type: ReceiptType): string {
  if (type === "income") return "Pemasukan";
  if (type === "savings") return "Tabungan";
  return "Pengeluaran";
}

/** Format an amount as Indonesian Rupiah without decimals. */
export function formatRupiahIDR(amount: number): string {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(amount || 0);
}
