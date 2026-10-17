"use client";

import * as React from "react";
import { LANDING } from "@/lib/marketing/copy";

/**
 * Honest stat counters (spec doc 03 §B.4.3): only numbers that are literally
 * true. The final value is server-rendered into the DOM; the count-up is a
 * decorative enhancement that runs once when the block scrolls into view and is
 * skipped entirely for `prefers-reduced-motion`.
 */

function formatStat(prefix: string, value: number, suffix: string): string {
  return `${prefix}${value.toLocaleString("id-ID")}${suffix}`;
}

function Counter({ prefix, value, suffix }: { prefix: string; value: number; suffix: string }) {
  const ref = React.useRef<HTMLSpanElement>(null);
  const final = formatStat(prefix, value, suffix);

  React.useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (typeof IntersectionObserver === "undefined") return;
    if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) return;
    if (value === 0) return;

    let frame = 0;
    const run = () => {
      const duration = 900;
      const start = performance.now();
      const step = (now: number) => {
        const progress = Math.min(1, (now - start) / duration);
        const eased = 1 - Math.pow(1 - progress, 3);
        el.textContent = formatStat(prefix, Math.round(value * eased), suffix);
        if (progress < 1) {
          frame = requestAnimationFrame(step);
        } else {
          el.textContent = final;
        }
      };
      frame = requestAnimationFrame(step);
    };

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            observer.disconnect();
            run();
          }
        }
      },
      { threshold: 0.4 },
    );
    observer.observe(el);
    return () => {
      observer.disconnect();
      cancelAnimationFrame(frame);
    };
  }, [final, prefix, suffix, value]);

  return (
    <span ref={ref} className="font-heading text-[clamp(2.5rem,5vw,3.5rem)] font-black leading-none">
      {final}
    </span>
  );
}

export function StatCounters() {
  return (
    <ul className="grid gap-6 sm:grid-cols-3">
      {LANDING.stats.map((stat) => (
        <li key={stat.label} className="neo-card-sm flex flex-col gap-2 p-6">
          <Counter prefix={stat.prefix} value={stat.value} suffix={stat.suffix} />
          <span className="text-sm font-semibold text-muted-foreground">{stat.label}</span>
        </li>
      ))}
    </ul>
  );
}
