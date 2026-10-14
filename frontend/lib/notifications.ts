import type { DomainEvent } from "@/lib/realtime-dispatcher";

/**
 * A user-facing notification derived from a realtime domain event.
 */
export interface AppNotification {
  /** Unique id (the source event id when present, else a generated one). */
  id: string;
  /** Event name that produced it (kept for de-duplication & routing). */
  event: string;
  title: string;
  body: string;
  /** Unix ms the notification was created. */
  createdAt: number;
  /** Where clicking the notification should navigate. */
  href: string;
  /** Whether the user has seen it. */
  read: boolean;
}

/** Shape of the payloads we read fields from (all optional, defensive). */
interface EventPayload {
  amount_idr?: number;
  invoice_number?: string;
  status?: string;
  current_plan?: string;
  previous_plan?: string;
  plan_wide?: boolean;
  allocated_tokens?: number;
  endpoint_id?: string;
  user_email?: string;
  payment_id?: string;
  // ai_route_quarantined
  use_case?: string;
  model?: string;
  endpoint?: string;
  reason?: string;
  recovered?: boolean;
}

/** formatIDR renders an integer Rupiah amount for a notification body. */
function formatIDR(v: number | undefined): string {
  if (typeof v !== "number" || !Number.isFinite(v)) return "—";
  return `Rp ${v.toLocaleString("id-ID")}`;
}

/** readPayload narrows the unknown payload to the fields we use. */
function readPayload(payload: unknown): EventPayload {
  if (typeof payload !== "object" || payload === null) return {};
  return payload as EventPayload;
}

/**
 * notificationFor maps a realtime domain event to a notification, or null when
 * the event is not user-notifiable (e.g. an admin-only gateway event reaching a
 * plain user). `now` is injectable so tests are deterministic.
 *
 * Only events that concern the signed-in user become notifications; the
 * super-admin stream already carries its own operational events.
 */
export function notificationFor(event: DomainEvent, now: number = Date.now()): AppNotification | null {
  const payload = readPayload(event.payload);
  const base = { id: event.id || `${event.event}-${now}`, event: event.event, createdAt: now, read: false };

  switch (event.event) {
    case "payment_received":
      return {
        ...base,
        title: "Pembayaran diterima",
        body: `Pembayaran ${formatIDR(payload.amount_idr)}${payload.invoice_number ? ` untuk ${payload.invoice_number}` : ""} telah kami terima.`,
        href: "/user/settings/subscription?tab=payments",
      };

    case "plan_updated":
      return {
        ...base,
        title: "Paket diperbarui",
        body: payload.current_plan
          ? `Paket Anda kini ${payload.current_plan}.`
          : "Paket langganan Anda telah diperbarui.",
        href: "/user/settings/subscription",
      };

    case "quota_reset":
      return {
        ...base,
        title: "Kuota direset",
        body: payload.allocated_tokens
          ? `Siklus baru dimulai. Kuota ${payload.allocated_tokens.toLocaleString("id-ID")} token tersedia.`
          : "Siklus kuota baru telah dimulai.",
        href: "/user/settings/subscription",
      };

    case "manual_approval_needed":
      // Only super admins care; a plain user's stream never receives it, but the
      // mapping stays here so the admin UI can reuse the same store.
      return {
        ...base,
        title: "Pembayaran menunggu verifikasi",
        body: `${payload.user_email ?? "Pengguna"} mengirim ${formatIDR(payload.amount_idr)} untuk diverifikasi.`,
        href: "/super-admin/payments/review",
      };

    case "ai_endpoint_down":
      return {
        ...base,
        title: "Endpoint AI terganggu",
        body: `Endpoint ${payload.endpoint_id ?? ""} tidak sehat; trafik dialihkan otomatis.`.trim(),
        href: "/super-admin/ai-gateway",
      };

    case "ai_route_quarantined": {
      // Model-level failover: one route was quarantined or recovered.
      const where = [payload.use_case, payload.model].filter(Boolean).join(" / ");
      if (payload.recovered) {
        return {
          ...base,
          title: "Rute AI pulih",
          body: `Model ${payload.model ?? "?"} kembali normal dan siap melayani.`,
          href: "/super-admin/ai-gateway",
        };
      }
      return {
        ...base,
        title: "Model AI dikarantina",
        body: `${where || "Sebuah rute"} gagal berturut-turut dan dialihkan otomatis.${payload.reason ? ` (${payload.reason})` : ""}`,
        href: "/super-admin/ai-gateway",
      };
    }

    default:
      return null;
  }
}

/** The most notifications to retain in memory (older ones are dropped). */
export const MAX_NOTIFICATIONS = 20;

/**
 * appendNotification adds `n` to the front of `list`, de-duplicating by id and
 * capping the length. Pure so it can be unit-tested without React.
 */
export function appendNotification(list: AppNotification[], n: AppNotification): AppNotification[] {
  if (list.some((x) => x.id === n.id)) return list;
  return [n, ...list].slice(0, MAX_NOTIFICATIONS);
}

/** unreadCount returns how many notifications are unread. */
export function unreadCount(list: AppNotification[]): number {
  return list.reduce((acc, n) => (n.read ? acc : acc + 1), 0);
}
