"use client";

import * as React from "react";
import { Check } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { PRICING } from "@/lib/marketing/pricing";
import { formatRupiah } from "@/lib/format";
import {
  isFreePlan,
  orderPlans,
  planHighlights,
  taxBreakdown,
  yearlyMonthlyEquivalent,
  type BillingCycle,
} from "@/lib/marketing/plans";
import type { PublicPlan } from "@/lib/plans-types";
import { cn } from "@/lib/utils";

/**
 * Pricing cards with a billing-cycle toggle (spec doc 04 §A.4).
 *
 * The toggle only changes what is displayed — no transaction happens here. The
 * price is the **subtotal**; when the catalog enables tax it is shown as a
 * separate line with the resulting total. `-1` limits never appear on the card
 * (limits are handled by `ComparisonTable`).
 *
 * Accessibility: the toggle is a `role="group"` of two `aria-pressed` buttons,
 * and each price block is an `aria-live="polite"` region so a screen reader
 * announces the change when the cycle flips.
 */
export function PricingCards({ plans }: { plans: PublicPlan[] }) {
  const [cycle, setCycle] = React.useState<BillingCycle>("monthly");
  const ordered = orderPlans(plans);
  const cycles: Array<{ value: BillingCycle; label: string }> = [
    { value: "monthly", label: PRICING.cycle.monthly },
    { value: "yearly", label: PRICING.cycle.yearly },
  ];

  return (
    <div className="flex flex-col gap-10">
      <div className="flex justify-center">
        <div
          role="group"
          aria-label={PRICING.cycle.groupLabel}
          className="inline-flex rounded-md border-[3px] border-foreground bg-background p-1 shadow-[4px_4px_0px_0px] shadow-foreground"
        >
          {cycles.map((option) => {
            const selected = cycle === option.value;
            return (
              <button
                key={option.value}
                type="button"
                aria-pressed={selected}
                onClick={() => setCycle(option.value)}
                className={cn(
                  "rounded-sm px-5 py-2 text-sm font-bold transition-colors",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                  selected
                    ? "bg-accent-acid text-accent-acid-foreground"
                    : "bg-transparent text-foreground hover:bg-muted",
                )}
              >
                {option.label}
              </button>
            );
          })}
        </div>
      </div>

      <ul className="grid gap-6 lg:grid-cols-3">
        {ordered.map((plan) => {
          const free = isFreePlan(plan);
          const popular = plan.code?.toLowerCase() === "pro";
          const tax = taxBreakdown(plan, cycle);
          const yearly = cycle === "yearly" && !free;

          return (
            <li
              key={plan.code}
              className={cn(
                "neo-card-lg relative flex flex-col gap-5 p-6",
                popular && "lg:-mt-4 lg:mb-4",
              )}
            >
              {popular ? (
                <span className="neo-badge absolute -top-3 left-6 bg-accent-acid text-accent-acid-foreground">
                  {PRICING.card.popularBadge}
                </span>
              ) : null}

              <div className="flex flex-col gap-1">
                <h3 className="font-heading text-xl font-extrabold tracking-tight">{plan.name}</h3>
                {plan.description ? (
                  <p className="text-sm leading-relaxed text-muted-foreground">{plan.description}</p>
                ) : null}
              </div>

              <div aria-live="polite" className="flex flex-col gap-1">
                <p className="flex items-baseline gap-1">
                  <span className="font-heading text-4xl font-black">
                    {free
                      ? PRICING.card.freePriceLabel
                      : formatRupiah(yearly ? plan.price_yearly_idr : plan.price_monthly_idr)}
                  </span>
                  {!free ? (
                    <span className="text-sm text-muted-foreground">
                      {yearly ? PRICING.cycle.perYear : PRICING.cycle.perMonth}
                    </span>
                  ) : null}
                </p>
                {yearly ? (
                  <p className="text-sm text-muted-foreground">
                    {PRICING.cycle.yearlyEquivalentPrefix}{" "}
                    <span className="font-semibold text-foreground">
                      {formatRupiah(yearlyMonthlyEquivalent(plan))}
                    </span>{" "}
                    {PRICING.cycle.yearlyEquivalentSuffix}
                  </p>
                ) : null}
                {tax ? (
                  <p className="text-xs text-muted-foreground">
                    {tax.label} {tax.ratePercent}%: {formatRupiah(tax.taxIdr)} &middot; Total:{" "}
                    {formatRupiah(tax.totalIdr)}
                  </p>
                ) : null}
              </div>

              <ul className="flex flex-col gap-2 text-sm">
                {planHighlights(plan).map((highlight) => (
                  <li key={highlight} className="flex items-start gap-2">
                    <Check aria-hidden className="mt-0.5 size-4 shrink-0" />
                    <span>{highlight}</span>
                  </li>
                ))}
              </ul>

              <div className="mt-auto flex flex-col gap-3">
                <ActionLink
                  href={free ? PRICING.cta.cta.href : `/register?plan=${plan.code}`}
                  tone={popular ? "acid" : "outline"}
                  className="w-full"
                >
                  {free ? PRICING.card.freeCta : `${PRICING.card.paidCtaPrefix} ${plan.name}`}
                </ActionLink>
                {!free ? (
                  <p className="text-center text-xs text-muted-foreground">{PRICING.card.taxNote}</p>
                ) : null}
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
