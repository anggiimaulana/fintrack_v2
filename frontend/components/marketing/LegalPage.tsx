import { Breadcrumbs } from "./Breadcrumbs";
import { JsonLd } from "./JsonLd";
import { SectionHeading } from "./SectionHeading";
import type { LegalDocument } from "@/lib/marketing/legal";
import { webPageJsonLd } from "@/lib/seo";

/** Render an ISO date as an Indonesian long date, in UTC. */
function formatEffectiveDate(iso: string): string {
  const date = new Date(`${iso}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  }).format(date);
}

/**
 * Shared renderer for `/privacy` and `/terms` (spec doc 05 §6): breadcrumb, H1,
 * a plain-language summary, the effective date, an on-page table of contents for
 * large screens, and one `H2` per section. Static — no data fetching.
 */
export function LegalPage({ doc }: { doc: LegalDocument }) {
  return (
    <>
      <JsonLd
        id={`${doc.id}-webpage-jsonld`}
        data={webPageJsonLd(`${doc.title} — FinTrack`, `/${doc.id}`, doc.summary)}
      />

      <section className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-12 lg:py-16">
          <Breadcrumbs
            items={[
              { name: "Beranda", path: "/" },
              { name: doc.title, path: `/${doc.id}` },
            ]}
          />
          <SectionHeading as="h1" eyebrow="LEGAL" title={doc.title} description={doc.summary} />
          <p className="font-mono text-sm font-bold">
            Berlaku sejak: <time dateTime={doc.effectiveDate}>{formatEffectiveDate(doc.effectiveDate)}</time>
          </p>
          {doc.draft ? (
            <p className="rounded-md border-[3px] border-foreground bg-accent-acid px-4 py-3 text-sm font-bold text-accent-acid-foreground">
              Catatan: dokumen ini masih berstatus draf internal dan belum ditinjau oleh penasihat
              hukum. Isinya dapat berubah.
            </p>
          ) : null}
        </div>
      </section>

      <section className="border-b-[3px] border-foreground bg-background">
        <div className="mx-auto grid max-w-4xl gap-10 px-4 py-14 lg:grid-cols-[220px_1fr] lg:gap-12">
          <nav aria-labelledby="legal-toc-heading" className="hidden lg:block">
            <div className="sticky top-24">
              <h2
                id="legal-toc-heading"
                className="font-mono text-xs font-bold uppercase tracking-[0.08em] text-muted-foreground"
              >
                Isi
              </h2>
              <ol className="mt-4 flex flex-col gap-2 text-sm">
                {doc.sections.map((section, index) => (
                  <li key={section.id}>
                    <a
                      href={`#${section.id}`}
                      className="text-muted-foreground hover:text-foreground hover:underline"
                    >
                      {index + 1}. {section.heading}
                    </a>
                  </li>
                ))}
              </ol>
            </div>
          </nav>

          <div className="flex flex-col gap-10">
            {doc.sections.map((section) => (
              <article key={section.id} id={section.id} className="flex flex-col gap-3">
                <h2 className="font-heading text-xl font-extrabold tracking-tight">
                  {section.heading}
                </h2>
                {section.paragraphs?.map((paragraph) => (
                  <p key={paragraph} className="text-sm leading-relaxed text-muted-foreground">
                    {paragraph}
                  </p>
                ))}
                {section.list?.length ? (
                  <ul className="flex list-disc flex-col gap-2 pl-5 text-sm leading-relaxed text-muted-foreground">
                    {section.list.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                ) : null}
              </article>
            ))}
          </div>
        </div>
      </section>
    </>
  );
}
