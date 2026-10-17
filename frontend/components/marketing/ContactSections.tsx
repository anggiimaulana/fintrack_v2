import Link from "next/link";
import { ArrowRight, Clock, Mail, MessageCircle } from "lucide-react";
import { ActionLink } from "./ActionLink";
import { Breadcrumbs } from "./Breadcrumbs";
import { SectionHeading } from "./SectionHeading";
import { CONTACT } from "@/lib/marketing/support";
import { mailtoHref, site, whatsappHref } from "@/lib/marketing/site";

/**
 * `/contact` body (spec doc 05 §2). Backend-free by design: the only actions are
 * a `mailto:` link and a `wa.me` link, both built from `site.ts` so the address
 * can never differ between pages. External links get descriptive `aria-label`s
 * and `rel="noopener noreferrer"`.
 */
export function ContactSections() {
  const { channels, hours, before, account } = CONTACT;

  return (
    <>
      <section className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-12 lg:py-16">
          <Breadcrumbs
            items={[
              { name: "Beranda", path: "/" },
              { name: "Kontak", path: "/contact" },
            ]}
          />
          <SectionHeading
            as="h1"
            eyebrow="KONTAK"
            title={CONTACT.h1}
            description={CONTACT.intro}
          />
        </div>
      </section>

      <section aria-labelledby="contact-channels" className="border-b-[3px] border-foreground bg-muted">
        <div className="mx-auto max-w-4xl px-4 py-14">
          <h2
            id="contact-channels"
            className="font-heading text-2xl font-extrabold tracking-tight"
          >
            Kanal dukungan
          </h2>
          <ul className="mt-6 grid gap-6 sm:grid-cols-2">
            <li className="neo-card-sm flex flex-col gap-4 p-6">
              <span className="grid size-12 place-items-center rounded-md border-2 border-foreground bg-accent-acid">
                <Mail aria-hidden className="size-6 text-accent-acid-foreground" />
              </span>
              <h3 className="font-heading text-lg font-extrabold tracking-tight">
                {channels.email.title}
              </h3>
              <p className="text-sm leading-relaxed text-muted-foreground">{channels.email.text}</p>
              <p className="font-mono text-sm font-bold">{site.email}</p>
              <ActionLink
                href={mailtoHref()}
                tone="acid"
                ariaLabel={channels.email.ariaLabel}
                className="self-start"
              >
                {channels.email.ctaLabel}
                <ArrowRight aria-hidden className="size-4" />
              </ActionLink>
            </li>

            <li className="neo-card-sm flex flex-col gap-4 p-6">
              <span className="grid size-12 place-items-center rounded-md border-2 border-foreground bg-accent-acid">
                <MessageCircle aria-hidden className="size-6 text-accent-acid-foreground" />
              </span>
              <h3 className="font-heading text-lg font-extrabold tracking-tight">
                {channels.whatsapp.title}
              </h3>
              <p className="text-sm leading-relaxed text-muted-foreground">
                {channels.whatsapp.text}
              </p>
              <p className="font-mono text-sm font-bold">{site.whatsappDisplay}</p>
              <ActionLink
                href={whatsappHref()}
                tone="outline"
                target="_blank"
                rel="noopener noreferrer"
                ariaLabel={channels.whatsapp.ariaLabel}
                className="self-start"
              >
                {channels.whatsapp.ctaLabel}
                <ArrowRight aria-hidden className="size-4" />
              </ActionLink>
            </li>
          </ul>

          <div className="mt-8 flex items-start gap-3 rounded-md border-[3px] border-foreground bg-background p-5 shadow-[4px_4px_0px_0px] shadow-foreground">
            <Clock aria-hidden className="mt-0.5 size-5 shrink-0" />
            <div className="flex flex-col gap-1">
              <h3 className="font-heading text-base font-extrabold tracking-tight">
                {hours.title}
              </h3>
              <p className="text-sm leading-relaxed text-muted-foreground">{hours.text}</p>
            </div>
          </div>
        </div>
      </section>

      <section aria-labelledby="contact-before" className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto max-w-4xl px-4 py-14">
          <h2 id="contact-before" className="font-heading text-2xl font-extrabold tracking-tight">
            {before.title}
          </h2>
          <p className="mt-3 text-base leading-relaxed text-muted-foreground">{before.text}</p>
          <ul className="mt-6 flex flex-wrap gap-3">
            {before.links.map((link) => (
              <li key={link.href}>
                <Link
                  href={link.href}
                  className="inline-flex items-center gap-1.5 rounded-md border-[3px] border-foreground bg-background px-4 py-2 text-sm font-bold shadow-[3px_3px_0px_0px] shadow-foreground transition-transform hover:-translate-y-0.5"
                >
                  {link.label}
                  <ArrowRight aria-hidden className="size-4" />
                </Link>
              </li>
            ))}
          </ul>

          <div className="mt-8 rounded-md border-[3px] border-foreground bg-muted p-5">
            <h3 className="font-heading text-base font-extrabold tracking-tight">
              {account.title}
            </h3>
            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{account.text}</p>
          </div>
        </div>
      </section>

      <section aria-labelledby="contact-closing" className="neo-dark border-b-[3px] border-foreground">
        <div className="mx-auto max-w-7xl px-4 py-16 lg:py-24">
          <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 text-center">
            <h2
              id="contact-closing"
              className="font-heading text-[clamp(1.875rem,4vw,3rem)] font-extrabold leading-[1.1] tracking-[-0.02em] text-white text-balance"
            >
              Mulai rapikan keuangan Anda hari ini
            </h2>
            <ActionLink href={CONTACT.cta.href} tone="acid" size="lg">
              {CONTACT.cta.label}
              <ArrowRight aria-hidden className="size-4" />
            </ActionLink>
          </div>
        </div>
      </section>
    </>
  );
}
