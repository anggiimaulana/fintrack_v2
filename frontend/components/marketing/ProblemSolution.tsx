import { Check, X } from "lucide-react";
import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { LANDING } from "@/lib/marketing/copy";

/**
 * Problem → solution contrast (spec doc 03 §B.4.4). Red communicates the pain,
 * the acid accent marks the fix; both sides state the same three situations so
 * the comparison reads cleanly.
 */
export function ProblemSolution() {
  const { h2, problemTitle, solutionTitle, problems, solutions } = LANDING.problemSolution;

  return (
    <section id="solution" className="border-b-[3px] border-foreground bg-muted">
      <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
        <SectionHeading title={h2} align="center" className="mx-auto mb-12 max-w-3xl" />

        <Reveal>
          <div className="grid gap-6 lg:grid-cols-2">
            <div className="neo-card-lg border-destructive/50 bg-destructive/10 p-6 sm:p-8">
              <h3 className="font-heading text-xl font-extrabold uppercase tracking-wide text-destructive">
                {problemTitle}
              </h3>
              <ul className="mt-5 flex flex-col gap-4">
                {problems.map((problem) => (
                  <li key={problem} className="flex items-start gap-3 text-sm leading-relaxed sm:text-base">
                    <X aria-hidden className="mt-0.5 size-5 shrink-0 text-destructive" />
                    <span>{problem}</span>
                  </li>
                ))}
              </ul>
            </div>

            <div className="neo-card-lg p-6 sm:p-8">
              <h3 className="font-heading text-xl font-extrabold uppercase tracking-wide">
                <span className="neo-highlight">{solutionTitle}</span>
              </h3>
              <ul className="mt-5 flex flex-col gap-4">
                {solutions.map((solution) => (
                  <li key={solution} className="flex items-start gap-3 text-sm leading-relaxed sm:text-base">
                    <Check aria-hidden className="mt-0.5 size-5 shrink-0" />
                    <span>{solution}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </Reveal>
      </div>
    </section>
  );
}
