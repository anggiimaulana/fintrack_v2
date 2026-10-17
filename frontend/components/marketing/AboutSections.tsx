import Link from "next/link";
import { ArrowRight, Check } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { Breadcrumbs } from "./Breadcrumbs";
import { Reveal } from "./Reveal";
import { SectionHeading } from "./SectionHeading";
import { ABOUT } from "@/lib/marketing/support";

/**
 * `/about` body (spec doc 05 §1.2): hero + mission, an honest technical
 * overview, the four product principles, the audience pointer, and contact/CTA.
 * Every claim is one the product can actually back up.
 */
export function AboutSections() {
  return (
    <>
      <section className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-12 lg:py-16">
          <Breadcrumbs
            items={[
              { name: "Beranda", path: "/" },
              { name: "Tentang", path: "/about" },
            ]}
          />
          <SectionHeading as="h1" eyebrow="TENTANG" title={ABOUT.h1} description={ABOUT.intro} />
        </div>
      </section>

      <section aria-labelledby="about-mission" className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto grid max-w-4xl gap-3 px-4 py-14 lg:grid-cols-[auto_1fr] lg:gap-8">
          <h2 id="about-mission" className="font-heading text-2xl font-extrabold tracking-tight">
            {ABOUT.mission.title}
          </h2>
          <p className="font-heading text-xl font-bold leading-relaxed tracking-[-0.01em]">
            {ABOUT.mission.text}
          </p>
        </div>
      </section>

      <section aria-labelledby="about-tech" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-4xl flex-col gap-4 px-4 py-14">
          <h2 id="about-tech" className="font-heading text-2xl font-extrabold tracking-tight">
            {ABOUT.tech.title}
          </h2>
          {ABOUT.tech.paragraphs.map((paragraph) => (
            <p key={paragraph} className="text-base leading-relaxed text-muted-foreground">
              {paragraph}
            </p>
          ))}
        </div>
      </section>

      <section
        aria-labelledby="about-principles"
        className="border-b-[3px] border-foreground bg-muted"
      >
        <div className="mx-auto max-w-4xl px-4 py-14">
          <h2 id="about-principles" className="font-heading text-2xl font-extrabold tracking-tight">
            {ABOUT.principles.title}
          </h2>
          <Reveal>
            <ul className="mt-6 grid gap-4 sm:grid-cols-2">
              {ABOUT.principles.items.map((principle) => (
                <li key={principle.title} className="neo-card-sm flex flex-col gap-2 p-5">
                  <span className="flex items-center gap-2">
                    <Check aria-hidden className="size-5" />
                    <h3 className="font-heading text-base font-extrabold tracking-tight">
                      {principle.title}
                    </h3>
                  </span>
                  <p className="text-sm leading-relaxed text-muted-foreground">{principle.text}</p>
                </li>
              ))}
            </ul>
          </Reveal>
        </div>
      </section>

      <section aria-labelledby="about-audience" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto grid max-w-4xl gap-4 px-4 py-14 lg:grid-cols-[auto_1fr] lg:gap-8">
          <h2 id="about-audience" className="font-heading text-2xl font-extrabold tracking-tight">
            {ABOUT.audience.title}
          </h2>
          <div className="flex flex-col gap-4">
            <p className="text-base leading-relaxed text-muted-foreground">{ABOUT.audience.text}</p>
            <Link
              href={ABOUT.audience.link.href}
              className="inline-flex items-center gap-1.5 self-start font-bold underline decoration-accent-acid decoration-[3px] underline-offset-4 hover:decoration-foreground"
            >
              {ABOUT.audience.link.label}
              <ArrowRight aria-hidden className="size-4" />
            </Link>
          </div>
        </div>
      </section>

      <section aria-labelledby="about-contact" className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto grid max-w-4xl gap-4 px-4 py-14 lg:grid-cols-[auto_1fr] lg:gap-8">
          <h2 id="about-contact" className="font-heading text-2xl font-extrabold tracking-tight">
            {ABOUT.contact.title}
          </h2>
          <div className="flex flex-col gap-4">
            <p className="text-base leading-relaxed text-muted-foreground">{ABOUT.contact.text}</p>
            <ActionLink href={ABOUT.contact.link.href} tone="outline" className="self-start">
              {ABOUT.contact.link.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
          </div>
        </div>
      </section>

      <section aria-labelledby="about-closing" className="neo-dark border-b-[3px] border-foreground">
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 text-center">
            <h2
              id="about-closing"
              className="font-heading text-[clamp(1.875rem,4vw,3rem)] font-extrabold leading-[1.1] tracking-[-0.02em] text-white text-balance"
            >
              Mulai rapikan keuangan Anda hari ini
            </h2>
            <ActionLink href={ABOUT.cta.href} tone="acid" size="lg">
              {ABOUT.cta.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
          </div>
        </div>
      </section>
    </>
  );
}
