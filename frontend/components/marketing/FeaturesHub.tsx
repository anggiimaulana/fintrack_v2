import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { Breadcrumbs } from "./Breadcrumbs";
import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { FEATURES_HUB } from "@/lib/marketing/copy";
import { FEATURES, featureHref } from "@/lib/marketing/features";

/**
 * `/features` hub (spec doc 04 §B.7): H1, a six-card grid built from the shared
 * feature model, the "how it all connects" explainer, and the closing CTA.
 * Server component; the breadcrumb supplies the `BreadcrumbList` JSON-LD.
 */
export function FeaturesHub() {
  return (
    <>
      <section className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-7xl flex-col gap-6 px-4 py-12 lg:py-16">
          <Breadcrumbs
            items={[
              { name: "Beranda", path: "/" },
              { name: "Fitur", path: "/features" },
            ]}
          />
          <SectionHeading
            as="h1"
            eyebrow="FITUR"
            title={FEATURES_HUB.h1}
            description={FEATURES_HUB.subhead}
          />
          <div className="flex flex-wrap gap-3">
            <ActionLink href={FEATURES_HUB.cta.href} tone="acid">
              {FEATURES_HUB.cta.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
            <ActionLink href={FEATURES_HUB.pricingCta.href} tone="outline">
              {FEATURES_HUB.pricingCta.label}
            </ActionLink>
          </div>
        </div>
      </section>

      <section
        aria-label={FEATURES_HUB.gridTitle}
        className="border-b-[3px] border-foreground bg-background"
      >
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          <Reveal>
            <ul className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
              {FEATURES.map((feature) => {
                const Icon = feature.icon;
                return (
                  <li key={feature.slug} className="neo-card-sm flex flex-col gap-4 p-6">
                    <span className="grid size-12 place-items-center rounded-md border-2 border-foreground bg-accent-acid">
                      <Icon aria-hidden className="size-6 text-accent-acid-foreground" />
                    </span>
                    <h2 className="font-heading text-lg font-extrabold tracking-tight">
                      {feature.title}
                    </h2>
                    <p className="text-sm leading-relaxed text-muted-foreground">{feature.tagline}</p>
                    <Link
                      href={featureHref(feature.slug)}
                      aria-label={`${FEATURES_HUB.cardCta}: ${feature.title}`}
                      className="mt-auto inline-flex items-center gap-1.5 text-sm font-bold underline decoration-accent-acid decoration-[3px] underline-offset-4 hover:decoration-foreground"
                    >
                      {FEATURES_HUB.cardCta}
                      <ArrowRight aria-hidden className="size-4" />
                    </Link>
                  </li>
                );
              })}
            </ul>
          </Reveal>
        </div>
      </section>

      <section
        aria-labelledby="features-connected-heading"
        className="border-b-[3px] border-foreground bg-muted"
      >
        <div className="mx-auto grid max-w-7xl gap-8 px-4 py-16 lg:grid-cols-2 lg:py-24">
          <SectionHeading
            id="features-connected-heading"
            eyebrow="SATU ALUR"
            title={FEATURES_HUB.connected.title}
          />
          <Reveal>
            <div className="flex flex-col gap-4">
              {FEATURES_HUB.connected.paragraphs.map((paragraph) => (
                <p key={paragraph} className="text-base leading-relaxed text-muted-foreground">
                  {paragraph}
                </p>
              ))}
              <ActionLink href={FEATURES_HUB.pricingCta.href} tone="outline" className="self-start">
                {FEATURES_HUB.pricingCta.label}
              </ActionLink>
            </div>
          </Reveal>
        </div>
      </section>

      <section aria-labelledby="features-closing-heading" className="neo-dark border-b-[3px] border-foreground">
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 text-center">
            <h2
              id="features-closing-heading"
              className="font-heading text-[clamp(1.875rem,4vw,3rem)] font-extrabold leading-[1.1] tracking-[-0.02em] text-white text-balance"
            >
              Mulai rapikan keuangan Anda hari ini
            </h2>
            <ActionLink href={FEATURES_HUB.cta.href} tone="acid" size="lg">
              {FEATURES_HUB.cta.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
          </div>
        </div>
      </section>
    </>
  );
}
