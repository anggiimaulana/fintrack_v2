import { describe, it, expect } from "vitest";
import { sanitizeAssistantContent } from "./chat-sanitize";

describe("sanitizeAssistantContent", () => {
  it("removes tool call/result tags", () => {
    const input = 'Sebelum\n<TOOL_CALL>\n{"tool":"generate_chart"}\n</TOOL_CALL>\nSesudah';
    const out = sanitizeAssistantContent(input);
    expect(out).toBe("Sebelum\n\nSesudah");
    expect(out).not.toContain("TOOL_CALL");
  });

  it("removes a partial tool tag with no closing tag", () => {
    const out = sanitizeAssistantContent('Halo\n<TOOL_CALL>\n{"tool":"x"');
    expect(out).toBe("Halo");
  });

  it("removes a ```json fenced payload (the screenshot case)", () => {
    const input = [
      "Berikut grafiknya:",
      "```json",
      '{ "chart": { "type": "bar", "labels": ["2026-06"] } }',
      "```",
      "→ skipped: advanced chart styling, add when UI framework requires theme integration.",
    ].join("\n");
    const out = sanitizeAssistantContent(input);
    expect(out).toBe("Berikut grafiknya:");
    expect(out).not.toContain("chart");
    expect(out).not.toContain("skipped");
  });

  it("removes a bare JSON fence without a language tag", () => {
    const out = sanitizeAssistantContent('Hasil:\n```\n{"datasets":[]}\n```');
    expect(out).toBe("Hasil:");
  });

  it("removes a standalone internal JSON object line", () => {
    const input = 'Data pengeluaran Anda:\n{"chart_type":"comparison","labels":["2026-06"]}';
    const out = sanitizeAssistantContent(input);
    expect(out).toBe("Data pengeluaran Anda:");
  });

  it("keeps ordinary prose, numbers and bullet lists intact", () => {
    const input = [
      "Pengeluaran Anda bulan ini **Rp 1.250.000**.",
      "",
      "1. Makan: Rp 500.000",
      "2. Transport: Rp 250.000",
    ].join("\n");
    expect(sanitizeAssistantContent(input)).toBe(input);
  });

  it("keeps a genuine (non-JSON) code fence intact", () => {
    const input = "Contoh:\n```bash\ngo build ./...\n```";
    expect(sanitizeAssistantContent(input)).toBe(input);
  });

  it("falls back to a friendly line when only a payload was sent", () => {
    const out = sanitizeAssistantContent('```json\n{"chart":{}}\n```');
    expect(out).not.toContain("chart");
    expect(out.length).toBeGreaterThan(0);
  });
});
