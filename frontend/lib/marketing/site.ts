/**
 * Single source of truth for site-level identity and contact channels.
 *
 * Everything user-facing that refers to the brand, the canonical URL, or a
 * support channel reads from here so no page drifts (spec doc 05 §2.3).
 */

const FALLBACK_URL = "http://localhost:3001";

export interface SiteConfig {
  name: string;
  /** Short brand line used in the footer. */
  tagline: string;
  /** Absolute canonical origin, without a trailing slash. */
  url: string;
  email: string;
  /** Human-readable phone number shown on screen. */
  whatsappDisplay: string;
  /** Digits only (country code first) for the wa.me link. */
  whatsappNumber: string;
  /** Realistic support window (owner-confirmed before release). */
  supportHours: string;
  /** Expected response time, stated plainly. */
  responseTime: string;
}

export const site: SiteConfig = {
  name: "FinTrack",
  tagline: "Catat. Pahami. Kelola.",
  url: (process.env.NEXT_PUBLIC_SITE_URL ?? FALLBACK_URL).replace(/\/+$/, ""),
  // PLACEHOLDER — confirm the real addresses before release (spec doc 10 §1).
  email: "halo@fintrack.id",
  whatsappDisplay: "+62 812-0000-0000",
  whatsappNumber: "6281200000000",
  supportHours: "Senin–Jumat, 09.00–17.00 WIB",
  responseTime: "Kami membalas pada hari kerja, umumnya dalam 1×24 jam.",
};

/** Build a `mailto:` link with a pre-filled, URL-encoded subject. */
export function mailtoHref(subject = `Pertanyaan tentang ${site.name}`): string {
  return `mailto:${site.email}?subject=${encodeURIComponent(subject)}`;
}

/** Build a `wa.me` link with an optional pre-filled message. */
export function whatsappHref(message = `Halo ${site.name}, saya ingin bertanya.`): string {
  return `https://wa.me/${site.whatsappNumber}?text=${encodeURIComponent(message)}`;
}
