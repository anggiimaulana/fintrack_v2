import { Check } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { SectionHeading } from "./SectionHeading";
import { LANDING } from "@/lib/marketing/copy";
import { formatRupiah } from "@/lib/format";
import { isFreePlan, orderPlans, planHighlights } from "@/lib/marketing/plans";
import type { PublicPlan } from "@/lib/plans-types";

/**
 * Pricing teaser (spec doc 03 §B.4.9). Renders three summary cards from the
 * live catalog; when the catalog is unavailable it degrades to a single link to
 * `/pricing` instead of showing empty or stale prices.
 */
export function PricingTeaser({ plans }: { plans: PublicPlan[] }) {
  const { eyebrow, h2, subhead, cta, emptyFallback, popularBadge } = LANDING.pricingTeaser;
  const ordered = orderPlans(plans);

  return (
    <section id="pricing" className="border-b-[3px] border-foreground bg-muted">
      <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
        <SectionHeading
          eyebrow={eyebrow}
          title={h2}
          description={subhead}
          align="center"
          className="mx-auto mb-12 max-w-3xl"
        />

        {ordered.length === 0 ? (
          <div className="flex justify-center">
            <ActionLink href={emptyFallback.href} tone="outline" size="lg">
              {emptyFallback.label}
            </ActionLink>
          </div>
        ) : (
          <ul className="grid gap-6 md:grid-cols-3">
            {ordered.map((plan) => {
              const free = isFreePlan(plan);
              const popular = plan.code?.toLowerCase() === "pro";
              return (
                <li key={plan.code} className="neo-card-lg relative flex flex-col gap-4 p-6">
                  {popular ? (
                    <span className="neo-badge absolute -top-3 left-6 bg-accent-acid text-accent-acid-foreground">
                      {popularBadge}
                    </span>
                  ) : null}
                  <h3 className="font-heading text-lg font-extrabold tracking-tight">{plan.name}</h3>
                  <p className="flex items-baseline gap-1">
                    <span className="font-heading text-3xl font-black">
                      {free ? "Gratis" : formatRupiah(plan.price_monthly_idr)}
                    </span>
                    {!free ? <span className="text-sm text-muted-foreground">/ bulan</span> : null}
                  </p>
                  <ul className="flex flex-col gap-2 text-sm">
                    {planHighlights(plan)
                      .slice(0, 3)
                      .map((highlight) => (
                        <li key={highlight} className="flex items-start gap-2">
                          <Check aria-hidden className="mt-0.5 size-4 shrink-0" />
                          <span>{highlight}</span>
                        </li>
                      ))}
                  </ul>
                  <ActionLink href={cta.href} tone="outline" className="mt-auto w-full">
                    {cta.label}
                  </ActionLink>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}
