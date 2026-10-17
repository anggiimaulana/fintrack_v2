import { LANDING } from "@/lib/marketing/copy";

/**
 * Oversized ticker band (signature element #3, spec doc 03 §B.4.8): an acid-
 * yellow separator ribbon with dark text. The moving track is `aria-hidden` and
 * carries one static text equivalent for screen readers. Stops on hover and
 * wraps into a static line under `prefers-reduced-motion`.
 */

function TickerRow({ clone = false }: { clone?: boolean }) {
  return (
    <ul className={`flex shrink-0 items-center${clone ? " neo-marquee-clone" : ""}`}>
      {LANDING.ticker.words.map((word) => (
        <li key={word} className="flex items-center">
          <span className="px-4 py-3 font-heading text-2xl font-black uppercase tracking-tight sm:text-3xl">
            {word}
          </span>
          <span aria-hidden className="font-heading text-2xl font-black sm:text-3xl">
            ·
          </span>
        </li>
      ))}
    </ul>
  );
}

export function TickerBand() {
  return (
    <aside
      aria-label={LANDING.ticker.ariaLabel}
      className="overflow-hidden border-y-[3px] border-foreground bg-accent-acid text-accent-acid-foreground"
    >
      <div aria-hidden className="neo-marquee flex w-max">
        <TickerRow />
        <TickerRow clone />
      </div>
      <p className="sr-only">{LANDING.ticker.ariaLabel}</p>
    </aside>
  );
}
