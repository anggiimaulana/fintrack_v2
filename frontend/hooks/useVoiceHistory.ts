"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { recordBrowserHistory, type VoiceHistoryEntry } from "@/lib/voice-history";
import { toast } from "sonner";

export interface VoiceHistoryPage {
  data: VoiceHistoryEntry[];
  total: number;
  page: number;
  perPage: number;
}

export interface VoiceHistoryFilter {
  page?: number;
  perPage?: number;
  /** When false, the query is not executed (e.g. the panel is closed). */
  enabled?: boolean;
}

function historyParams(filter: VoiceHistoryFilter): string {
  const p = new URLSearchParams();
  p.set("page", String(filter.page ?? 1));
  p.set("per_page", String(filter.perPage ?? 20));
  return p.toString();
}

/** useVoiceHistory lists the user's transcription history (Q-04). */
export function useVoiceHistory(filter: VoiceHistoryFilter = {}) {
  const key = queryKeys.voice.history(historyParams(filter));
  return useQuery<VoiceHistoryPage>({
    queryKey: key,
    queryFn: async () => {
      const res = await api.get(`/me/voice-history?${historyParams(filter)}`);
      return {
        data: (res.data?.data ?? []) as VoiceHistoryEntry[],
        total: res.data?.meta?.total ?? 0,
        page: res.data?.meta?.page ?? 1,
        perPage: res.data?.meta?.per_page ?? 20,
      };
    },
    // Deferred until the panel is opened, so a closed dialog issues no request.
    enabled: filter.enabled ?? true,
    staleTime: 1000 * 60,
    placeholderData: (prev) => prev,
  });
}

/** useRecordBrowserHistory persists an on-device transcription. */
export function useRecordBrowserHistory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: recordBrowserHistory,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.voice.historyAll });
      queryClient.invalidateQueries({ queryKey: queryKeys.voice.config });
      queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.me });
    },
    // Best-effort: a failed history write must not surface an error toast — the
    // transcription itself already succeeded on-device.
    onError: () => {},
  });
}

/** useDeleteVoiceHistory removes one history row. */
export function useDeleteVoiceHistory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/me/voice-history/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.voice.historyAll }),
    onError: () => toast.error("Gagal menghapus riwayat suara."),
  });
}

/** useClearVoiceHistory removes every history row. */
export function useClearVoiceHistory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete("/me/voice-history");
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.voice.historyAll }),
    onError: () => toast.error("Gagal membersihkan riwayat suara."),
  });
}
