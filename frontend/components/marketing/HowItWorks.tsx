import { SectionHeading } from "./SectionHeading";
import { LANDING } from "@/lib/marketing/copy";

/**
 * "How it works" in three steps (spec doc 03 §B.4.6, F-03). No API-key step —
 * signing up is genuinely the first thing a user does.
 */
export function HowItWorks() {
  const { h2, steps } = LANDING.howItWorks;

  return (
    <section id="how-it-works" className="border-b-[3px] border-foreground bg-muted">
      <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
        <SectionHeading title={h2} align="center" className="mx-auto mb-12 max-w-3xl" />

        <ol className="grid gap-6 md:grid-cols-3">
          {steps.map((step, index) => (
            <li key={step.title} className="neo-card-lg flex flex-col gap-4 p-6 sm:p-8">
              <span className="grid size-12 place-items-center rounded-md border-[3px] border-foreground bg-accent-acid font-heading text-xl font-black text-accent-acid-foreground shadow-[3px_3px_0px_0px] shadow-foreground">
                {index + 1}
              </span>
              <h3 className="font-heading text-lg font-extrabold tracking-tight">{step.title}</h3>
              <p className="text-sm leading-relaxed text-muted-foreground">{step.text}</p>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
