import type { PublicPlan } from "@/lib/plans-types";
import { LANDING } from "@/lib/marketing/copy";
import { LANDING_FAQ } from "@/lib/marketing/faq";
import { Hero } from "./Hero";
import { ReceiptTapeMarquee } from "./ReceiptTapeMarquee";
import { TrustBar } from "./TrustBar";
import { ProblemSolution } from "./ProblemSolution";
import { FeatureGrid } from "./FeatureGrid";
import { HowItWorks } from "./HowItWorks";
import { SpkHighlight } from "./SpkHighlight";
import { PricingTeaser } from "./PricingTeaser";
import { PersonaUseCases } from "./PersonaUseCases";
import { FaqAccordion } from "./FaqAccordion";
import { CtaSection } from "./CtaSection";

/**
 * The landing page body, in the order fixed by spec doc 03 §B.3. Kept separate
 * from `app/(marketing)/page.tsx` so it can be rendered in tests without a
 * network call (the page fetches the catalog and passes `plans` down).
 *
 * `#pricing` and `#how-it-works` are real in-page anchors (closing F-01/F-03).
 */
export function LandingSections({ plans }: { plans: PublicPlan[] }) {
  return (
    <>
      <Hero />
      <ReceiptTapeMarquee />
      <TrustBar />
      <ProblemSolution />
      <FeatureGrid />
      <HowItWorks />
      <SpkHighlight />
      <PricingTeaser plans={plans} />
      <PersonaUseCases />
      <FaqAccordion heading={LANDING.faq.h2} items={LANDING_FAQ} footer={LANDING.faq.viewAll} />
      <CtaSection />
    </>
  );
}
