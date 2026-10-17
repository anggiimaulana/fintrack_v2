import { Download, Gauge, Lock, Trash2 } from "lucide-react";
import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { StatCounters } from "./StatCounters";
import { LANDING } from "@/lib/marketing/copy";

/**
 * Trust bar + stat counters (spec doc 03 §B.4.3). Every claim maps to a real
 * product capability (per-account isolation, data export, account deletion,
 * visible AI usage). The stats are the only three numbers we can stand behind.
 */

const ICONS = [Lock, Download, Trash2, Gauge];

export function TrustBar() {
  return (
    <section aria-labelledby="trust-heading" className="border-b-[3px] border-foreground bg-background">
      <div className="mx-auto max-w-7xl px-4 py-16 lg:py-20">
        <SectionHeading
          id="trust-heading"
          eyebrow="KEAMANAN & PRIVASI"
          title="Transparansi dan perlindungan data tanpa kompromi"
          description="Arsitektur data terisolasi, kuota AI transparan, dan kebebasan ekspor atau hapus akun kapan saja."
          className="mb-10 lg:mb-12"
        />

        <Reveal>
          <ul className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {LANDING.trustBar.map((point, index) => {
              const Icon = ICONS[index] ?? Lock;
              return (
                <li key={point.title} className="neo-card-sm flex flex-col gap-3 p-6">
                  <span className="grid size-10 place-items-center rounded-md border-2 border-foreground bg-accent-acid">
                    <Icon aria-hidden className="size-5 text-accent-acid-foreground" />
                  </span>
                  <h3 className="font-heading text-base font-extrabold tracking-tight">{point.title}</h3>
                  <p className="text-sm leading-relaxed text-muted-foreground">{point.text}</p>
                </li>
              );
            })}
          </ul>
        </Reveal>

        <Reveal className="mt-12">
          <StatCounters />
        </Reveal>
      </div>
    </section>
  );
}
