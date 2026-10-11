"use client";

import { useState, useCallback, useMemo } from "react";
import { Calendar, ArrowRight, SwitchCamera } from "lucide-react";

export type PresetKey =
  | "today"
  | "this_week"
  | "this_month"
  | "last_month"
  | "3months"
  | "this_year"
  | "custom";

export interface DateRangeValue {
  preset: PresetKey;
  startDate: string; // YYYY-MM-DD
  endDate: string;
  compareStart: string;
  compareEnd: string;
  enableCompare: boolean;
}

interface DateRangeFilterProps {
  value: DateRangeValue;
  onChange: (value: DateRangeValue) => void;
  showCompare?: boolean;
}

function formatDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function getWeekRange(now: Date): { start: Date; end: Date } {
  const day = now.getDay();
  const diff = day === 0 ? -6 : 1 - day;
  const monday = new Date(now);
  monday.setDate(now.getDate() + diff);
  monday.setHours(0, 0, 0, 0);
  const sunday = new Date(monday);
  sunday.setDate(monday.getDate() + 6);
  sunday.setHours(23, 59, 59, 999);
  return { start: monday, end: sunday };
}

function getMonthRange(now: Date, offset: number): { start: Date; end: Date } {
  const start = new Date(now.getFullYear(), now.getMonth() + offset, 1, 0, 0, 0, 0);
  const end = new Date(now.getFullYear(), now.getMonth() + offset + 1, 0, 23, 59, 59, 999);
  return { start, end };
}

export function computePresetDates(preset: PresetKey, now: Date): { start: string; end: string } {
  let start: Date, end: Date;
  switch (preset) {
    case "today":
      start = end = now;
      break;
    case "this_week": {
      const w = getWeekRange(now);
      start = w.start;
      end = w.end;
      break;
    }
    case "this_month": {
      start = new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0, 0);
      end = now;
      break;
    }
    case "last_month": {
      const lm = getMonthRange(now, -1);
      start = lm.start;
      end = lm.end;
      break;
    }
    case "3months": {
      start = new Date(now.getFullYear(), now.getMonth() - 2, 1, 0, 0, 0, 0);
      end = now;
      break;
    }
    case "this_year": {
      start = new Date(now.getFullYear(), 0, 1, 0, 0, 0, 0);
      end = now;
      break;
    }
    default:
      start = end = now;
  }
  return { start: formatDate(start), end: formatDate(end) };
}

function computeCompareDates(startDate: string, endDate: string, comparePreset: "prev_period" | "year_ago"): { start: string; end: string } {
  const s = new Date(startDate);
  const e = new Date(endDate);
  const diffMs = e.getTime() - s.getTime();

  if (comparePreset === "year_ago") {
    const cs = new Date(s.getFullYear() - 1, s.getMonth(), s.getDate());
    const ce = new Date(e.getFullYear() - 1, e.getMonth(), e.getDate());
    return { start: formatDate(cs), end: formatDate(ce) };
  }

  // previous period
  const prevEnd = new Date(s.getTime() - 86400000);
  const prevStart = new Date(prevEnd.getTime() - diffMs);
  return { start: formatDate(prevStart), end: formatDate(prevEnd) };
}

const PRESETS: { key: PresetKey; label: string }[] = [
  { key: "today", label: "Hari Ini" },
  { key: "this_week", label: "Minggu Ini" },
  { key: "this_month", label: "Bulan Ini" },
  { key: "last_month", label: "Bulan Lalu" },
  { key: "3months", label: "3 Bulan" },
  { key: "this_year", label: "Tahun Ini" },
  { key: "custom", label: "Kustom" },
];

export function DateRangeFilter({ value, onChange, showCompare = true }: DateRangeFilterProps) {
  const now = useMemo(() => new Date(), []);
  const [localComparePreset, setLocalComparePreset] = useState<"prev_period" | "year_ago">("prev_period");

  const handlePreset = useCallback(
    (preset: PresetKey) => {
      if (preset === "custom") {
        onChange({ ...value, preset });
        return;
      }
      const { start, end } = computePresetDates(preset, now);
      const next: DateRangeValue = {
        preset,
        startDate: start,
        endDate: end,
        compareStart: value.compareStart,
        compareEnd: value.compareEnd,
        enableCompare: value.enableCompare,
      };
      if (next.enableCompare) {
        const cmp = computeCompareDates(start, end, localComparePreset);
        next.compareStart = cmp.start;
        next.compareEnd = cmp.end;
      }
      onChange(next);
    },
    [now, onChange, value, localComparePreset]
  );

  const toggleCompare = useCallback(() => {
    const enable = !value.enableCompare;
    let cmpStart = value.compareStart;
    let cmpEnd = value.compareEnd;
    if (enable) {
      const cmp = computeCompareDates(value.startDate, value.endDate, localComparePreset);
      cmpStart = cmp.start;
      cmpEnd = cmp.end;
    }
    onChange({ ...value, enableCompare: enable, compareStart: cmpStart, compareEnd: cmpEnd });
  }, [value, onChange, localComparePreset]);

  const handleComparePreset = useCallback(
    (preset: "prev_period" | "year_ago") => {
      setLocalComparePreset(preset);
      const cmp = computeCompareDates(value.startDate, value.endDate, preset);
      onChange({ ...value, compareStart: cmp.start, compareEnd: cmp.end });
    },
    [value, onChange]
  );

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-1 flex-wrap">
        <Calendar className="w-4 h-4 text-muted-foreground shrink-0 mr-1" />
        {PRESETS.map((p) => (
          <button
            key={p.key}
            onClick={() => handlePreset(p.key)}
            className={`px-3 py-1.5 text-xs font-bold uppercase cursor-pointer transition-all border-2 ${
              value.preset === p.key
                ? "bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground border-foreground"
                : "bg-background text-muted-foreground border-transparent hover:border-foreground hover:text-foreground"
            }`}
          >
            {p.label}
          </button>
        ))}
      </div>

      {value.preset === "custom" && (
        <div className="flex items-center gap-3 flex-wrap">
          <div className="flex items-center gap-2">
            <label htmlFor="drf-from" className="text-xs font-bold text-muted-foreground">Dari</label>
            <input
              id="drf-from"
              type="date"
              value={value.startDate}
              onChange={(e) => onChange({ ...value, startDate: e.target.value })}
              className="border-2 border-foreground bg-background px-2 py-1 text-xs font-bold focus:outline-none"
            />
          </div>
          <ArrowRight className="w-4 h-4 text-muted-foreground" />
          <div className="flex items-center gap-2">
            <label htmlFor="drf-to" className="text-xs font-bold text-muted-foreground">Ke</label>
            <input
              id="drf-to"
              type="date"
              value={value.endDate}
              onChange={(e) => onChange({ ...value, endDate: e.target.value })}
              className="border-2 border-foreground bg-background px-2 py-1 text-xs font-bold focus:outline-none"
            />
          </div>
        </div>
      )}

      {showCompare && (
        <div className="flex items-center gap-4 flex-wrap">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={value.enableCompare}
              onChange={toggleCompare}
              className="w-4 h-4 cursor-pointer"
            />
            <span className="text-xs font-bold text-muted-foreground">Bandingkan dengan periode sebelumnya</span>
          </label>

          {value.enableCompare && (
            <>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleComparePreset("prev_period")}
                  className={`px-2 py-1 text-[10px] font-bold uppercase cursor-pointer border-2 ${
                    localComparePreset === "prev_period"
                      ? "bg-card text-foreground border-foreground"
                      : "bg-background text-muted-foreground border-transparent"
                  }`}
                >
                  Periode Sebelumnya
                </button>
                <button
                  onClick={() => handleComparePreset("year_ago")}
                  className={`px-2 py-1 text-[10px] font-bold uppercase cursor-pointer border-2 ${
                    localComparePreset === "year_ago"
                      ? "bg-card text-foreground border-foreground"
                      : "bg-background text-muted-foreground border-transparent"
                  }`}
                >
                  Tahun Lalu
                </button>
              </div>
              <div className="flex items-center gap-2 flex-wrap">
                <div className="flex items-center gap-1">
                  <SwitchCamera className="w-3 h-3 text-muted-foreground" />
                  <input
                    type="date"
                    value={value.compareStart}
                    onChange={(e) => onChange({ ...value, compareStart: e.target.value })}
                    className="border-2 border-foreground bg-background px-2 py-1 text-xs font-bold focus:outline-none w-[140px]"
                  />
                  <span className="text-xs text-muted-foreground">→</span>
                  <input
                    type="date"
                    value={value.compareEnd}
                    onChange={(e) => onChange({ ...value, compareEnd: e.target.value })}
                    className="border-2 border-foreground bg-background px-2 py-1 text-xs font-bold focus:outline-none w-[140px]"
                  />
                </div>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
