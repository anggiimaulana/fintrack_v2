import Link from "next/link";
import { ChevronRight } from "lucide-react";
import { JsonLd } from "./JsonLd";
import { breadcrumbJsonLd, type BreadcrumbItem } from "@/lib/seo";

/**
 * Visible breadcrumb trail plus matching `BreadcrumbList` JSON-LD. The caller
 * passes the full trail (including the home crumb). Server component.
 */
export function Breadcrumbs({ items }: { items: BreadcrumbItem[] }) {
  return (
    <>
      <nav aria-label="Breadcrumb" className="w-full">
        <ol className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
          {items.map((item, index) => {
            const isLast = index === items.length - 1;
            return (
              <li key={item.path} className="flex items-center gap-1.5">
                {isLast ? (
                  <span aria-current="page" className="font-semibold text-foreground">
                    {item.name}
                  </span>
                ) : (
                  <Link href={item.path} className="hover:text-foreground hover:underline">
                    {item.name}
                  </Link>
                )}
                {!isLast ? <ChevronRight aria-hidden className="size-3.5" /> : null}
              </li>
            );
          })}
        </ol>
      </nav>
      <JsonLd id="breadcrumbs-jsonld" data={breadcrumbJsonLd(items)} />
    </>
  );
}
