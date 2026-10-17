import { ArrowRight } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { InteractiveDemo } from "./InteractiveDemo";
import { LANDING } from "@/lib/marketing/copy";

/**
 * Landing hero (spec doc 03 §B.4.1). Server component: the only interactive
 * element is the embedded `InteractiveDemo`. The H1 carries the primary keyword
 * "catat keuangan otomatis" and highlights the word "tenang" with the acid
 * accent (dark text on the accent, never white).
 */
function HighlightedH1() {
  const { h1, h1HighlightWord } = LANDING.hero;
  const index = h1.indexOf(h1HighlightWord);
  if (index === -1) return <>{h1}</>;
  return (
    <>
      {h1.slice(0, index)}
      <span className="neo-highlight text-[0.85em]">{h1HighlightWord}</span>
      {h1.slice(index + h1HighlightWord.length)}
    </>
  );
}

export function Hero() {
  const { eyebrow, subhead, primaryCta, secondaryCta, trustChips } = LANDING.hero;

  return (
    <section id="hero" className="neo-dark border-b-[3px] border-foreground">
      <div className="mx-auto grid max-w-7xl items-center gap-12 px-4 py-16 lg:grid-cols-2 lg:py-24">
        <div className="flex flex-col gap-6">
          <span className="font-mono text-xs font-bold uppercase tracking-[0.12em] text-neutral-300">
            {eyebrow}
          </span>

          <h1 className="font-heading text-[clamp(2.5rem,6vw,4.25rem)] font-extrabold leading-[1.05] tracking-[-0.02em] text-white text-balance">
            <HighlightedH1 />
          </h1>

          <p className="max-w-xl text-lg leading-relaxed text-neutral-200">{subhead}</p>

          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <ActionLink href={primaryCta.href} tone="acid" size="lg">
              {primaryCta.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
            <ActionLink href={secondaryCta.href} tone="outlineOnDark" size="lg">
              {secondaryCta.label}
            </ActionLink>
          </div>

          <ul className="flex flex-wrap gap-x-6 gap-y-2 text-sm text-neutral-300">
            {trustChips.map((chip) => (
              <li key={chip} className="flex items-center gap-2">
                <span aria-hidden className="size-1.5 rounded-full bg-accent-acid-strong" />
                {chip}
              </li>
            ))}
          </ul>
        </div>

        <InteractiveDemo />
      </div>
    </section>
  );
}
