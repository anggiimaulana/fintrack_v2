import { describe, it, expect, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { PricingSections } from "./PricingSections";
import { MICROCOPY } from "@/lib/marketing/copy";
import { PRICING } from "@/lib/marketing/pricing";
import { PRICING_FAQ } from "@/lib/marketing/faq";
import { SAMPLE_PLANS } from "@/test/fixtures/public-plans";

// The retry block is a client component that reads the router.
vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: vi.fn() }),
}));

describe("PricingSections", () => {
  it("opens with the H1, breadcrumb, and honest chips", () => {
    render(<PricingSections plans={SAMPLE_PLANS} />);
    expect(screen.getByRole("heading", { level: 1, name: PRICING.hero.h1 })).toBeInTheDocument();
    const crumbs = screen.getByRole("navigation", { name: "Breadcrumb" });
    expect(within(crumbs).getByText("Harga")).toHaveAttribute("aria-current", "page");
    PRICING.hero.chips.forEach((chip) => expect(screen.getByText(chip)).toBeInTheDocument());
  });

  it("exposes the comparison anchor the pricing FAQ links to", () => {
    const { container } = render(<PricingSections plans={SAMPLE_PLANS} />);
    expect(container.querySelector("#perbandingan")).not.toBeNull();
  });

  it("answers the eight pricing objections in a native accordion", () => {
    const { container } = render(<PricingSections plans={SAMPLE_PLANS} />);
    const faq = container.querySelector("#faq-harga") as HTMLElement;
    expect(faq.querySelectorAll("details")).toHaveLength(PRICING_FAQ.length);
    expect(within(faq).getByText(PRICING_FAQ[0].q)).toBeInTheDocument();
  });

  it("explains both payment methods", () => {
    render(<PricingSections plans={SAMPLE_PLANS} />);
    PRICING.payment.methods.forEach((method) =>
      expect(screen.getByRole("heading", { name: method.title })).toBeInTheDocument(),
    );
  });

  it("closes with the register CTA", () => {
    render(<PricingSections plans={SAMPLE_PLANS} />);
    const section = screen.getByRole("heading", { name: PRICING.cta.h2 }).closest("section") as HTMLElement;
    expect(within(section).getByRole("link", { name: /Mulai Gratis/ })).toHaveAttribute(
      "href",
      "/register",
    );
  });

  it("never advertises bank sync and never lists unenforced limits", () => {
    const { container } = render(<PricingSections plans={SAMPLE_PLANS} />);
    const text = container.textContent ?? "";
    expect(text).not.toMatch(/\bbank\b/i);
    expect(text).not.toMatch(/bank_sync/i);
    expect(text).not.toContain("Kategori");
    expect(text).not.toContain("Dompet");
  });

  it("degrades to a retry block when the catalog is empty but keeps the FAQ and CTA", () => {
    const { container } = render(<PricingSections plans={[]} />);
    expect(screen.getByText(MICROCOPY.catalogError)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: MICROCOPY.retry })).toBeInTheDocument();
    expect(container.querySelector("#perbandingan")).toBeNull();
    expect(container.querySelectorAll("details")).toHaveLength(PRICING_FAQ.length);
    expect(screen.getByRole("link", { name: /Mulai Gratis/ })).toHaveAttribute("href", "/register");
  });
});
