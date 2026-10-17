import Link from "next/link";
import { Mail, MessageCircle, Wallet } from "lucide-react";
import { FOOTER_COLUMNS } from "@/lib/marketing/copy";
import { mailtoHref, site, whatsappHref } from "@/lib/marketing/site";

/**
 * Public marketing footer: dark block, brand + contact channels and the site
 * columns. Server component (no interactivity).
 */
export function MarketingFooter() {
  const year = new Date().getFullYear();

  return (
    <footer className="neo-dark mt-auto border-t-[3px] border-foreground">
      <div className="mx-auto grid max-w-7xl gap-10 px-4 py-12 md:grid-cols-[1.5fr_repeat(3,1fr)]">
        <div className="flex flex-col gap-4">
          <Link href="/" className="flex items-center gap-2" aria-label="FinTrack — beranda">
            <span className="grid size-8 place-items-center rounded-md border-2 border-black bg-accent-acid">
              <Wallet aria-hidden className="size-4 text-black" />
            </span>
            <span className="font-heading text-xl font-black tracking-tight text-white">
              {site.name}
            </span>
          </Link>
          <p className="max-w-xs text-sm text-neutral-300">{site.tagline}</p>
          <ul className="flex flex-col gap-2 text-sm">
            <li>
              <a
                href={mailtoHref()}
                className="inline-flex items-center gap-2 text-neutral-200 hover:text-white hover:underline"
              >
                <Mail aria-hidden className="size-4" />
                {site.email}
              </a>
            </li>
            <li>
              <a
                href={whatsappHref()}
                className="inline-flex items-center gap-2 text-neutral-200 hover:text-white hover:underline"
                rel="noopener noreferrer"
                target="_blank"
              >
                <MessageCircle aria-hidden className="size-4" />
                WhatsApp {site.whatsappDisplay}
              </a>
            </li>
          </ul>
          <p className="text-xs text-neutral-400">{site.supportHours}</p>
        </div>

        {FOOTER_COLUMNS.map((column) => (
          <nav key={column.title} aria-label={column.title} className="flex flex-col gap-3">
            <h2 className="font-mono text-xs font-bold uppercase tracking-[0.08em] text-neutral-400">
              {column.title}
            </h2>
            <ul className="flex flex-col gap-2 text-sm">
              {column.links.map((link) => (
                <li key={`${column.title}-${link.href}`}>
                  <Link href={link.href} className="text-neutral-200 hover:text-white hover:underline">
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        ))}
      </div>

      <div className="border-t-2 border-neutral-700">
        <div className="mx-auto flex max-w-7xl flex-col gap-2 px-4 py-6 text-xs text-neutral-400 sm:flex-row sm:items-center sm:justify-between">
          <p>
            © {year} {site.name}. Semua hak dilindungi.
          </p>
          <p>
            Dibuat untuk pengguna di Indonesia. Konten berbahasa Indonesia.
          </p>
        </div>
      </div>
    </footer>
  );
}
