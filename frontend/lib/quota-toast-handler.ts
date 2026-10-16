import { toast } from "sonner"
import { formatQuotaRemaining } from "@/components/billing/QuotaCountdown"

const shownKeys: Record<string, number> = {}

export function handleQuotaToast(event: string, payload: any) {
  if (!payload) return

  const key = `${event}:${payload.limit_key || ""}:${payload.cycle_type || ""}`
  const now = Date.now()
  if (shownKeys[key] && now - shownKeys[key] < 30_000) {
    return // deduplicate within 30 seconds
  }
  shownKeys[key] = now

  switch (event) {
    case "quota_degraded": {
      const resetMsg = payload.weekly_reset_at
        ? ` Reset: ${formatQuotaRemaining(Date.parse(payload.weekly_reset_at) - now)}.`
        : ""
      toast.warning(payload.message || "Kuota mingguan habis.", {
        description: `Mode degraded aktif.${resetMsg}`,
        duration: 8000,
      })
      break
    }

    case "quota_blocked": {
      const resetMsg = payload.monthly_reset_at
        ? ` Reset: ${formatQuotaRemaining(Date.parse(payload.monthly_reset_at) - now)}.`
        : ""
      toast.error(payload.message || "Batas kuota tercapai.", {
        description: `Fitur diblokir sementara.${resetMsg}`,
        duration: 10000,
      })
      break
    }

    case "quota_reset": {
      const type = payload.cycle_type === "weekly" ? "mingguan" : "bulanan"
      toast.success(`Kuota ${type} Anda telah direset.`, {
        description: "Selamat melanjutkan aktivitas keuangan Anda!",
        duration: 5000,
      })
      break
    }
  }
}
