"use client";

import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";

/** Where an effective value came from: a DB override, the env default, or nothing. */
export type ConfigSource = "db" | "env" | "none";

/** Effective runtime configuration reported by the backend (secrets masked). */
export interface ConfigStatus {
  /** Non-secret effective values. */
  effective: Record<string, string | number | boolean>;
  /** Masked view of each secret (e.g. "SB-M••••1234"). */
  secrets: Record<string, string>;
  /** Whether each secret is set at all. */
  has_secret: Record<string, boolean>;
  /** Precedence source per secret. */
  source: Record<string, ConfigSource>;
}

export const configStatusKey = ["admin", "config-status"] as const;

export function useConfigStatus() {
  return useQuery({
    queryKey: configStatusKey,
    queryFn: async (): Promise<ConfigStatus> => {
      const res = await api.get("/admin/settings/config");
      return res.data?.data as ConfigStatus;
    },
    staleTime: 30_000,
  });
}
