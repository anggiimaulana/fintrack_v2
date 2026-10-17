import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LegalPage } from "./LegalPage";
import { PRIVACY, TERMS } from "@/lib/marketing/legal";

describe("LegalPage", () => {
  it("renders the H1, summary, and effective date for a document", () => {
    render(<LegalPage doc={PRIVACY} />);
    expect(screen.getByRole("heading", { level: 1, name: PRIVACY.title })).toBeInTheDocument();
    expect(screen.getByText(PRIVACY.summary)).toBeInTheDocument();
    expect(screen.getByText(/Berlaku sejak:/)).toBeInTheDocument();
    expect(screen.getByText("17 Oktober 2026")).toBeInTheDocument();
  });

  it("renders every section as an H2 with its body", () => {
    render(<LegalPage doc={PRIVACY} />);
    PRIVACY.sections.forEach((section) => {
      expect(screen.getByRole("heading", { level: 2, name: section.heading })).toBeInTheDocument();
    });
  });

  it("exposes a table of contents that anchors to the sections", () => {
    const { container } = render(<LegalPage doc={TERMS} />);
    const toc = container.querySelector("nav[aria-labelledby='legal-toc-heading']");
    expect(toc).not.toBeNull();
    TERMS.sections.forEach((section) => {
      expect(toc?.querySelector(`a[href="#${section.id}"]`)).not.toBeNull();
      expect(container.querySelector(`#${section.id}`)).not.toBeNull();
    });
  });

  it("keeps the honesty marker visible while the document is a draft", () => {
    render(<LegalPage doc={PRIVACY} />);
    expect(screen.getByText(/draf internal/i)).toBeInTheDocument();
  });

  it("emits WebPage JSON-LD (never FAQPage)", () => {
    const { container } = render(<LegalPage doc={PRIVACY} />);
    const script = container.querySelector("script[type='application/ld+json']");
    expect(script).not.toBeNull();
    const data = JSON.parse(script?.innerHTML ?? "{}");
    expect(data["@type"]).toBe("WebPage");
    expect(script?.innerHTML ?? "").not.toContain("FAQPage");
  });

  it("uses no unverifiable certification claims", () => {
    const { container } = render(<LegalPage doc={TERMS} />);
    const text = container.textContent ?? "";
    ["ISO 27001", "tersertifikasi", "100% aman"].forEach((claim) =>
      expect(text).not.toContain(claim),
    );
  });
});
