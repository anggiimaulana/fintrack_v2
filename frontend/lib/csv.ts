/**
 * Minimal client-side CSV export used by the data tables' "export selected"
 * action. Values are escaped per RFC 4180 (quotes doubled, cells wrapped when
 * they contain a comma, quote, or newline) and a UTF-8 BOM is prepended so
 * Excel on Windows reads Indonesian characters correctly.
 */

/** Escape one CSV cell. */
export function csvCell(value: unknown): string {
  if (value == null) return "";
  const s = String(value);
  if (/[",\n\r]/.test(s)) return `"${s.replace(/"/g, '""')}"`;
  return s;
}

/** Build a CSV document from a header row and data rows. */
export function toCsv(headers: string[], rows: (string | number | null | undefined)[][]): string {
  const lines = [headers.map(csvCell).join(",")];
  for (const row of rows) lines.push(row.map(csvCell).join(","));
  return lines.join("\r\n");
}

/**
 * Trigger a browser download of `content` as `filename`. Adds the BOM and uses
 * a temporary object URL that is revoked immediately after the click.
 */
export function downloadCsv(filename: string, content: string): void {
  const blob = new Blob(["\uFEFF" + content], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
