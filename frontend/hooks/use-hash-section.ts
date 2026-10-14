"use client";

import { useSyncExternalStore } from "react";

function subscribe(onStoreChange: () => void) {
  window.addEventListener("hashchange", onStoreChange);
  return () => window.removeEventListener("hashchange", onStoreChange);
}

function getSnapshot(): string {
  return window.location.hash.replace("#", "");
}

// Server snapshot: no hash, so the markup is deterministic during hydration.
function getServerSnapshot(): string {
  return "";
}

/**
 * Returns the current URL hash when it names one of `validIds`, else null.
 *
 * Backed by useSyncExternalStore so a deep link such as /user/profile#privacy
 * opens the matching accordion section without writing state from an effect
 * (which would trigger a cascading render).
 */
export function useHashSection(validIds: readonly string[]): string | null {
  const hash = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  return validIds.includes(hash) ? hash : null;
}
