"use client";

import { useCallback, useSyncExternalStore } from "react";

const PRIMARY_STORAGE_KEY = "fintrack_sidebar_collapsed";
const LEGACY_STORAGE_KEY = "fintrack.sidebar.collapsed";

// A tiny external store over localStorage. Reading through useSyncExternalStore
// (instead of setState in an effect) keeps the server and first client render
// consistent and lets a toggle notify every subscriber — including other tabs
// via the `storage` event.
const listeners = new Set<() => void>();

function subscribe(onStoreChange: () => void) {
  listeners.add(onStoreChange);
  window.addEventListener("storage", onStoreChange);
  return () => {
    listeners.delete(onStoreChange);
    window.removeEventListener("storage", onStoreChange);
  };
}

function getSnapshot(): boolean {
  try {
    const val = window.localStorage.getItem(PRIMARY_STORAGE_KEY);
    if (val !== null) {
      return val === "true" || val === "1";
    }
    return window.localStorage.getItem(LEGACY_STORAGE_KEY) === "1";
  } catch {
    // localStorage can be unavailable (private mode); default to expanded.
    return false;
  }
}

// Server / hydration snapshot: always expanded, so the markup is deterministic.
function getServerSnapshot(): boolean {
  return false;
}

/**
 * Persists the desktop sidebar's collapsed state in localStorage so the choice
 * survives navigation and reloads.
 */
export function useSidebarCollapsed() {
  const collapsed = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

  const toggle = useCallback(() => {
    try {
      const current = getSnapshot();
      const next = !current;
      window.localStorage.setItem(PRIMARY_STORAGE_KEY, next ? "true" : "false");
      window.localStorage.setItem(LEGACY_STORAGE_KEY, next ? "1" : "0");
    } catch {
      // Ignore persistence failures; notify anyway so the UI still updates.
    }
    for (const listener of listeners) listener();
  }, []);

  return { collapsed, toggle };
}
