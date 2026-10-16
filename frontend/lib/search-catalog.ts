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
  LayoutDashboard,
  Users,
  ArrowUpCircle,
  ArrowDownCircle,
  Bot,
  Coins,
  Repeat,
  FileCheck,
  Package,
  History,
  Cpu,
  Settings,
  Activity,
  Gauge,
  SlidersHorizontal,
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
  /** Official Lucide icon name string for serialization / reflection */
  iconName?: string;
  /** Extra terms (synonyms, English, Indonesian) matched alongside the label. */
  keywords: string[];
  /** Optional short description shown under the label. */
  description?: string;
  /** Target roles that can see and access this search entry. */
  roles?: ("user" | "admin" | "super_admin")[];
  /** Optional action identifier for instantaneous client actions */
  actionId?: string;
}

/**
 * The catalog the header search matches against. Kept in one place so adding a
 * page or a feature makes it discoverable from the search box automatically.
 */
export const SEARCH_ENTRIES: SearchEntry[] = [
  // ── User Modules (pages) ────────────────────────────────────────
  { id: "dashboard", label: "Dashboard", group: "Modul", href: "/user", icon: Home, iconName: "Home", keywords: ["beranda", "home", "ringkasan", "overview", "ikhtisar"], description: "Ringkasan keuangan Anda", roles: ["user"] },
  { id: "transactions", label: "Transaksi", group: "Modul", href: "/user/transactions", icon: ArrowLeftRight, iconName: "ArrowLeftRight", keywords: ["transaksi", "transaction", "catatan", "riwayat", "pengeluaran", "pemasukan"], description: "Catat & kelola transaksi", roles: ["user"] },
  { id: "ai-chat", label: "AI Chat", group: "Modul", href: "/user/ai", icon: MessageSquare, iconName: "MessageSquare", keywords: ["ai", "chat", "asisten", "assistant", "tanya", "bot"], description: "Asisten keuangan AI", roles: ["user"] },
  { id: "reports", label: "Laporan", group: "Modul", href: "/user/reports", icon: FileText, iconName: "FileText", keywords: ["laporan", "report", "ekspor", "export", "unduh"], description: "Laporan keuangan", roles: ["user"] },
  { id: "subscription", label: "Langganan", group: "Modul", href: "/user/subscription", icon: CreditCard, iconName: "CreditCard", keywords: ["langganan", "subscription", "paket", "plan", "upgrade", "harga", "billing", "tagihan", "kuota"], description: "Paket, kuota & pembayaran", roles: ["user"] },
  { id: "profile", label: "Profil & Pengaturan", group: "Modul", href: "/user/profile", icon: User, iconName: "User", keywords: ["profil", "profile", "akun", "account", "nama", "telepon", "avatar", "foto", "pengaturan", "settings", "preferensi", "konfigurasi", "password", "tema", "bahasa", "notifikasi"], description: "Data akun & preferensi", roles: ["user", "admin", "super_admin"] },

  // ── User Features (on a page) ───────────────────────────────────
  { id: "feat-new-tx", label: "Tambah Transaksi", group: "Fitur", href: "/user/transactions?new=1", icon: Plus, iconName: "Plus", keywords: ["tambah", "baru", "catat", "transaksi", "input"], description: "Catat transaksi baru", roles: ["user"] },
  { id: "feat-ocr", label: "Pindai Struk (OCR)", group: "Fitur", href: "/user/transactions?scan=1", icon: ScanLine, iconName: "ScanLine", keywords: ["scan", "struk", "ocr", "receipt", "foto", "kamera", "pindai"], description: "Ambil data dari foto struk", roles: ["user"] },
  { id: "feat-voice", label: "Input Suara", group: "Fitur", href: "/user/ai", icon: Mic, iconName: "Mic", keywords: ["suara", "voice", "audio", "rekam", "bicara", "mic", "dikte"], description: "Dikte lewat suara di AI Chat", roles: ["user"] },
  { id: "feat-export", label: "Ekspor Laporan", group: "Fitur", href: "/user/reports", icon: FileText, iconName: "FileText", keywords: ["ekspor", "export", "unduh", "download", "laporan", "pdf", "csv"], description: "Unduh laporan keuangan", roles: ["user"] },
  { id: "feat-privacy", label: "Privasi & UU PDP", group: "Fitur", href: "/user/profile#privacy", icon: ShieldCheck, iconName: "ShieldCheck", keywords: ["privasi", "pdp", "hapus akun", "ekspor data", "gdpr", "kepatuhan"], description: "Hak data pribadi Anda", roles: ["user", "admin", "super_admin"] },
  { id: "feat-usage", label: "Penggunaan", group: "Modul", href: "/user/usage", icon: BarChart3, iconName: "BarChart3", keywords: ["penggunaan", "usage", "pemakaian", "kuota", "quota", "token", "statistik", "grafik", "riwayat pemakaian"], description: "Statistik & riwayat pemakaian", roles: ["user"] },
  { id: "feat-quota", label: "Pemakaian Kuota", group: "Fitur", href: "/user/usage", icon: BarChart3, iconName: "BarChart3", keywords: ["kuota", "quota", "pemakaian", "usage", "token", "limit", "sisa"], description: "Sisa kuota Anda", roles: ["user"] },
  { id: "feat-plans", label: "Tingkatkan Paket", group: "Fitur", href: "/user/subscription", icon: Sparkles, iconName: "Sparkles", keywords: ["upgrade", "tingkatkan", "paket", "pro", "business", "berlangganan"], description: "Naikkan paket langganan", roles: ["user"] },
  { id: "feat-invoices", label: "Riwayat Pembayaran", group: "Fitur", href: "/user/subscription?tab=payment", icon: Receipt, iconName: "Receipt", keywords: ["invoice", "faktur", "pembayaran", "payment", "riwayat", "transfer", "bukti"], description: "Faktur & bukti transfer", roles: ["user"] },
  { id: "feat-savings", label: "Tabungan", group: "Fitur", href: "/user/transactions?filter_type=savings", icon: PiggyBank, iconName: "PiggyBank", keywords: ["tabungan", "savings", "menabung", "investasi"], description: "Lihat transaksi tabungan", roles: ["user"] },

  // ── Admin Modules & Features ─────────────────────────────────────
  { id: "adm-dashboard", label: "Dashboard Admin", group: "Modul", href: "/admin", icon: LayoutDashboard, iconName: "LayoutDashboard", keywords: ["admin", "dashboard", "beranda", "ringkasan", "transaksi", "transaksi platform", "kpi", "statistik", "home"], description: "Ringkasan metrik platform admin", roles: ["admin", "super_admin"] },
  { id: "adm-users", label: "Manajemen Pengguna", group: "Modul", href: "/admin/users", icon: Users, iconName: "Users", keywords: ["user", "users", "pengguna", "akun", "suspend", "aktivasi", "daftar pengguna", "daftar user"], description: "Kelola status dan akun pengguna", roles: ["admin", "super_admin"] },
  { id: "adm-cat-income", label: "Kategori Pemasukan", group: "Fitur", href: "/admin/categories/income", icon: ArrowUpCircle, iconName: "ArrowUpCircle", keywords: ["kategori", "pemasukan", "income", "pendapatan", "pos masuk", "gaji", "dividen", "omset"], description: "Master kategori pemasukan", roles: ["admin", "super_admin"] },
  { id: "adm-cat-expense", label: "Kategori Pengeluaran", group: "Fitur", href: "/admin/categories/expense", icon: ArrowDownCircle, iconName: "ArrowDownCircle", keywords: ["kategori", "pengeluaran", "expense", "biaya", "pos keluar", "belanja", "pos biaya"], description: "Master kategori pengeluaran", roles: ["admin", "super_admin"] },
  { id: "adm-cat-savings", label: "Kategori Tabungan", group: "Fitur", href: "/admin/categories/savings", icon: PiggyBank, iconName: "PiggyBank", keywords: ["kategori", "tabungan", "savings", "investasi", "pos simpanan", "pos tabungan", "alokasi"], description: "Master kategori tabungan", roles: ["admin", "super_admin"] },
  { id: "adm-providers", label: "Monitoring Provider", group: "Modul", href: "/admin/providers", icon: Bot, iconName: "Bot", keywords: ["provider", "ai", "payment", "gateway", "midtrans", "xendit", "openai", "anthropic"], description: "Status gateway pembayaran & AI", roles: ["admin", "super_admin"] },

  // ── Super Admin Modules & Features ───────────────────────────────
  { id: "sa-dashboard", label: "Dashboard Super Admin", group: "Modul", href: "/super-admin", icon: LayoutDashboard, iconName: "LayoutDashboard", keywords: ["super", "admin", "mrr", "kpi", "overview", "finansial", "ringkasan platform"], description: "Pusat komando dan KPI eksekutif", roles: ["super_admin"] },
  { id: "sa-tokens", label: "Token & Kuota AI", group: "Modul", href: "/super-admin/tokens", icon: Coins, iconName: "Coins", keywords: ["token", "kuota", "ai", "ledger", "kredit", "debit", "adjust", "saldo", "saldo token", "penyesuaian"], description: "Saldo & penyesuaian token AI pengguna", roles: ["super_admin"] },
  { id: "sa-subscriptions", label: "Manajemen Langganan", group: "Modul", href: "/super-admin/subscriptions", icon: Repeat, iconName: "Repeat", keywords: ["langganan", "subscription", "paket", "downgrade", "perpanjang", "extend", "cancel", "langganan aktif"], description: "Siklus hidup langganan pengguna", roles: ["super_admin"] },
  { id: "sa-payments", label: "Pembayaran & Transaksi", group: "Modul", href: "/super-admin/payments", icon: CreditCard, iconName: "CreditCard", keywords: ["pembayaran", "payments", "transaksi", "faktur", "invoice", "refund", "mutasi", "transfer manual", "bukti bayar", "verifikasi"], description: "Histori seluruh transaksi pembayaran", roles: ["super_admin"] },
  { id: "sa-payment-review", label: "Review Pembayaran Manual", group: "Fitur", href: "/super-admin/payments/review", icon: FileCheck, iconName: "FileCheck", keywords: ["review", "verifikasi", "transfer", "bukti bayar", "approval", "manual"], description: "Antrean verifikasi bukti transfer", roles: ["super_admin"] },
  { id: "sa-plans", label: "Katalog & Manajemen Paket", group: "Modul", href: "/super-admin/plans", icon: Package, iconName: "Package", keywords: ["plans", "paket", "katalog", "entitlement", "harga", "limit", "katalog paket", "batas kuota"], description: "Konfigurasi paket & entitlement", roles: ["super_admin"] },
  { id: "sa-audit", label: "Jejak Audit (Audit Log)", group: "Modul", href: "/super-admin/audit", icon: History, iconName: "History", keywords: ["audit", "log", "aktivitas", "jejak", "diff", "pdp", "kepatuhan", "log aktivitas", "diff json", "uu pdp", "inspeksi"], description: "Pencatatan mutasi sistem & UU PDP", roles: ["super_admin"] },
  { id: "sa-ai-gateway", label: "Operasional AI Gateway", group: "Modul", href: "/super-admin/ai-gateway", icon: Cpu, iconName: "Cpu", keywords: ["ai", "gateway", "circuit breaker", "kill switch", "routing", "latency", "endpoint ai", "latensi"], description: "Kendali pool endpoint & routing AI", roles: ["super_admin"] },
  { id: "sa-settings", label: "Pengaturan Platform Global", group: "Modul", href: "/super-admin/settings", icon: Settings, iconName: "Settings", keywords: ["pengaturan", "settings", "konfigurasi", "feature flag", "gateway", "email", "storage", "konfigurasi global", "payment gateway"], description: "Konfigurasi sistem & feature flags", roles: ["super_admin"] },
  { id: "sa-metrics", label: "Metrik & Performa Sistem", group: "Modul", href: "/super-admin/metrics", icon: Activity, iconName: "Activity", keywords: ["metrik", "metrics", "performa", "kpi", "latensi", "sistem"], description: "Analisis performa & KPI agregat", roles: ["super_admin"] },
  { id: "sa-kuota", label: "Monitoring Kuota AI", group: "Modul", href: "/super-admin/kuota", icon: Gauge, iconName: "Gauge", keywords: ["kuota", "quota", "monitor", "pemakaian", "konsumsi", "top users"], description: "Alokasi & pemakaian kuota AI", roles: ["super_admin"] },
  { id: "sa-spk-metrics", label: "Metrik SPK", group: "Fitur", href: "/super-admin/spk-metrics", icon: SlidersHorizontal, iconName: "SlidersHorizontal", keywords: ["spk", "metrik", "bobot", "skor", "kriteria"], description: "Metrik sistem pendukung keputusan", roles: ["super_admin"] },
];

/**
 * normalize folds case and strips diacritics so "Profil" matches "profil" and
 * "Aktivitas" matches "aktivitas" regardless of accents.
 */
function normalize(s: string): string {
  return s.normalize("NFD").replace(/\p{Diacritic}/gu, "").toLowerCase().trim();
}

/**
 * searchEntries returns entries matching `query` for the given `role`, ranked so
 * the most relevant come first:
 *   1. label starts with the query (strongest),
 *   2. label contains the query,
 *   3. a keyword starts with the query,
 *   4. a keyword contains the query,
 *   5. the description contains the query.
 * Ties break by group order (Modul, Fitur) then label, so the list is stable.
 */
export function searchEntries(
  query: string,
  entries: SearchEntry[] = SEARCH_ENTRIES,
  role: string = "user"
): SearchEntry[] {
  const q = normalize(query);
  if (!q) return [];

  const eligible = entries.filter((e) => !e.roles || e.roles.includes(role as "user" | "admin" | "super_admin"));
  const scored: { entry: SearchEntry; score: number }[] = [];
  for (const entry of eligible) {
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
