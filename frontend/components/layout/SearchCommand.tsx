"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Search, CornerDownLeft } from "lucide-react";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { searchEntries, SEARCH_ENTRIES, type SearchEntry } from "@/lib/search-catalog";

interface SearchCommandProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Groups entries by their `group` field, preserving the ranked order. */
function groupResults(entries: SearchEntry[]): { group: string; items: SearchEntry[] }[] {
  const order: SearchEntry["group"][] = ["Modul", "Fitur", "Aksi"];
  return order
    .map((group) => ({ group, items: entries.filter((e) => e.group === group) }))
    .filter((g) => g.items.length > 0);
}

/**
 * Command palette for the header: type to find a module or feature, then click
 * (or press Enter) to navigate. Purely a navigator — the backend stays the gate.
 */
export function SearchCommand({ open, onOpenChange }: SearchCommandProps) {
  const router = useRouter();
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [query, setQuery] = React.useState("");
  const [activeIndex, setActiveIndex] = React.useState(0);

  const results = React.useMemo(
    () => (query.trim() ? searchEntries(query) : SEARCH_ENTRIES),
    [query]
  );
  const groups = React.useMemo(() => groupResults(results), [results]);
  const flat = React.useMemo(() => groups.flatMap((g) => g.items), [groups]);

  // Reset the query when the palette opens so it never reopens mid-search, and
  // reset the highlighted row on every keystroke — both done in the event
  // handlers rather than effects to avoid a redundant render pass.
  const handleOpenChange = (next: boolean) => {
    if (next) {
      setQuery("");
      setActiveIndex(0);
    }
    onOpenChange(next);
  };

  const onQueryChange = (value: string) => {
    setQuery(value);
    setActiveIndex(0);
  };

  const go = React.useCallback(
    (entry: SearchEntry | undefined) => {
      if (!entry) return;
      onOpenChange(false);
      router.push(entry.href);
    },
    [onOpenChange, router]
  );

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex((i) => Math.min(i + 1, flat.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      go(flat[activeIndex]);
    }
  };

  let flatCursor = -1;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className="p-0 gap-0 border-2 rounded-md border-foreground shadow-[6px_6px_0px_0px] shadow-foreground bg-card sm:max-w-lg top-[15%] translate-y-0"
        showCloseButton={false}
        onOpenAutoFocus={(e) => {
          // Radix would focus the content container; focus the input instead so
          // typing starts immediately without relying on the autoFocus prop.
          e.preventDefault();
          inputRef.current?.focus();
        }}
      >
        <DialogTitle className="sr-only">Cari modul atau fitur</DialogTitle>

        <div className="flex items-center gap-2 px-4 py-3 border-b-2 border-foreground">
          <Search className="w-4 h-4 text-muted-foreground shrink-0" aria-hidden="true" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="Cari modul atau fitur…"
            aria-label="Kata kunci pencarian"
            className="flex-1 bg-transparent text-sm font-bold focus:outline-none placeholder:text-muted-foreground/70"
          />
          <kbd className="hidden sm:inline-flex items-center gap-1 text-[10px] font-bold text-muted-foreground border-2 rounded-md border-foreground/30 px-1.5 py-0.5">
            ESC
          </kbd>
        </div>

        <div className="max-h-[60vh] overflow-y-auto py-2">
          {flat.length === 0 ? (
            <p className="px-4 py-8 text-center text-sm font-bold text-muted-foreground">
              Tidak ada hasil untuk “{query}”.
            </p>
          ) : (
            groups.map(({ group, items }) => (
              <div key={group} className="mb-1">
                <p className="px-4 py-1 text-[10px] font-black uppercase tracking-wider text-muted-foreground">{group}</p>
                {items.map((entry) => {
                  flatCursor += 1;
                  const idx = flatCursor;
                  const active = idx === activeIndex;
                  const Icon = entry.icon;
                  return (
                    <button
                      key={entry.id}
                      type="button"
                      onMouseEnter={() => setActiveIndex(idx)}
                      onClick={() => go(entry)}
                      className={`w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors cursor-pointer ${
                        active ? "bg-primary text-primary-foreground" : "hover:bg-muted"
                      }`}
                    >
                      <Icon className="w-4 h-4 shrink-0" aria-hidden="true" />
                      <span className="min-w-0 flex-1">
                        <span className="block text-sm font-bold truncate">{entry.label}</span>
                        {entry.description && (
                          <span className={`block text-xs font-medium truncate ${active ? "text-primary-foreground/80" : "text-muted-foreground"}`}>
                            {entry.description}
                          </span>
                        )}
                      </span>
                      {active && <CornerDownLeft className="w-3.5 h-3.5 shrink-0 opacity-70" aria-hidden="true" />}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
