import Link from "next/link";
import { Check, Minus } from "lucide-react";
import { SectionHeading } from "./SectionHeading";
import { PRICING } from "@/lib/marketing/pricing";
import { PLAN_FEATURE_ROWS, buildLimitRows, orderPlans, planHasFeature } from "@/lib/marketing/plans";
import type { PublicPlan } from "@/lib/plans-types";

/**
 * Feature + limit comparison tables (spec doc 04 §A.5–A.6). Server component:
 * both tables are built purely from the live catalog, so the labels, units, and
 * the set of shown limits always track the backend.
 *
 * Guards encoded here (not just documented):
 *  - Only limits the backend enforces appear (`buildLimitRows` drops
 *    `enforced: false`), so `max_categories`/`max_wallets` never show.
 *  - `bank_sync` is absent from `PLAN_FEATURE_ROWS` (the catalog marks it
 *    "belum tersedia"), so it can never be advertised.
 *  - `-1` renders as "Tanpa batas" via `formatLimitValue`.
 */

/** Check / strip cell with a descriptive accessible name. */
function AvailabilityCell({ available }: { available: boolean }) {
  return (
    <td className="px-4 py-3 text-center">
      {available ? (
        <span role="img" aria-label={PRICING.comparison.available} className="inline-flex">
          <Check aria-hidden className="size-5" />
        </span>
      ) : (
        <span role="img" aria-label={PRICING.comparison.unavailable} className="inline-flex text-muted-foreground">
          <Minus aria-hidden className="size-5" />
        </span>
      )}
    </td>
  );
}

export function ComparisonTable({ plans }: { plans: PublicPlan[] }) {
  const ordered = orderPlans(plans);
  const limitRows = buildLimitRows(ordered);

  return (
    <section id="perbandingan" className="scroll-mt-24 border-b-[3px] border-foreground">
      <div className="mx-auto max-w-5xl px-4 py-16 lg:py-24">
        <SectionHeading title={PRICING.comparison.h2} align="center" className="mb-12" />

        <div className="flex flex-col gap-12">
          <div className="flex flex-col gap-4">
            <h3 className="font-heading text-xl font-extrabold tracking-tight">
              {PRICING.comparison.featuresH3}
            </h3>
            <p className="text-sm text-muted-foreground">{PRICING.comparison.featuresSubhead}</p>

            <div className="overflow-x-auto">
              <table className="w-full min-w-[36rem] border-collapse text-sm">
                <caption className="sr-only">{PRICING.comparison.featuresH3}</caption>
                <thead>
                  <tr>
                    <th scope="col" className="border-b-[3px] border-foreground px-4 py-3 text-left">
                      {PRICING.comparison.featureColumn}
                    </th>
                    {ordered.map((plan) => (
                      <th
                        key={plan.code}
                        scope="col"
                        className="border-b-[3px] border-foreground px-4 py-3 text-center font-heading font-extrabold"
                      >
                        {plan.name}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {PLAN_FEATURE_ROWS.map((row) => (
                    <tr key={row.key} className="border-b-2 border-foreground/15">
                      <th scope="row" className="px-4 py-3 text-left font-semibold">
                        {row.href ? (
                          <Link
                            href={row.href}
                            className="underline decoration-accent-acid decoration-[3px] underline-offset-4 hover:decoration-foreground"
                          >
                            {row.label}
                          </Link>
                        ) : (
                          row.label
                        )}
                      </th>
                      {ordered.map((plan) => (
                        <AvailabilityCell key={plan.code} available={planHasFeature(plan, row.key)} />
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {limitRows.length > 0 ? (
            <div className="flex flex-col gap-4">
              <h3 className="font-heading text-xl font-extrabold tracking-tight">
                {PRICING.comparison.limitsH3}
              </h3>
              <p className="text-sm text-muted-foreground">{PRICING.comparison.limitsSubhead}</p>

              <div className="overflow-x-auto">
                <table className="w-full min-w-[36rem] border-collapse text-sm">
                  <caption className="sr-only">{PRICING.comparison.limitsH3}</caption>
                  <thead>
                    <tr>
                      <th scope="col" className="border-b-[3px] border-foreground px-4 py-3 text-left">
                        {PRICING.comparison.limitColumn}
                      </th>
                      {ordered.map((plan) => (
                        <th
                          key={plan.code}
                          scope="col"
                          className="border-b-[3px] border-foreground px-4 py-3 text-center font-heading font-extrabold"
                        >
                          {plan.name}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {limitRows.map((row) => (
                      <tr key={row.key} className="border-b-2 border-foreground/15">
                        <th scope="row" className="px-4 py-3 text-left font-semibold">
                          {row.label}
                        </th>
                        {row.values.map((value) => (
                          <td key={value.code} className="px-4 py-3 text-center">
                            {value.text}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </section>
  );
}
