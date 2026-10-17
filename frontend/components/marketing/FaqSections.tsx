import { ArrowRight } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { Breadcrumbs } from "./Breadcrumbs";
import { FaqAccordion } from "./FaqAccordion";
import { SectionHeading } from "./SectionHeading";
import { FAQ_PAGE } from "@/lib/marketing/copy";
import { FAQ_CATEGORIES } from "@/lib/marketing/faq";

/**
 * `/faq` body (spec doc 05 §3). Four grouped categories, each rendered by the
 * shared native-`<details>` accordion so the answers exist in the server HTML
 * (good for readers and for AI citation). No `FAQPage` structured data is
 * emitted — the answers are plain, quotable text.
 */
export function FaqSections() {
  return (
    <>
      <section className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-12 lg:py-16">
          <Breadcrumbs
            items={[
              { name: "Beranda", path: "/" },
              { name: "FAQ", path: "/faq" },
            ]}
          />
          <SectionHeading
            as="h1"
            eyebrow="FAQ"
            title={FAQ_PAGE.h1}
            description={FAQ_PAGE.subhead}
          />
        </div>
      </section>

      <section aria-labelledby="faq-toc-heading" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto max-w-4xl px-4 py-10">
          <h2 id="faq-toc-heading" className="font-mono text-xs font-bold uppercase tracking-[0.08em] text-muted-foreground">
            {FAQ_PAGE.tocTitle}
          </h2>
          <ul className="mt-4 flex flex-wrap gap-3">
            {FAQ_CATEGORIES.map((category) => (
              <li key={category.id}>
                <a
                  href={`#${category.id}`}
                  className="inline-flex items-center gap-1.5 rounded-md border-[3px] border-foreground bg-background px-4 py-2 text-sm font-bold shadow-[3px_3px_0px_0px] shadow-foreground transition-transform hover:-translate-y-0.5"
                >
                  {category.title}
                </a>
              </li>
            ))}
          </ul>
        </div>
      </section>

      {FAQ_CATEGORIES.map((category) => (
        <FaqAccordion key={category.id} id={category.id} heading={category.title} items={category.items} />
      ))}

      <section aria-labelledby="faq-still" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-3xl flex-col items-center gap-4 px-4 py-14 text-center">
          <h2 id="faq-still" className="font-heading text-2xl font-extrabold tracking-tight">
            {FAQ_PAGE.stillTitle}
          </h2>
          <p className="text-base leading-relaxed text-muted-foreground">{FAQ_PAGE.stillText}</p>
          <ActionLink href={FAQ_PAGE.contactCta.href} tone="outline">
            {FAQ_PAGE.contactCta.label}
            <ArrowRight aria-hidden className="size-4" />
          </ActionLink>
        </div>
      </section>

      <section aria-labelledby="faq-closing" className="neo-dark border-b-[3px] border-foreground">
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 text-center">
            <h2
              id="faq-closing"
              className="font-heading text-[clamp(1.875rem,4vw,3rem)] font-extrabold leading-[1.1] tracking-[-0.02em] text-white text-balance"
            >
              {FAQ_PAGE.closing.h2}
            </h2>
            <ActionLink href="/register" tone="acid" size="lg">
              Daftar Gratis
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
          </div>
        </div>
      </section>
    </>
  );
}
