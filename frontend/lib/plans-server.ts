import type { PublicPlan } from "@/lib/plans-types";

/**
 * SERVER-ONLY. Fetches the public plan catalog for Server Components (the
 * `/pricing` page and the landing pricing teaser).
 *
 * Do NOT import this from a Client Component or a `"use client"` module: it
 * reads `process.env.BACKEND_ORIGIN` and talks to the backend directly. (The
 * `server-only` guard package is not a dependency of this project, so this
 * header comment is the contract instead.)
 *
 * The catalog endpoint (`GET /api/v1/public/plans`) is public by design and
 * returns only `is_public && is_active` plans, so caching its response with ISR
 * leaks nothing user-specific.
 */

const BACKEND_ORIGIN = process.env.BACKEND_ORIGIN ?? "http://localhost:8081";

/** Cache window for the catalog. Pricing changes by a Super Admin appear within
 * this window; checkout always reads the authoritative DB price. */
const REVALIDATE_SECONDS = 3600;

/** Network timeout so a hung backend cannot stall the page render. */
const FETCH_TIMEOUT_MS = 5000;

/**
 * Fetch the public plan catalog.
 *
 * Returns `[]` on any failure (non-OK response, malformed body, or network
 * error) so callers can render a graceful fallback instead of crashing the
 * route.
 */
export async function fetchPublicPlans(): Promise<PublicPlan[]> {
  try {
    const res = await fetch(`${BACKEND_ORIGIN}/api/v1/public/plans`, {
      next: { revalidate: REVALIDATE_SECONDS },
      signal: AbortSignal.timeout(FETCH_TIMEOUT_MS),
    });
    if (!res.ok) return [];
    const body = (await res.json()) as { data?: PublicPlan[] };
    return Array.isArray(body.data) ? body.data : [];
  } catch {
    return [];
  }
}

/**
 * Plan limits the UI may advertise: only those the backend actually enforces.
 *
 * A limit absent from `limit_meta` is treated as enforced (backwards
 * compatible). `max_categories` and `max_wallets` are marked `enforced: false`
 * by the backend and are therefore never shown.
 */
export function enforcedLimits(plan: PublicPlan): Array<[string, number]> {
  return Object.entries(plan.limits).filter(
    ([key]) => plan.limit_meta?.[key]?.enforced !== false,
  );
}
