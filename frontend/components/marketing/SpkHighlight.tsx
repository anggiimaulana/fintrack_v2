import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { LANDING } from "@/lib/marketing/copy";

/**
 * Financial-health highlight (spec doc 03 §B.4.7). The five statuses come from
 * the backend SPK metric configuration, and every status carries a text label so
 * meaning never depends on colour alone.
 */

const STATUS_COLORS = ["bg-chart-3", "bg-primary", "bg-chart-4", "bg-chart-2", "bg-destructive"];

const RADIUS = 46;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

function ScoreRing({ score }: { score: number }) {
  const offset = CIRCUMFERENCE * (1 - Math.min(100, Math.max(0, score)) / 100);
  return (
    <div className="relative grid size-28 shrink-0 place-items-center">
      <svg aria-hidden viewBox="0 0 100 100" className="absolute inset-0 size-full -rotate-90 text-chart-3">
        <circle cx="50" cy="50" r={RADIUS} fill="none" strokeWidth="8" className="stroke-muted" />
        <circle
          cx="50"
          cy="50"
          r={RADIUS}
          fill="none"
          strokeWidth="8"
          strokeLinecap="round"
          strokeDasharray={CIRCUMFERENCE}
          strokeDashoffset={offset}
          className="stroke-current"
        />
      </svg>
      <span className="font-heading text-3xl font-black">{score}</span>
    </div>
  );
}

export function SpkHighlight() {
  const { eyebrow, h2, description, statuses, example } = LANDING.spk;

  return (
    <section id="insights" className="scroll-mt-24 border-b-[3px] border-foreground bg-background">
      <div className="mx-auto grid max-w-7xl items-center gap-12 px-4 py-16 lg:grid-cols-2 lg:py-24">
        <div className="flex flex-col gap-6">
          <SectionHeading eyebrow={eyebrow} title={h2} description={description} />
          <ul className="flex flex-wrap gap-3">
            {statuses.map((status, index) => (
              <li
                key={status}
                className="inline-flex items-center gap-2 rounded-md border-2 border-foreground bg-card px-3 py-1.5 text-sm font-bold"
              >
                <span aria-hidden className={`size-3 rounded-sm border border-foreground ${STATUS_COLORS[index] ?? "bg-muted"}`} />
                {status}
              </li>
            ))}
          </ul>
        </div>

        <Reveal>
          <div className="neo-card-lg flex items-center gap-6 bg-muted p-6 sm:p-8">
            <div className="neo-card-sm flex w-full items-center gap-6 p-6">
              <ScoreRing score={example.score} />
              <div className="flex flex-col gap-1">
                <h3 className="font-heading text-xl font-extrabold tracking-tight">{example.status}</h3>
                <p className="text-sm leading-relaxed text-muted-foreground">{example.note}</p>
              </div>
            </div>
          </div>
        </Reveal>
      </div>
    </section>
  );
}
