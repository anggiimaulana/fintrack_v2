import type { PublicPlan } from "@/lib/plans-types";

/**
 * Sample public catalog for landing/pricing component tests.
 *
 * Values mirror the real seed (`database/seeds/0002_subscription_token_seeds.sql`)
 * and the shape returned by `GET /api/v1/public/plans` so components render
 * exactly what production data would produce — including `bank_sync: false`
 * (present in the data, never advertised by the UI) and the unenforced
 * `max_categories`/`max_wallets` limits.
 *
 * One deliberate deviation: `tax_enabled` is set `true` on the paid plans to
 * exercise the separate tax line. The shipped default is `false`
 * (`tax_enabled` in migration 0023/0029), so in production the tax line stays
 * hidden until a Super Admin turns it on.
 */
export const SAMPLE_PLANS: PublicPlan[] = [
  {
    code: "free",
    name: "Free",
    description: "Paket gratis untuk mulai mencatat keuangan.",
    price_monthly_idr: 0,
    price_yearly_idr: 0,
    billing_period_days: 30,
    is_public: true,
    sort_order: 1,
    fallback_gateway: "manual",
    features: { ai_chat: true, bank_sync: false },
    limits: {
      ai_tokens_per_cycle: 5000,
      ai_requests_per_cycle: 50,
      max_transactions: 100,
      max_categories: 5,
      max_wallets: 1,
      ocr_scans_per_cycle: 0,
      voice_requests_per_day: 1,
    },
    limit_meta: {
      ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
      ai_requests_per_cycle: { key: "ai_requests_per_cycle", label: "Permintaan AI / siklus", unit: "permintaan", enforced: true },
      max_transactions: { key: "max_transactions", label: "Transaksi", unit: "transaksi", enforced: true },
      max_categories: { key: "max_categories", label: "Kategori", unit: "kategori", enforced: false },
      max_wallets: { key: "max_wallets", label: "Dompet", unit: "dompet", enforced: false },
      ocr_scans_per_cycle: { key: "ocr_scans_per_cycle", label: "Pindai struk / siklus", unit: "scan", enforced: true },
      voice_requests_per_day: { key: "voice_requests_per_day", label: "Permintaan suara / hari", unit: "permintaan", enforced: true },
    },
    tax_enabled: false,
  },
  {
    code: "pro",
    name: "Pro",
    description: "Untuk pengguna aktif dengan kebutuhan AI.",
    price_monthly_idr: 39000,
    price_yearly_idr: 390000,
    billing_period_days: 30,
    is_public: true,
    sort_order: 2,
    fallback_gateway: "midtrans",
    features: {
      ai_chat: true,
      ocr_receipt: true,
      voice_input: true,
      rag_search: true,
      reports_export: true,
      advanced_budgeting: true,
      api_access: false,
      bank_sync: false,
    },
    limits: {
      ai_tokens_per_cycle: 150000,
      ai_requests_per_cycle: 2000,
      max_transactions: 5000,
      max_categories: 30,
      max_wallets: 5,
      ocr_scans_per_cycle: 100,
      voice_requests_per_day: 50,
    },
    limit_meta: {
      ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
      ai_requests_per_cycle: { key: "ai_requests_per_cycle", label: "Permintaan AI / siklus", unit: "permintaan", enforced: true },
      max_transactions: { key: "max_transactions", label: "Transaksi", unit: "transaksi", enforced: true },
      max_categories: { key: "max_categories", label: "Kategori", unit: "kategori", enforced: false },
      max_wallets: { key: "max_wallets", label: "Dompet", unit: "dompet", enforced: false },
      ocr_scans_per_cycle: { key: "ocr_scans_per_cycle", label: "Pindai struk / siklus", unit: "scan", enforced: true },
      voice_requests_per_day: { key: "voice_requests_per_day", label: "Permintaan suara / hari", unit: "permintaan", enforced: true },
    },
    tax_enabled: true,
    tax_label: "PPN",
    tax_rate_percent: 11,
    tax_idr: 4290,
    total_idr: 43290,
  },
  {
    code: "business",
    name: "Business",
    description: "Untuk power user dan usaha kecil.",
    price_monthly_idr: 129000,
    price_yearly_idr: 1290000,
    billing_period_days: 30,
    is_public: true,
    sort_order: 3,
    fallback_gateway: "midtrans",
    features: {
      ai_chat: true,
      ocr_receipt: true,
      voice_input: true,
      rag_search: true,
      reports_export: true,
      advanced_budgeting: true,
      api_access: true,
      bank_sync: false,
    },
    limits: {
      ai_tokens_per_cycle: 1000000,
      ai_requests_per_cycle: -1,
      max_transactions: -1,
      max_categories: -1,
      max_wallets: -1,
      ocr_scans_per_cycle: -1,
      voice_requests_per_day: -1,
    },
    limit_meta: {
      ai_tokens_per_cycle: { key: "ai_tokens_per_cycle", label: "Token AI / siklus", unit: "token", enforced: true },
      ai_requests_per_cycle: { key: "ai_requests_per_cycle", label: "Permintaan AI / siklus", unit: "permintaan", enforced: true },
      max_transactions: { key: "max_transactions", label: "Transaksi", unit: "transaksi", enforced: true },
      max_categories: { key: "max_categories", label: "Kategori", unit: "kategori", enforced: false },
      max_wallets: { key: "max_wallets", label: "Dompet", unit: "dompet", enforced: false },
      ocr_scans_per_cycle: { key: "ocr_scans_per_cycle", label: "Pindai struk / siklus", unit: "scan", enforced: true },
      voice_requests_per_day: { key: "voice_requests_per_day", label: "Permintaan suara / hari", unit: "permintaan", enforced: true },
    },
    tax_enabled: true,
    tax_label: "PPN",
    tax_rate_percent: 11,
    tax_idr: 14190,
    total_idr: 143190,
  },
];
