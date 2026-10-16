/**
 * Shared theme constants. Kept out of the client provider module so the server
 * layout can import the anti-flash snippet as a real string (importing a value
 * from a "use client" module would hand the server a client reference instead).
 */
export type Theme = "light" | "dark" | "system";
export type ResolvedTheme = "light" | "dark";

export const THEME_STORAGE_KEY = "theme";

/** Inline snippet for <head>: apply the stored theme before first paint. */
export const THEME_INIT_SCRIPT = `(function(){try{var t=localStorage.getItem("${THEME_STORAGE_KEY}")||"light";var d=t==="system"?(window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"):t;var r=document.documentElement;r.classList.toggle("dark",d==="dark");r.style.colorScheme=d;}catch(e){}})();`;
