"use client";

import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

/**
 * Effective entitlements for the signed-in user, as returned by
 * `GET /api/v1/me/entitlements`.
 *
 * This is a UX mirror only. The backend middleware is the authoritative gate —
 * the UI uses these values to hide/disable actions and show upgrade hints, never
 * to decide access. A user who bypasses the UI still receives 403/402 from the
 * API.
 */
export interface Entitlements {
  user_id: string;
  plan_code: string;
  plan_name: string;
  status: string;
  features: Record<string, boolean>;
  limits: Record<string, number>;
  /** Real-time consumption per limit key (F6). Absent on older backends. */
  usage?: Record<string, UsageEntry>;
  /** Active quota cycle (F6), when the backend tracks one. */
  period?: Period | null;
  version: number;
}
export interface UseEntitlementsResult {
  entitlements: Entitlements | null;
  isLoading: boolean;
  isError: boolean;
  planCode: string;
  planName: string;
  status: string;
  usage: Record<string, UsageEntry>;
  period: Period | null;
  hasFeature: (featureKey: string) => boolean;
  withinLimit: (limitKey: string, current: number) => boolean;
  limitValue: (limitKey: string) => number | undefined;
}

/** UsageEntry is the consumption for one limit key. */
export interface UsageEntry {
  used: number;
  limit: number;
  /** -1 when the limit is unlimited. */
  remaining: number;
  unlimited: boolean;
  /**
   * Display unit for this limit (e.g. "kuota", "permintaan"). Supplied by the
   * backend so the UI never hardcodes a unit that could change.
   */
  unit?: string;
}

/** Period is the active quota cycle bounds (RFC3339). */
export interface Period {
  start: string;
  end: string;
}

/** An empty entitlement set, used while loading and on failure (fail-closed). */
export const EMPTY_ENTITLEMENTS: Entitlements = {
  user_id: "",
  plan_code: "",
  plan_name: "",
  status: "",
  features: {},
  limits: {},
  version: 0,
};

/** hasFeature reports whether a plan feature is enabled. */
export function hasFeature(ent: Entitlements | null, featureKey: string): boolean {
  if (!ent || !ent.features) return false;
  return ent.features[featureKey] === true;
}

/**
 * withinLimit reports whether `current` usage is within the plan limit for
 * `limitKey`. Unlimited (-1) always passes. An absent key fails closed, matching
 * the backend's `limit_not_in_plan` behaviour.
 */
export function withinLimit(ent: Entitlements | null, limitKey: string, current: number): boolean {
  if (!ent || !ent.limits) return false;
  const limit = ent.limits[limitKey];
  if (limit === undefined) return false;
  if (limit < 0) return true;
  return current < limit;
}

/** limitValue returns the raw limit (-1 = unlimited, undefined = absent). */
export function limitValue(ent: Entitlements | null, limitKey: string): number | undefined {
  if (!ent || !ent.limits) return undefined;
  return ent.limits[limitKey];
}

async function fetchEntitlements(): Promise<Entitlements> {
  const res = await api.get("/me/entitlements");
  return (res.data?.data ?? EMPTY_ENTITLEMENTS) as Entitlements;
}

export function useEntitlements() {
  const query = useQuery({
    queryKey: queryKeys.entitlements.me,
    queryFn: fetchEntitlements,
    staleTime: staleTimeConfig.entitlements,
    retry: false,
    placeholderData: (previousData) => previousData,
  });
  const data = query.data ?? null;

  return {
    entitlements: data,
    isLoading: query.isLoading,
    isError: query.isError,
    planCode: data?.plan_code ?? "",
    planName: data?.plan_name ?? "",
    status: data?.status ?? "",
    usage: data?.usage ?? {},
    period: data?.period ?? null,
    hasFeature: (featureKey: string) => hasFeature(data, featureKey),
    withinLimit: (limitKey: string, current: number) => withinLimit(data, limitKey, current),
    limitValue: (limitKey: string) => limitValue(data, limitKey),
  };
}
