import { ArrowRight } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { Breadcrumbs } from "./Breadcrumbs";
import { CatalogRetry } from "./CatalogRetry";
import { ComparisonTable } from "./ComparisonTable";
import { FaqAccordion } from "./FaqAccordion";
import { PricingCards } from "./PricingCards";
import { SectionHeading } from "./SectionHeading";
import { PRICING } from "@/lib/marketing/pricing";
import { PRICING_FAQ } from "@/lib/marketing/faq";
import type { PublicPlan } from "@/lib/plans-types";

/**
 * The `/pricing` page body (spec doc 04 §A.3), separated from the route so it
 * can be rendered in tests without a network call (the page fetches the catalog
 * and passes `plans` down). `#perbandingan` is the anchor the pricing FAQ links
 * to.
 *
 * On a failed/empty catalog the page still renders the hero, payment methods,
 * FAQ, and CTA — only the cards and comparison table degrade to a retry block.
 */
export function PricingSections({ plans }: { plans: PublicPlan[] }) {
  const hasPlans = plans.length > 0;

  return (
    <>
      <section className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto flex max-w-5xl flex-col gap-6 px-4 py-16 lg:py-20">
          <Breadcrumbs
            items={[
              { name: "Beranda", path: "/" },
              { name: "Harga", path: "/pricing" },
            ]}
          />
          <SectionHeading as="h1" title={PRICING.hero.h1} description={PRICING.hero.subhead} />
          <ul className="flex flex-wrap gap-x-6 gap-y-2 text-sm font-semibold">
            {PRICING.hero.chips.map((chip) => (
              <li key={chip} className="flex items-center gap-2">
                <span aria-hidden className="size-1.5 rounded-full bg-foreground" />
                {chip}
              </li>
            ))}
          </ul>
        </div>
      </section>

      <section className="border-b-[3px] border-foreground">
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          {hasPlans ? (
            <>
              <h2 className="sr-only">{PRICING.card.h2}</h2>
              <PricingCards plans={plans} />
            </>
          ) : (
            <CatalogRetry />
          )}
        </div>
      </section>

      {hasPlans ? <ComparisonTable plans={plans} /> : null}

      <section className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto max-w-5xl px-4 py-16 lg:py-24">
          <SectionHeading
            eyebrow={PRICING.payment.eyebrow}
            title={PRICING.payment.h2}
            description={PRICING.payment.subhead}
            className="mb-12"
          />
          <ul className="grid gap-6 md:grid-cols-2">
            {PRICING.payment.methods.map((method) => (
              <li key={method.title} className="neo-card-lg flex flex-col gap-2 p-6">
                <h3 className="font-heading text-lg font-extrabold tracking-tight">{method.title}</h3>
                <p className="text-sm leading-relaxed text-muted-foreground">{method.text}</p>
              </li>
            ))}
          </ul>
          <p className="mt-8 text-sm leading-relaxed text-muted-foreground">{PRICING.payment.taxNote}</p>
        </div>
      </section>

      <FaqAccordion
        id="faq-harga"
        eyebrow={PRICING.faq.eyebrow}
        heading={PRICING.faq.h2}
        description={PRICING.faq.description}
        items={PRICING_FAQ}
      />

      <section className="neo-dark border-b-[3px] border-foreground">
        <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 px-4 py-16 text-center lg:py-20">
          <h2 className="font-heading text-[clamp(1.875rem,4vw,3rem)] font-extrabold leading-[1.1] tracking-[-0.02em] text-white text-balance">
            {PRICING.cta.h2}
          </h2>
          <p className="max-w-xl text-base leading-relaxed text-neutral-200">{PRICING.cta.subhead}</p>
          <ActionLink href={PRICING.cta.cta.href} tone="acid" size="lg">
            {PRICING.cta.cta.label}
            <ArrowRight aria-hidden className="size-4" />
          </ActionLink>
        </div>
      </section>
    </>
  );
}
