"use client";

import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";

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
  version: number;
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
    staleTime: 1000 * 60 * 2,
    retry: false,
  });

  const data = query.data ?? null;

  return {
    entitlements: data,
    isLoading: query.isLoading,
    isError: query.isError,
    planCode: data?.plan_code ?? "",
    planName: data?.plan_name ?? "",
    status: data?.status ?? "",
    hasFeature: (featureKey: string) => hasFeature(data, featureKey),
    withinLimit: (limitKey: string, current: number) => withinLimit(data, limitKey, current),
    limitValue: (limitKey: string) => limitValue(data, limitKey),
  };
}
