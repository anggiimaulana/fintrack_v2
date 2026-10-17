"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

/**
 * Scroll-reveal wrapper (spec doc 03 §A.5).
 *
 * Only opacity/transform animate (see `.neo-reveal` in `globals.css`). The
 * observer flips a single data attribute on the element — no React state and no
 * render on scroll, so long pages stay cheap. Users with `prefers-reduced-motion`
 * get the final state from CSS; browsers without `IntersectionObserver` are
 * revealed immediately.
 */
export function Reveal({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  const ref = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const reveal = () => {
      el.dataset.visible = "true";
    };

    if (typeof IntersectionObserver === "undefined") {
      reveal();
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            reveal();
            observer.disconnect();
          }
        }
      },
      { threshold: 0.15, rootMargin: "0px 0px -8% 0px" },
    );

    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return (
    <div ref={ref} data-visible="false" className={cn("neo-reveal", className)}>
      {children}
    </div>
  );
}
