import { toast } from "sonner";

/**
 * Shared handling for the backend's plan-gate responses.
 *
 * The API answers a blocked request with 403 `feature_not_in_plan` or 402
 * `limit_exceeded` / `limit_not_in_plan`, both carrying an `upgrade_url`. This
 * module turns that shape into a consistent toast with a link to the
 * subscription page, so every gated action explains itself the same way.
 *
 * The UI is a mirror, not the gate: callers should treat a `true` return as
 * "the backend refused this request on plan grounds" and simply stop, not as
 * the only place access is enforced.
 */

export const UPGRADE_URL = "/user/subscription";

const FEATURE_MESSAGES: Record<string, string> = {
  ai_chat: "Asisten AI tidak tersedia pada paket Anda.",
  ocr_receipt: "Pindai struk (OCR) membutuhkan paket Pro atau lebih tinggi.",
  voice_input: "Input suara membutuhkan paket Pro atau lebih tinggi.",
  rag_search: "Pencarian cerdas membutuhkan paket Pro atau lebih tinggi.",
  reports_export: "Ekspor laporan membutuhkan paket Pro atau lebih tinggi.",
  advanced_budgeting: "Anggaran lanjutan membutuhkan paket Pro atau lebih tinggi.",
  api_access: "Akses API membutuhkan paket Business.",
  bank_sync: "Sinkronisasi bank belum tersedia pada paket Anda.",
};

interface ApiErrorBody {
  error?: string;
  feature_key?: string;
  limit_key?: string;
  /** Present on a limit gate: the plan limit and the usage that hit it. */
  limit_value?: number;
  current_usage?: number;
  upgrade_url?: string;
}

interface AxiosLikeError {
  response?: {
    status?: number;
    data?: ApiErrorBody;
  };
}

/**
 * QUOTA_ERRORS are the ledger/quota errors that also warrant an upgrade CTA.
 * They are returned by the token ledger (HTTP 429 `insufficient_quota`), which
 * is a different code path from the plan-gate middleware (403/402).
 */
const QUOTA_ERRORS: Record<string, true> = {
  insufficient_quota: true,
  quota_exceeded: true,
};

/** isEntitlementError reports whether an error came from a plan gate. */
export function isEntitlementError(err: unknown): boolean {
  const body = (err as AxiosLikeError)?.response?.data;
  if (!body?.error) return false;
  return (
    body.error === "feature_not_in_plan" ||
    body.error === "limit_exceeded" ||
    body.error === "limit_not_in_plan" ||
    QUOTA_ERRORS[body.error] === true
  );
}

/** Human labels for the limit keys the backend reports on a gate response. */
const LIMIT_MESSAGES: Record<string, string> = {
  ai_tokens_per_cycle: "Kuota AI Anda untuk siklus ini sudah habis.",
  ai_requests_per_cycle: "Jumlah permintaan AI Anda untuk siklus ini sudah habis.",
  max_transactions: "Jumlah transaksi Anda sudah mencapai batas paket.",
  ocr_scans_per_cycle: "Kuota pindai struk (OCR) Anda untuk siklus ini sudah habis.",
};

/**
 * notifyEntitlementError shows an upgrade toast when `err` is a plan or quota
 * gate. Returns true when it handled the error, so callers can skip their
 * generic failure message.
 */
export function notifyEntitlementError(err: unknown): boolean {
  const response = (err as AxiosLikeError)?.response;
  const body = response?.data;
  if (!body?.error || !isEntitlementError(err)) return false;

  const url = body.upgrade_url || UPGRADE_URL;
  let message: string;
  let description = "Lihat paket Anda untuk meningkatkan batas.";

  if (body.error === "feature_not_in_plan") {
    message = FEATURE_MESSAGES[body.feature_key ?? ""] ?? "Fitur ini tidak tersedia pada paket Anda.";
  } else if (QUOTA_ERRORS[body.error] === true) {
    message = "Sisa kuota Anda tidak mencukupi untuk permintaan ini.";
  } else {
    // A limit gate: name the exact limit and, when the backend reports it, how
    // much has been used so the user understands *why* it stopped.
    message = LIMIT_MESSAGES[body.limit_key ?? ""] ?? "Anda telah mencapai batas paket Anda untuk periode ini.";
    if (typeof body.limit_value === "number" && typeof body.current_usage === "number") {
      description = `Terpakai ${body.current_usage.toLocaleString("id-ID")} dari ${body.limit_value.toLocaleString("id-ID")}. Lihat paket untuk meningkatkan batas.`;
    }
  }

  toast.error(message, {
    description,
    action: {
      label: "Lihat paket",
      onClick: () => {
        if (typeof window !== "undefined") window.location.href = url;
      },
    },
  });
  return true;
}
