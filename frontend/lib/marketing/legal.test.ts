import { describe, it, expect } from "vitest";
import { PRIVACY, TERMS, REQUIRED_SECTIONS, type LegalDocument } from "./legal";

const DOCS: LegalDocument[] = [PRIVACY, TERMS];

describe("legal documents", () => {
  DOCS.forEach((doc) => {
    describe(doc.id, () => {
      it("has a parseable effective date", () => {
        expect(Number.isNaN(new Date(doc.effectiveDate).getTime())).toBe(false);
      });

      it("is flagged as a draft until legally reviewed", () => {
        expect(doc.draft).toBe(true);
      });

      it("contains every required section", () => {
        const ids = doc.sections.map((s) => s.id);
        REQUIRED_SECTIONS[doc.id].forEach((required) => {
          expect(ids).toContain(required);
        });
      });

      it("gives every section a heading and body", () => {
        doc.sections.forEach((section) => {
          expect(section.heading.length).toBeGreaterThan(0);
          const hasBody = (section.paragraphs?.length ?? 0) > 0 || (section.list?.length ?? 0) > 0;
          expect(hasBody).toBe(true);
        });
      });

      it("avoids unverifiable or overreaching claims", () => {
        const text = JSON.stringify(doc).toLowerCase();
        ["terbaik", "nomor 1", "tercepat", "100%", "tersertifikasi", "iso 27001"].forEach((claim) => {
          expect(text).not.toContain(claim);
        });
        expect(text).not.toMatch(/\bkamu\b/);
      });
    });
  });
});
