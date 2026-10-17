import { LANDING } from "@/lib/marketing/copy";

/**
 * Receipt-tape marquee (signature element #1, spec doc 03 §B.4.2).
 *
 * Decorative sample transactions on a slow horizontal loop. The moving track is
 * `aria-hidden`; the same rows are exposed once to screen readers, and the
 * caption makes clear this is an illustration, not user data. Stops on hover and
 * turns into a wrapped static list under `prefers-reduced-motion`.
 */

function TapeRows({ clone = false }: { clone?: boolean }) {
  return (
    <ul className={`flex shrink-0${clone ? " neo-marquee-clone" : ""}`}>
      {LANDING.receiptTape.rows.map((row) => (
        <li
          key={row.label}
          className="flex items-center gap-4 whitespace-nowrap px-6 py-3 font-mono text-sm"
        >
          <span className="font-bold">{row.label}</span>
          <span className="tabular-nums">{row.amount}</span>
          <span className="rounded-sm border-2 border-foreground bg-background px-2 py-0.5 text-[0.65rem] font-bold uppercase tracking-wide">
            {row.category}
          </span>
        </li>
      ))}
    </ul>
  );
}

export function ReceiptTapeMarquee() {
  return (
    <section aria-labelledby="receipt-tape-caption" className="bg-muted">
      <div className="overflow-hidden border-y-[3px] border-foreground">
        <div aria-hidden className="neo-marquee flex w-max">
          <TapeRows />
          <TapeRows clone />
        </div>
      </div>
      <ul className="sr-only">
        {LANDING.receiptTape.rows.map((row) => (
          <li key={`sr-${row.label}`}>
            {row.label} {row.amount} — {row.category}
          </li>
        ))}
      </ul>
      <p id="receipt-tape-caption" className="mx-auto max-w-7xl px-4 py-3 text-center text-xs text-muted-foreground">
        {LANDING.receiptTape.caption}
      </p>
    </section>
  );
}
