import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

export interface SpkResult {
  score: number;
  status: "Sangat Baik" | "Baik" | "Cukup" | "Boros" | "Kritis";
  insights: string[];
}

function mapSpk(data: unknown): SpkResult {
  if (!data) {
    return { score: 0, status: "Kritis", insights: ["Belum ada data yang cukup untuk analisis."] };
  }
  const insights: string[] = [];
  const typedData = data as { score?: number; status?: string; breakdown?: { saving_ratio?: number; needs_vs_wants_ratio?: number; consistency?: number; expense_trend?: number; diversification?: number } };
  if (typedData.breakdown) {
    const b = typedData.breakdown;
    if (b.saving_ratio !== undefined) insights.push(`Rasio Tabungan: ${(b.saving_ratio * 100).toFixed(1)}%`);
    if (b.needs_vs_wants_ratio !== undefined) insights.push(`Rasio Kebutuhan vs Keinginan: ${(b.needs_vs_wants_ratio * 100).toFixed(1)}%`);
    if (b.consistency !== undefined) insights.push(`Konsistensi Catatan: ${(b.consistency * 100).toFixed(1)}%`);
    if (b.expense_trend !== undefined) insights.push(`Tren Pengeluaran: ${(b.expense_trend * 100).toFixed(1)}%`);
    if (b.diversification !== undefined) insights.push(`Diversifikasi: ${(b.diversification * 100).toFixed(1)}%`);
  }
  if (insights.length === 0) {
    if (typedData.score !== undefined && typedData.score > 0) {
      insights.push(`Skor keuangan Anda: ${typedData.score.toFixed(1)} (${typedData.status})`);
    } else {
      insights.push("Belum ada data yang cukup untuk analisis.");
    }
  }
  return {
    score: typedData.score || 0,
    status: (typedData.status as SpkResult["status"]) || "Kritis",
    insights,
  };
}

export function useSpkLatest() {
  return useQuery({
    queryKey: queryKeys.spk.latest,
    queryFn: async () => {
      const res = await api.get("/spk/latest");
      return mapSpk(res.data?.data);
    },
    staleTime: staleTimeConfig.spk,
  });
}

export function useCalculateSpk() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await api.post("/spk/calculate");
      return res.data?.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
    },
  });
}

export function useSpkHistory() {
  return useQuery({
    queryKey: queryKeys.spk.history,
    queryFn: async () => {
      const res = await api.get("/spk/history");
      return res.data?.data || [];
    },
    staleTime: staleTimeConfig.spk,
  });
}
