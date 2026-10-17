import Link from "next/link";
import { ChevronDown } from "lucide-react";
import { SectionHeading } from "./SectionHeading";
import type { FaqItem } from "@/lib/marketing/faq";

/**
 * FAQ accordion built on native `<details>`/`<summary>` so it works without
 * JavaScript and keeps browser search/anchor behaviour (spec doc 03 §B.8). Every
 * answer starts with the direct answer; optional links deep-link into the site.
 */
export interface FaqAccordionProps {
  id?: string;
  eyebrow?: string;
  heading: string;
  description?: string;
  items: FaqItem[];
  footer?: { label: string; href: string };
}

export function FaqAccordion({ id = "faq", eyebrow, heading, description, items, footer }: FaqAccordionProps) {
  return (
    <section id={id} className="scroll-mt-24 border-b-[3px] border-foreground bg-muted">
      <div className="mx-auto max-w-3xl px-4 py-16 lg:py-24">
        <SectionHeading eyebrow={eyebrow} title={heading} description={description} align="center" className="mb-12" />

        <div className="flex flex-col gap-3">
          {items.map((item) => (
            <details key={item.q} className="neo-card-sm group p-0">
              <summary className="flex cursor-pointer list-none items-center justify-between gap-4 p-5 font-heading text-base font-extrabold tracking-tight marker:content-none [&::-webkit-details-marker]:hidden">
                <span>{item.q}</span>
                <ChevronDown
                  aria-hidden
                  className="size-5 shrink-0 transition-transform group-open:rotate-180"
                />
              </summary>
              <div className="flex flex-col gap-3 border-t-2 border-foreground/20 p-5 pt-4 text-sm leading-relaxed text-muted-foreground">
                <p>{item.a}</p>
                {item.links?.length ? (
                  <ul className="flex flex-wrap gap-x-4 gap-y-1">
                    {item.links.map((link) => (
                      <li key={`${item.q}-${link.href}`}>
                        <Link
                          href={link.href}
                          className="font-bold text-foreground underline decoration-accent-acid decoration-[3px] underline-offset-4 hover:decoration-foreground"
                        >
                          {link.label}
                        </Link>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            </details>
          ))}
        </div>

        {footer ? (
          <p className="mt-8 text-center">
            <Link
              href={footer.href}
              className="inline-flex items-center gap-1.5 font-bold underline decoration-accent-acid decoration-[3px] underline-offset-4 hover:decoration-foreground"
            >
              {footer.label}
            </Link>
          </p>
        ) : null}
      </div>
    </section>
  );
}
