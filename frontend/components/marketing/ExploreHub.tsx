import Link from "next/link";
import { ArrowRight, FileText, Headphones, HelpCircle, Info, ShieldCheck } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { Breadcrumbs } from "./Breadcrumbs";
import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { EXPLORE_HUB } from "@/lib/marketing/copy";

const ICON_MAP = {
  HelpCircle,
  Info,
  Headphones,
  ShieldCheck,
  FileText,
} as const;

export function ExploreHub() {
  return (
    <>
      <section className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-7xl flex-col gap-6 px-4 py-12 lg:py-16">
          <Breadcrumbs
            items={[
              { name: "Beranda", path: "/" },
              { name: "Jelajahi", path: "/explore" },
            ]}
          />
          <SectionHeading
            as="h1"
            eyebrow={EXPLORE_HUB.eyebrow}
            title={EXPLORE_HUB.h1}
            description={EXPLORE_HUB.subhead}
          />
          <div className="flex flex-wrap gap-3">
            <ActionLink href={EXPLORE_HUB.cta.primary.href} tone="acid">
              {EXPLORE_HUB.cta.primary.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
            <ActionLink href={EXPLORE_HUB.cta.secondary.href} tone="outline">
              {EXPLORE_HUB.cta.secondary.label}
            </ActionLink>
          </div>
        </div>
      </section>

      <section
        aria-label={EXPLORE_HUB.gridTitle}
        className="border-b-[3px] border-foreground bg-muted/40"
      >
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          <Reveal>
            <ul className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
              {EXPLORE_HUB.cards.map((card) => {
                const Icon = ICON_MAP[card.iconName];
                return (
                  <li
                    key={card.href}
                    className="neo-card-sm flex flex-col justify-between gap-6 p-6 transition-transform hover:-translate-y-1"
                  >
                    <div className="flex flex-col gap-4">
                      <div className="flex items-center justify-between">
                        <span className="grid size-12 place-items-center rounded-md border-2 border-foreground bg-accent-acid">
                          <Icon aria-hidden className="size-6 text-accent-acid-foreground" />
                        </span>
                        <span className="rounded-md border-2 border-foreground bg-muted px-2.5 py-0.5 font-mono text-xs font-bold uppercase tracking-wider text-muted-foreground">
                          {card.badge}
                        </span>
                      </div>
                      <h2 className="font-heading text-xl font-extrabold tracking-tight">
                        {card.title}
                      </h2>
                      <p className="text-sm leading-relaxed text-muted-foreground">
                        {card.tagline}
                      </p>
                    </div>

                    <Link
                      href={card.href}
                      aria-label={`${card.cta}: ${card.title}`}
                      className="inline-flex items-center gap-1.5 text-sm font-bold underline decoration-accent-acid decoration-[3px] underline-offset-4 hover:decoration-foreground"
                    >
                      {card.cta}
                      <ArrowRight aria-hidden className="size-4" />
                    </Link>
                  </li>
                );
              })}
            </ul>
          </Reveal>
        </div>
      </section>

      <section aria-labelledby="explore-closing-heading" className="neo-dark border-b-[3px] border-foreground">
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          <Reveal className="flex flex-col items-center gap-6 text-center">
            <h2
              id="explore-closing-heading"
              className="max-w-2xl font-heading text-3xl font-extrabold tracking-tight text-white sm:text-4xl"
            >
              {EXPLORE_HUB.cta.h2}
            </h2>
            <p className="max-w-xl text-base leading-relaxed text-neutral-300">
              {EXPLORE_HUB.cta.subhead}
            </p>
            <div className="flex flex-wrap justify-center gap-3">
              <ActionLink href={EXPLORE_HUB.cta.primary.href} tone="acid" size="lg">
                {EXPLORE_HUB.cta.primary.label}
                <ArrowRight aria-hidden className="size-4" />
              </ActionLink>
              <ActionLink href={EXPLORE_HUB.cta.secondary.href} tone="outline" size="lg">
                {EXPLORE_HUB.cta.secondary.label}
              </ActionLink>
            </div>
          </Reveal>
        </div>
      </section>
    </>
  );
}
