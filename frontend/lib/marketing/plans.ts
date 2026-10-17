import type { PublicPlan } from "@/lib/plans-types";

/**
 * Display helpers for the pricing surface (spec doc 04 §A). Pure functions so
 * the landing teaser, the pricing cards, and the comparison tables all read the
 * live catalog through the same rules — no local price/limit dictionary drifts
 * from the backend.
 *
 * Honesty rules enforced here:
 *  - A limit the backend does not enforce (`limit_meta[key].enforced === false`)
 *    is never turned into a row.
 *  - `-1` (the backend "unbounded" sentinel) always renders as "Tanpa batas".
 *  - Limit labels/units come from `limit_meta`, never from a local table.
 */

/** A single row of the feature comparison table (spec doc 04 §A.5). */
export interface PlanFeatureRow {
  /** Backend entitlement key (`features[key]`). */
  key: string;
  /** Official catalog label (spec doc 13 §3). */
  label: string;
  /** Deep link to the matching feature page, when one exists. */
  href?: string;
}

/**
 * Feature rows in display order. Mirrors the backend entitlement catalog
 * (`backend/internal/entitlement/catalog.go`) and doc 04 §A.5. `bank_sync` is
 * deliberately absent — the catalog marks it "belum tersedia".
 */
export const PLAN_FEATURE_ROWS: PlanFeatureRow[] = [
  { key: "ai_chat", label: "Asisten AI (chat)", href: "/features/ai-chat" },
  { key: "ocr_receipt", label: "Pindai struk (OCR)", href: "/features/receipt-scan" },
  { key: "voice_input", label: "Input suara", href: "/features/voice-input" },
  { key: "rag_search", label: "Pencarian cerdas (RAG)" },
  { key: "advanced_budgeting", label: "Anggaran lanjutan" },
  { key: "reports_export", label: "Ekspor laporan", href: "/features/reports" },
  { key: "api_access", label: "Akses API" },
];

/** A curated pricing-card bullet. When `key` is set the bullet only shows if the
 * plan actually enables that feature, so a card can never over-promise. */
export interface PlanHighlight {
  key?: string;
  label: string;
}

/** Card highlights per plan code (spec doc 04 §A.4), max four bullets. */
export const PLAN_HIGHLIGHTS: Record<string, PlanHighlight[]> = {
  free: [
    { key: "ai_chat", label: "Asisten AI (chat)" },
    { label: "Catat lewat chat seperti mengirim pesan" },
  ],
  pro: [
    { key: "ocr_receipt", label: "Pindai struk (OCR)" },
    { key: "voice_input", label: "Input suara" },
    { key: "rag_search", label: "Pencarian cerdas (RAG)" },
    { key: "advanced_budgeting", label: "Anggaran lanjutan" },
  ],
  business: [
    { key: "ocr_receipt", label: "Semua fitur Pro" },
    { key: "api_access", label: "Akses API" },
    { key: "rag_search", label: "Pencarian cerdas (RAG)" },
    { key: "reports_export", label: "Ekspor laporan" },
  ],
};

/** Maximum bullets rendered on a pricing card. */
export const MAX_CARD_HIGHLIGHTS = 4;

/**
 * Short feature bullets for a pricing card. Uses the curated list when the plan
 * code is known (better differentiators), else falls back to the top enabled
 * catalog features. Every keyed bullet is gated by `plan.features`.
 */
export function planHighlights(plan: PublicPlan): string[] {
  const curated = PLAN_HIGHLIGHTS[plan.code?.toLowerCase() ?? ""];
  if (curated) {
    return curated
      .filter((h) => !h.key || plan.features?.[h.key] === true)
      .map((h) => h.label)
      .slice(0, MAX_CARD_HIGHLIGHTS);
  }
  return PLAN_FEATURE_ROWS.filter((row) => plan.features?.[row.key] === true)
    .map((row) => row.label)
    .slice(0, MAX_CARD_HIGHLIGHTS);
}

/** Whether a plan enables a catalog feature. */
export function planHasFeature(plan: PublicPlan, key: string): boolean {
  return plan.features?.[key] === true;
}

/** Stable order for the limit comparison table (spec doc 04 §A.6). */
export const LIMIT_ROW_ORDER = [
  "ai_tokens_per_cycle",
  "ai_requests_per_cycle",
  "max_transactions",
  "ocr_scans_per_cycle",
  "voice_requests_per_day",
];

export interface LimitRow {
  key: string;
  /** Label from `limit_meta`, never a local dictionary. */
  label: string;
  unit: string;
  /** One display value per plan, aligned with the plan order. */
  values: Array<{ code: string; text: string }>;
}

/**
 * Render a limit value for display. `-1` (and any negative sentinel) means the
 * backend applies no cap and always reads "Tanpa batas"; a missing value reads
 * as an em dash rather than a misleading zero.
 */
export function formatLimitValue(value: number | undefined | null, unit: string): string {
  if (value == null) return "—";
  if (value < 0) return "Tanpa batas";
  const n = value.toLocaleString("id-ID");
  return unit ? `${n} ${unit}` : n;
}

/**
 * Build the limit comparison rows from the live catalog. Only limits the
 * backend enforces (per `limit_meta[key].enforced !== false`) appear, and only
 * when at least one plan defines the key. Rows follow `LIMIT_ROW_ORDER`, then
 * any unknown enforced key alphabetically so nothing silently disappears.
 */
export function buildLimitRows(plans: PublicPlan[]): LimitRow[] {
  const keys = new Set<string>();
  for (const plan of plans) {
    for (const [key, meta] of Object.entries(plan.limit_meta ?? {})) {
      if (meta.enforced !== false && plan.limits?.[key] != null) keys.add(key);
    }
  }

  const ordered = [
    ...LIMIT_ROW_ORDER.filter((key) => keys.has(key)),
    ...[...keys].filter((key) => !LIMIT_ROW_ORDER.includes(key)).sort(),
  ];

  return ordered.map((key) => {
    const source = plans.find((plan) => plan.limit_meta?.[key] != null);
    const meta = source?.limit_meta?.[key];
    return {
      key,
      label: meta?.label ?? key,
      unit: meta?.unit ?? "",
      values: plans.map((plan) => ({
        code: plan.code,
        text: formatLimitValue(plan.limits?.[key], meta?.unit ?? ""),
      })),
    };
  });
}

/** Order plans for display following `sort_order` (Free → Pro → Business). */
export function orderPlans(plans: PublicPlan[]): PublicPlan[] {
  return [...plans].sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0));
}

/** A plan code that identifies the free tier (never charged). */
export function isFreePlan(plan: PublicPlan): boolean {
  return plan.code?.toLowerCase() === "free" || plan.price_monthly_idr === 0;
}

/** Effective monthly price when paying yearly (for honesty on the toggle). */
export function yearlyMonthlyEquivalent(plan: PublicPlan): number {
  return Math.round((plan.price_yearly_idr ?? 0) / 12);
}
