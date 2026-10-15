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
 * Returns the raw URL hash (without the leading `#`), or "" when absent.
 *
 * Backed by useSyncExternalStore so a component re-renders when the hash
 * changes (deep link, in-page navigation) without writing state from an effect.
 * Use `useHashSection` instead when you only care about a known set of ids.
 */
export function useHash(): string {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
