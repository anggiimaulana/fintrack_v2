import Link from "next/link";
import { ArrowRight, Check, Info } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { Breadcrumbs } from "./Breadcrumbs";
import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { FEATURE_PAGE } from "@/lib/marketing/copy";
import { getFeature, featureHref, PLAN_LABEL, type FeaturePage } from "@/lib/marketing/features";

/**
 * Shared body for every `/features/{slug}` page (spec doc 04 §B.4): problem,
 * how it works, deeper detail, benefits, a concrete example, use cases, honest
 * notes, the plan badge, cross-links, and a closing CTA. The visible
 * `BreadcrumbList` JSON-LD comes from `Breadcrumbs`; the route adds a `WebPage`
 * node with `about`.
 */
export function FeatureDetail({ feature }: { feature: FeaturePage }) {
  const Icon = feature.icon;
  const related = feature.relatedSlugs
    .map((slug) => getFeature(slug))
    .filter((item): item is FeaturePage => Boolean(item));
  const availability = feature.enabledInPlans.map((code) => PLAN_LABEL[code]).join(", ");

  return (
    <>
      <section className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-12 lg:py-16">
          <Breadcrumbs
            items={[
              { name: "Beranda", path: "/" },
              { name: "Fitur", path: "/features" },
              { name: feature.title, path: featureHref(feature.slug) },
            ]}
          />
          <span className="grid size-14 place-items-center rounded-md border-[3px] border-foreground bg-accent-acid shadow-[4px_4px_0px_0px] shadow-foreground">
            <Icon aria-hidden className="size-7 text-accent-acid-foreground" />
          </span>
          <SectionHeading
            as="h1"
            eyebrow="FITUR"
            title={feature.title}
            description={feature.tagline}
          />
          <p className="text-lg leading-relaxed text-muted-foreground">{feature.heroDescription}</p>
          <div className="flex flex-wrap gap-3">
            <ActionLink href={FEATURE_PAGE.primaryCta.href} tone="acid">
              {FEATURE_PAGE.primaryCta.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
            <ActionLink href={FEATURE_PAGE.pricingCta.href} tone="outline">
              {FEATURE_PAGE.pricingCta.label}
            </ActionLink>
          </div>
        </div>
      </section>

      <section aria-labelledby="feature-problem" className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto grid max-w-4xl gap-3 px-4 py-14 lg:grid-cols-[auto_1fr] lg:gap-8">
          <h2 id="feature-problem" className="font-heading text-2xl font-extrabold tracking-tight">
            {FEATURE_PAGE.problemTitle}
          </h2>
          <p className="text-base leading-relaxed text-muted-foreground">{feature.problem}</p>
        </div>
      </section>

      <section aria-labelledby="feature-how" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto max-w-4xl px-4 py-14">
          <h2 id="feature-how" className="font-heading text-2xl font-extrabold tracking-tight">
            {FEATURE_PAGE.howTitle}
          </h2>
          <ol className="mt-6 flex flex-col gap-4">
            {feature.howItWorks.map((step, index) => (
              <li key={step} className="flex items-start gap-4">
                <span className="grid size-8 shrink-0 place-items-center rounded-md border-2 border-foreground bg-accent-acid font-mono text-sm font-bold text-accent-acid-foreground">
                  {index + 1}
                </span>
                <p className="pt-1 text-sm leading-relaxed text-muted-foreground">{step}</p>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section aria-labelledby="feature-detail" className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto flex max-w-4xl flex-col gap-4 px-4 py-14">
          <h2 id="feature-detail" className="font-heading text-2xl font-extrabold tracking-tight">
            {FEATURE_PAGE.detailTitle}
          </h2>
          {feature.detail.map((paragraph) => (
            <p key={paragraph} className="text-base leading-relaxed text-muted-foreground">
              {paragraph}
            </p>
          ))}
        </div>
      </section>

      <section aria-labelledby="feature-benefits" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto max-w-4xl px-4 py-14">
          <h2 id="feature-benefits" className="font-heading text-2xl font-extrabold tracking-tight">
            {FEATURE_PAGE.benefitsTitle}
          </h2>
          <Reveal>
            <ul className="mt-6 grid gap-4 sm:grid-cols-2">
              {feature.benefits.map((benefit) => (
                <li key={benefit} className="neo-card-sm flex items-start gap-3 p-5">
                  <Check aria-hidden className="mt-0.5 size-5 shrink-0 text-foreground" />
                  <p className="text-sm leading-relaxed">{benefit}</p>
                </li>
              ))}
            </ul>
          </Reveal>
        </div>
      </section>

      <section aria-labelledby="feature-example" className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto max-w-3xl px-4 py-14">
          <h2 id="feature-example" className="font-heading text-2xl font-extrabold tracking-tight">
            {FEATURE_PAGE.exampleTitle}
          </h2>
          <figure className="neo-card mt-6 p-6">
            <p className="font-mono text-sm text-muted-foreground">
              <span className="font-bold text-foreground">Anda: </span>
              {feature.example.prompt}
            </p>
            <p className="mt-4 font-mono text-sm">
              <span className="font-bold">FinTrack: </span>
              {feature.example.result}
            </p>
            <figcaption className="mt-6 border-t-2 border-foreground/20 pt-4 text-xs font-bold uppercase tracking-[0.08em] text-muted-foreground">
              {feature.example.caption ?? "Contoh tampilan"}
            </figcaption>
          </figure>
        </div>
      </section>

      <section aria-labelledby="feature-use-cases" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto max-w-4xl px-4 py-14">
          <h2 id="feature-use-cases" className="font-heading text-2xl font-extrabold tracking-tight">
            {FEATURE_PAGE.useCasesTitle}
          </h2>
          <ul className="mt-6 grid gap-4 sm:grid-cols-2">
            {feature.useCases.map((useCase) => (
              <li key={useCase.title} className="neo-card-sm flex flex-col gap-2 p-5">
                <h3 className="font-heading text-base font-extrabold tracking-tight">
                  {useCase.title}
                </h3>
                <p className="text-sm leading-relaxed text-muted-foreground">{useCase.text}</p>
              </li>
            ))}
          </ul>
        </div>
      </section>

      <section aria-labelledby="feature-notes" className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto max-w-4xl px-4 py-14">
          <h2 id="feature-notes" className="font-heading text-2xl font-extrabold tracking-tight">
            {FEATURE_PAGE.notesTitle}
          </h2>
          <ul className="mt-6 flex flex-col gap-3">
            {feature.notes.map((note) => (
              <li key={note} className="flex items-start gap-3">
                <Info aria-hidden className="mt-0.5 size-5 shrink-0 text-muted-foreground" />
                <p className="text-sm leading-relaxed text-muted-foreground">{note}</p>
              </li>
            ))}
          </ul>

          <p className="mt-8 inline-flex flex-wrap items-center gap-2 rounded-md border-[3px] border-foreground bg-background px-4 py-2 text-sm shadow-[3px_3px_0px_0px] shadow-foreground">
            <span className="font-bold">{FEATURE_PAGE.availabilityLabel}:</span>
            <span>{availability}</span>
          </p>
          <p className="mt-3 max-w-2xl text-xs leading-relaxed text-muted-foreground">
            {FEATURE_PAGE.availabilityNote}
          </p>
        </div>
      </section>

      <section aria-labelledby="feature-related" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto max-w-4xl px-4 py-14">
          <h2 id="feature-related" className="font-heading text-2xl font-extrabold tracking-tight">
            {FEATURE_PAGE.relatedTitle}
          </h2>
          <ul className="mt-6 grid gap-4 sm:grid-cols-2">
            {related.map((item) => {
              const RelatedIcon = item.icon;
              return (
                <li key={item.slug}>
                  <Link
                    href={featureHref(item.slug)}
                    className="neo-card-sm flex h-full flex-col gap-3 p-5 transition-transform hover:-translate-y-0.5"
                  >
                    <span className="flex items-center gap-3">
                      <RelatedIcon aria-hidden className="size-5" />
                      <span className="font-heading text-base font-extrabold tracking-tight">
                        {item.title}
                      </span>
                    </span>
                    <span className="text-sm leading-relaxed text-muted-foreground">
                      {item.tagline}
                    </span>
                  </Link>
                </li>
              );
            })}
          </ul>
          <p className="mt-8">
            <Link
              href={FEATURE_PAGE.allFeatures.href}
              className="inline-flex items-center gap-1.5 font-bold underline decoration-accent-acid decoration-[3px] underline-offset-4 hover:decoration-foreground"
            >
              {FEATURE_PAGE.allFeatures.label}
              <ArrowRight aria-hidden className="size-4" />
            </Link>
          </p>
        </div>
      </section>

      <section aria-labelledby="feature-closing" className="neo-dark border-b-[3px] border-foreground">
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 text-center">
            <h2
              id="feature-closing"
              className="font-heading text-[clamp(1.875rem,4vw,3rem)] font-extrabold leading-[1.1] tracking-[-0.02em] text-white text-balance"
            >
              {FEATURE_PAGE.closing.h2}
            </h2>
            <p className="max-w-2xl text-lg leading-relaxed text-neutral-200">
              {FEATURE_PAGE.closing.subhead}
            </p>
            <ActionLink href={FEATURE_PAGE.primaryCta.href} tone="acid" size="lg">
              {FEATURE_PAGE.primaryCta.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
          </div>
        </div>
      </section>
    </>
  );
}
