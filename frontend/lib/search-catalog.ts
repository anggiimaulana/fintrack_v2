import type { LucideIcon } from "lucide-react";
import {
  Home,
  ArrowLeftRight,
  MessageSquare,
  FileText,
  CreditCard,
  User,
  Mic,
  ScanLine,
  Receipt,
  Plus,
  ShieldCheck,
  Sparkles,
  PiggyBank,
  BarChart3,
} from "lucide-react";

/**
 * A searchable entry in the header command palette. It points at a route
 * (a page) or a feature on a page (carried as a query param / anchor).
 */
export interface SearchEntry {
  /** Stable id (used as the React key and for tests). */
  id: string;
  /** Display label, e.g. "Transaksi". */
  label: string;
  /** Grouping shown in the results list, e.g. "Modul" or "Fitur". */
  group: "Modul" | "Fitur" | "Aksi";
  /** Destination route. */
  href: string;
  icon: LucideIcon;
  /** Extra terms (synonyms, English, Indonesian) matched alongside the label. */
  keywords: string[];
  /** Optional short description shown under the label. */
  description?: string;
}

/**
 * The catalog the header search matches against. Kept in one place so adding a
 * page or a feature makes it discoverable from the search box automatically.
 */
export const SEARCH_ENTRIES: SearchEntry[] = [
  // ── Modules (pages) ────────────────────────────────────────────
  { id: "dashboard", label: "Dashboard", group: "Modul", href: "/user", icon: Home, keywords: ["beranda", "home", "ringkasan", "overview", "ikhtisar"], description: "Ringkasan keuangan Anda" },
  { id: "transactions", label: "Transaksi", group: "Modul", href: "/user/transactions", icon: ArrowLeftRight, keywords: ["transaksi", "transaction", "catatan", "riwayat", "pengeluaran", "pemasukan"], description: "Catat & kelola transaksi" },
  { id: "ai-chat", label: "AI Chat", group: "Modul", href: "/user/ai", icon: MessageSquare, keywords: ["ai", "chat", "asisten", "assistant", "tanya", "bot"], description: "Asisten keuangan AI" },
  { id: "reports", label: "Laporan", group: "Modul", href: "/user/reports", icon: FileText, keywords: ["laporan", "report", "ekspor", "export", "unduh"], description: "Laporan keuangan" },
  { id: "subscription", label: "Langganan", group: "Modul", href: "/user/subscription", icon: CreditCard, keywords: ["langganan", "subscription", "paket", "plan", "upgrade", "harga", "billing", "tagihan", "kuota"], description: "Paket, kuota & pembayaran" },
  { id: "profile", label: "Profil & Pengaturan", group: "Modul", href: "/user/profile", icon: User, keywords: ["profil", "profile", "akun", "account", "nama", "telepon", "avatar", "foto", "pengaturan", "settings", "preferensi", "konfigurasi", "password", "tema", "bahasa", "notifikasi"], description: "Data akun & preferensi" },

  // ── Features (on a page) ───────────────────────────────────────
  { id: "feat-new-tx", label: "Tambah Transaksi", group: "Fitur", href: "/user/transactions?new=1", icon: Plus, keywords: ["tambah", "baru", "catat", "transaksi", "input"], description: "Catat transaksi baru" },
  { id: "feat-ocr", label: "Pindai Struk (OCR)", group: "Fitur", href: "/user/transactions?scan=1", icon: ScanLine, keywords: ["scan", "struk", "ocr", "receipt", "foto", "kamera", "pindai"], description: "Ambil data dari foto struk" },
  { id: "feat-voice", label: "Input Suara", group: "Fitur", href: "/user/ai", icon: Mic, keywords: ["suara", "voice", "audio", "rekam", "bicara", "mic", "dikte"], description: "Dikte lewat suara di AI Chat" },
  { id: "feat-export", label: "Ekspor Laporan", group: "Fitur", href: "/user/reports", icon: FileText, keywords: ["ekspor", "export", "unduh", "download", "laporan", "pdf", "csv"], description: "Unduh laporan keuangan" },
  { id: "feat-privacy", label: "Privasi & UU PDP", group: "Fitur", href: "/user/profile#privacy", icon: ShieldCheck, keywords: ["privasi", "pdp", "hapus akun", "ekspor data", "gdpr", "kepatuhan"], description: "Hak data pribadi Anda" },
  { id: "feat-usage", label: "Penggunaan", group: "Modul", href: "/user/usage", icon: BarChart3, keywords: ["penggunaan", "usage", "pemakaian", "kuota", "quota", "token", "statistik", "grafik", "riwayat pemakaian"], description: "Statistik & riwayat pemakaian" },
  { id: "feat-quota", label: "Pemakaian Kuota", group: "Fitur", href: "/user/usage", icon: BarChart3, keywords: ["kuota", "quota", "pemakaian", "usage", "token", "limit", "sisa"], description: "Sisa kuota Anda" },
  { id: "feat-plans", label: "Tingkatkan Paket", group: "Fitur", href: "/user/subscription", icon: Sparkles, keywords: ["upgrade", "tingkatkan", "paket", "pro", "business", "berlangganan"], description: "Naikkan paket langganan" },
  { id: "feat-invoices", label: "Riwayat Pembayaran", group: "Fitur", href: "/user/subscription?tab=payment", icon: Receipt, keywords: ["invoice", "faktur", "pembayaran", "payment", "riwayat", "transfer", "bukti"], description: "Faktur & bukti transfer" },
  { id: "feat-savings", label: "Tabungan", group: "Fitur", href: "/user/transactions?filter_type=savings", icon: PiggyBank, keywords: ["tabungan", "savings", "menabung", "investasi"], description: "Lihat transaksi tabungan" },
];

/**
 * normalize folds case and strips diacritics so "Profil" matches "profil" and
 * "Aktivitas" matches "aktivitas" regardless of accents.
 */
function normalize(s: string): string {
  return s.normalize("NFD").replace(/\p{Diacritic}/gu, "").toLowerCase().trim();
}

/**
 * searchEntries returns entries matching `query`, ranked so the most relevant
 * come first:
 *   1. label starts with the query (strongest),
 *   2. label contains the query,
 *   3. a keyword starts with the query,
 *   4. a keyword contains the query,
 *   5. the description contains the query.
 * Ties break by group order (Modul, Fitur) then label, so the list is stable.
 * An empty query returns nothing (the caller shows recent/all instead).
 */
export function searchEntries(query: string, entries: SearchEntry[] = SEARCH_ENTRIES): SearchEntry[] {
  const q = normalize(query);
  if (!q) return [];

  const scored: { entry: SearchEntry; score: number }[] = [];
  for (const entry of entries) {
    const label = normalize(entry.label);
    const keywords = entry.keywords.map(normalize);
    const description = normalize(entry.description ?? "");

    let score = -1;
    if (label.startsWith(q)) score = 5;
    else if (label.includes(q)) score = 4;
    else if (keywords.some((k) => k.startsWith(q))) score = 3;
    else if (keywords.some((k) => k.includes(q))) score = 2;
    else if (description.includes(q)) score = 1;

    if (score > 0) scored.push({ entry, score });
  }

  const groupRank: Record<SearchEntry["group"], number> = { Modul: 0, Fitur: 1, Aksi: 2 };
  scored.sort((a, b) => {
    if (a.score !== b.score) return b.score - a.score;
    if (groupRank[a.entry.group] !== groupRank[b.entry.group]) return groupRank[a.entry.group] - groupRank[b.entry.group];
    return a.entry.label.localeCompare(b.entry.label);
  });

  return scored.map((s) => s.entry);
}
