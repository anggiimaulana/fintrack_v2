import { Briefcase, Home, Store, UserRound } from "lucide-react";
import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { LANDING } from "@/lib/marketing/copy";

/**
 * "Who it's for" personas (spec doc 03 §B.4.10). Deliberately NOT testimonials:
 * no invented names, quotes, ratings, or user counts — just honest situations the
 * product fits.
 */

const ICONS = [Briefcase, Home, UserRound, Store];

export function PersonaUseCases() {
  const { eyebrow, h2, items } = LANDING.personas;

  return (
    <section id="who-its-for" className="border-b-[3px] border-foreground bg-background">
      <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
        <SectionHeading eyebrow={eyebrow} title={h2} align="center" className="mx-auto mb-12 max-w-3xl" />

        <Reveal>
          <ul className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {items.map((item, index) => {
              const Icon = ICONS[index] ?? UserRound;
              return (
                <li key={item.title} className="neo-card-sm flex flex-col gap-3 p-6">
                  <span className="grid size-10 place-items-center rounded-md border-2 border-foreground bg-muted">
                    <Icon aria-hidden className="size-5" />
                  </span>
                  <h3 className="font-heading text-base font-extrabold tracking-tight">{item.title}</h3>
                  <p className="text-sm leading-relaxed text-muted-foreground">{item.text}</p>
                </li>
              );
            })}
          </ul>
        </Reveal>
      </div>
    </section>
  );
}
