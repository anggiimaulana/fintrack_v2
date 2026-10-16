"use client";

import { useEffect, useSyncExternalStore } from "react";
import {
  THEME_STORAGE_KEY,
  type ResolvedTheme,
  type Theme,
} from "@/lib/theme";

export type { ResolvedTheme, Theme };

interface ThemeState {
  theme: Theme;
  resolvedTheme: ResolvedTheme;
}

const DEFAULT_STATE: ThemeState = { theme: "light", resolvedTheme: "light" };

let current: ThemeState = DEFAULT_STATE;
const listeners = new Set<() => void>();

function subscribe(onChange: () => void) {
  listeners.add(onChange);
  return () => {
    listeners.delete(onChange);
  };
}

function resolveTheme(theme: Theme): ResolvedTheme {
  if (theme !== "system") return theme;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(resolved: ResolvedTheme) {
  const root = document.documentElement;
  root.classList.toggle("dark", resolved === "dark");
  root.style.colorScheme = resolved;
}

function readStoredTheme(): Theme {
  try {
    const raw = localStorage.getItem(THEME_STORAGE_KEY);
    if (raw === "light" || raw === "dark" || raw === "system") return raw;
  } catch {
    // Storage unavailable (private mode): fall back to light.
  }
  return "light";
}

function commit(next: ThemeState) {
  current = next;
  applyTheme(next.resolvedTheme);
  for (const listener of listeners) listener();
}

// Adopt the persisted theme when the client module loads (never on the server).
// useSyncExternalStore still serves DEFAULT_STATE during hydration, so this does
// not cause a mismatch — React re-renders with the client snapshot afterwards.
if (typeof window !== "undefined") {
  const theme = readStoredTheme();
  commit({ theme, resolvedTheme: resolveTheme(theme) });
}

export function setTheme(next: Theme) {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, next);
  } catch {
    // Persisting is best-effort; the in-memory choice still applies.
  }
  commit({ theme: next, resolvedTheme: resolveTheme(next) });
}

/**
 * Theme state backed by an external store rather than `next-themes`, which
 * renders an inline `<script>` inside a client component. React 19 refuses to
 * execute scripts rendered by a client component and logs "Encountered a script
 * tag while rendering React component" on every page. The anti-flash script now
 * lives in the server layout's <head>; this hook only toggles the `dark` class.
 */
export function useTheme(): ThemeState & { setTheme: (theme: Theme) => void } {
  const state = useSyncExternalStore(subscribe, () => current, () => DEFAULT_STATE);

  // Follow the OS preference while "system" is selected.
  useEffect(() => {
    if (state.theme !== "system") return;
    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => commit({ theme: "system", resolvedTheme: resolveTheme("system") });
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, [state.theme]);

  return { ...state, setTheme };
}
