import type { JsonLdNode } from "@/lib/seo";

/**
 * Renders JSON-LD structured data. Server component by design — structured data
 * must be present in the server HTML (not injected by client JS) for crawlers.
 *
 * `<` is escaped to `\u003c` so a stray `</script>` inside a string cannot break
 * out of the tag.
 */
export function JsonLd({ data, id }: { data: JsonLdNode | JsonLdNode[]; id?: string }) {
  return (
    <script
      id={id}
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data).replace(/</g, "\\u003c") }}
    />
  );
}
