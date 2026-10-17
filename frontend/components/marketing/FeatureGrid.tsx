import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { LANDING } from "@/lib/marketing/copy";
import { FEATURES, featureHref } from "@/lib/marketing/features";

/**
 * Featured feature grid (spec doc 03 §B.4.5): six cards, one per catalogue
 * feature, each linking to its deep-dive page. Copy comes from the shared
 * feature model so the landing, the hub, and the pricing table never diverge.
 */
export function FeatureGrid() {
  const { eyebrow, h2, subhead } = LANDING.featureSection;

  return (
    <section id="features" className="scroll-mt-24 border-b-[3px] border-foreground bg-background">
      <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
        <SectionHeading
          eyebrow={eyebrow}
          title={h2}
          description={subhead}
          align="center"
          className="mx-auto mb-12 max-w-3xl"
        />

        <Reveal>
          <ul className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map((feature) => {
              const Icon = feature.icon;
              return (
                <li key={feature.slug} className="neo-card-sm flex flex-col gap-4 p-6">
                  <span className="grid size-12 place-items-center rounded-md border-2 border-foreground bg-accent-acid">
                    <Icon aria-hidden className="size-6 text-accent-acid-foreground" />
                  </span>
                  <h3 className="font-heading text-lg font-extrabold tracking-tight">{feature.title}</h3>
                  <p className="text-sm leading-relaxed text-muted-foreground">{feature.tagline}</p>
                  <Link
                    href={featureHref(feature.slug)}
                    className="mt-auto inline-flex items-center gap-1.5 text-sm font-bold underline decoration-accent-acid decoration-[3px] underline-offset-4 hover:decoration-foreground"
                  >
                    Pelajari lebih lanjut
                    <ArrowRight aria-hidden className="size-4" />
                    <span className="sr-only"> tentang {feature.title}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </Reveal>
      </div>
    </section>
  );
}
