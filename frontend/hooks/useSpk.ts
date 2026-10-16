import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

export interface SpkResult {
  score: number;
  status: "Sangat Baik" | "Baik" | "Cukup" | "Boros" | "Kritis" | "Belum Ada Data";
  insights: string[];
  /** Weighted sub-score per indicator key (0..100), keyed by the model's key. */
  breakdown: Record<string, number>;
}

/** One point in the SPK score history. */
export interface SpkHistoryPoint {
  /** Short axis label, e.g. "14 Okt". */
  label: string;
  score: number;
  status: string;
  /** ISO timestamp of the calculation. */
  at: string;
}

/** Raw SPK payload as returned by the backend. */
interface RawSpk {
  score?: number;
  status?: string;
  breakdown?: Record<string, number>;
  calculated_at?: string;
}

export function mapSpk(data: unknown): SpkResult {
  if (!data) {
    return {
      score: 0,
      status: "Belum Ada Data",
      insights: ["Belum ada data yang cukup untuk analisis."],
      breakdown: {},
    };
  }
  const raw = data as RawSpk;
  const breakdown = raw.breakdown ?? {};

  // Human-readable insights for the dashboard widget. Labels are derived from the
  // indicator key; a custom model's key may not be listed, so we fall back to a
  // title-cased key rather than dropping it.
  const insights: string[] = [];
  for (const [key, value] of Object.entries(breakdown)) {
    if (typeof value !== "number") continue;
    insights.push(`${labelForKey(key)}: ${value.toFixed(1)}`);
  }
  if (insights.length === 0) {
    if (raw.score && raw.score > 0) {
      insights.push(`Skor keuangan Anda: ${raw.score.toFixed(1)} (${raw.status})`);
    } else {
      insights.push("Belum ada data yang cukup untuk analisis.");
    }
  }

  return {
    score: raw.score ?? 0,
    status: (raw.status as SpkResult["status"]) ?? "Belum Ada Data",
    insights,
    breakdown,
  };
}

/** Known indicator labels; unknown keys are title-cased as a fallback. */
const KNOWN_LABELS: Record<string, string> = {
  saving_ratio: "Rasio Tabungan",
  needs_vs_wants: "Kebutuhan vs Keinginan",
  needs_vs_wants_ratio: "Kebutuhan vs Keinginan",
  consistency: "Konsistensi Catatan",
  expense_trend: "Tren Pengeluaran",
  diversification: "Diversifikasi",
};

function labelForKey(key: string): string {
  return KNOWN_LABELS[key] ?? key.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

export function useSpkLatest() {
  return useQuery({
    queryKey: queryKeys.spk.latest,
    queryFn: async () => {
      const res = await api.get("/spk/latest");
      return mapSpk(res.data?.data);
    },
    staleTime: staleTimeConfig.spk,
    placeholderData: (prev) => prev,
  });
}

export function useCalculateSpk() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await api.post("/spk/calculate");
      return mapSpk(res.data?.data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
    },
  });
}

/** Maps the raw history payload into chart points (oldest first). */
export function mapSpkHistory(data: unknown): SpkHistoryPoint[] {
  if (!Array.isArray(data)) return [];
  const points = data
    .map((row) => {
      const r = row as RawSpk;
      if (typeof r.score !== "number") return null;
      const at = r.calculated_at ?? "";
      const d = at ? new Date(at) : null;
      const label = d && !Number.isNaN(d.getTime())
        ? d.toLocaleDateString("id-ID", { day: "numeric", month: "short" })
        : "";
      return { label, score: r.score, status: r.status ?? "", at };
    })
    .filter((p): p is SpkHistoryPoint => p !== null);
  // The API returns newest-first; the chart wants oldest-first.
  return points.reverse();
}

export function useSpkHistory(limit = 12) {
  return useQuery({
    queryKey: queryKeys.spk.history,
    queryFn: async () => {
      const res = await api.get(`/spk/history?limit=${limit}`);
      return mapSpkHistory(res.data?.data);
    },
    staleTime: staleTimeConfig.spk,
    placeholderData: (prev) => prev,
  });
}
