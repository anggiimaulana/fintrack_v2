import { ArrowRight } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { LANDING } from "@/lib/marketing/copy";

/**
 * Closing call to action (spec doc 03 §B.4.12) on a dark surface, with one acid
 * block as the only accent. The subhead replaces the old, unverifiable
 * "ribuan pengguna" claim with what the product actually offers.
 */
export function CtaSection() {
  const { h2, subhead, cta } = LANDING.closingCta;

  return (
    <section aria-labelledby="closing-cta-heading" className="neo-dark border-b-[3px] border-foreground">
      <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
        <div className="mx-auto flex max-w-4xl flex-col items-center gap-6 rounded-lg border-[3px] border-white/20 bg-surface-dark-2 p-8 text-center sm:p-12">
          <span className="rounded-md border-[3px] border-foreground bg-accent-acid px-4 py-1 font-mono text-xs font-bold uppercase tracking-[0.12em] text-accent-acid-foreground">
            Gratis untuk memulai
          </span>
          <h2
            id="closing-cta-heading"
            className="font-heading text-[clamp(1.875rem,4vw,3rem)] font-extrabold leading-[1.1] tracking-[-0.02em] text-white text-balance"
          >
            {h2}
          </h2>
          <p className="max-w-2xl text-lg leading-relaxed text-neutral-200">{subhead}</p>
          <ActionLink href={cta.href} tone="acid" size="lg">
            {cta.label}
            <ArrowRight aria-hidden className="size-4" />
          </ActionLink>
        </div>
      </div>
    </section>
  );
}
