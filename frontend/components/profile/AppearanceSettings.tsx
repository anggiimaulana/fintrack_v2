"use client";

import { useEffect, useState } from "react";
import { useTheme } from "next-themes";
import { cn } from "@/lib/utils";

const THEME_OPTIONS: { value: "light" | "dark" | "system"; label: string }[] = [
  { value: "light", label: "Terang" },
  { value: "dark", label: "Gelap" },
  { value: "system", label: "Sistem" },
];

/**
 * Appearance (theme) and language. Theme drives next-themes directly so the
 * choice takes effect immediately and the active option is visibly highlighted.
 * Language is fixed to Indonesian until an i18n layer exists.
 */
export function AppearanceSettings() {
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  // next-themes only knows the theme after hydration; render the buttons as
  // inactive until then to avoid a server/client mismatch.
  useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div className="space-y-3">
        <p className="text-xs font-bold uppercase tracking-wide text-foreground">Tema</p>
        <div className="flex gap-3">
          {THEME_OPTIONS.map((option) => {
            const isActive = mounted && theme === option.value;
            return (
              <button
                key={option.value}
                type="button"
                aria-pressed={isActive}
                onClick={() => setTheme(option.value)}
                className={cn(
                  "flex-1 py-2.5 text-sm font-bold border-2 border-foreground transition-all cursor-pointer",
                  isActive
                    ? "bg-primary text-primary-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                    : "bg-muted text-muted-foreground hover:text-foreground"
                )}
              >
                {option.label}
              </button>
            );
          })}
        </div>
      </div>

      <div className="space-y-3">
        <label htmlFor="settings-language" className="text-xs font-bold uppercase tracking-wide text-foreground">
          Bahasa &amp; Regional
        </label>
        {/* Only Indonesian ships today, so the field is a fixed, disabled value
            rather than a picker that would pretend English works. */}
        <select
          id="settings-language"
          value="id"
          disabled
          className="w-full border-2 border-foreground bg-muted px-4 py-2.5 text-sm font-bold text-muted-foreground cursor-not-allowed focus:outline-none"
        >
          <option value="id">Indonesia (ID)</option>
        </select>
        <p className="text-[11px] font-bold text-muted-foreground">
          Saat ini baru tersedia Bahasa Indonesia. Dukungan bahasa lain segera hadir.
        </p>
      </div>
    </div>
  );
}
