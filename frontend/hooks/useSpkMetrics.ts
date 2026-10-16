"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

/** How a raw indicator value is derived. Mirrors the Go domain type. */
export type SpkMetricType =
  | "saving_ratio"
  | "needs_ratio"
  | "consistency"
  | "expense_trend"
  | "diversification";

export interface SpkMetricIndicator {
  key: string;
  label: string;
  weight: number;
  target: number;
  type: SpkMetricType;
  categories?: string[];
  direction?: "higher_better" | "lower_better";
}

export interface SpkStatusThreshold {
  min: number;
  label: string;
}

export interface SpkMetricModel {
  indicators: SpkMetricIndicator[];
  status_thresholds: SpkStatusThreshold[];
  source?: string;
  name?: string;
}

export interface SpkMetricResponse {
  model: SpkMetricModel;
  is_override: boolean;
}

/** The wire payload for saving a model. */
export interface SpkMetricPayload {
  name?: string;
  description?: string;
  indicators: SpkMetricIndicator[];
  status_thresholds: SpkStatusThreshold[];
}

/** Labels for the metric types, used by the editor's type picker. */
export const SPK_METRIC_TYPE_LABELS: Record<SpkMetricType, string> = {
  saving_ratio: "Rasio Tabungan",
  needs_ratio: "Kebutuhan vs Keinginan",
  consistency: "Konsistensi Catatan",
  expense_trend: "Tren Pengeluaran",
  diversification: "Diversifikasi",
};

/**
 * useMySpkModel fetches the model that applies to the signed-in user, plus
 * whether it is their own override (vs the super-admin default).
 */
export function useMySpkModel() {
  return useQuery({
    queryKey: queryKeys.spkMetrics.mine,
    queryFn: async (): Promise<SpkMetricResponse> => {
      const res = await api.get("/spk/metrics");
      return res.data?.data as SpkMetricResponse;
    },
    staleTime: staleTimeConfig.spk,
    placeholderData: (prev) => prev,
  });
}

/** useSaveMySpkModel saves the user's personal override. */
export function useSaveMySpkModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: SpkMetricPayload) => {
      const res = await api.put("/spk/metrics", payload);
      return res.data?.data as SpkMetricResponse;
    },
    onSuccess: () => {
      // The override changes how the next calculation scores, so refresh the
      // SPK results too.
      queryClient.invalidateQueries({ queryKey: queryKeys.spkMetrics.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
    },
  });
}

/** useResetMySpkModel reverts the user to the super-admin default. */
export function useResetMySpkModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete("/spk/metrics");
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spkMetrics.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
    },
  });
}

/** useSpkDefaultModel fetches the super-admin default (super-admin only). */
export function useSpkDefaultModel() {
  return useQuery({
    queryKey: queryKeys.spkMetrics.default,
    queryFn: async (): Promise<SpkMetricModel> => {
      const res = await api.get("/admin/spk-metrics");
      return (res.data?.data as { model: SpkMetricModel }).model;
    },
    staleTime: staleTimeConfig.admin,
  });
}

/** useSaveSpkDefault replaces the super-admin default model. */
export function useSaveSpkDefault() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: SpkMetricPayload) => {
      const res = await api.put("/admin/spk-metrics", payload);
      return (res.data?.data as { model: SpkMetricModel }).model;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spkMetrics.all });
    },
  });
}
