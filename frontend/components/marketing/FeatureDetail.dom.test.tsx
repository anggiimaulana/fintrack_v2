import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { FeatureDetail } from "./FeatureDetail";
import { FEATURE_PAGE } from "@/lib/marketing/copy";
import { getFeature, featureHref, PLAN_LABEL, type FeaturePage } from "@/lib/marketing/features";

const receiptScan = getFeature("receipt-scan")!;
const aiChat = getFeature("ai-chat")!;

describe("FeatureDetail", () => {
  it("renders the hero, problem, steps, benefits, detail, and example", () => {
    const { container } = render(<FeatureDetail feature={receiptScan} />);
    expect(screen.getByRole("heading", { level: 1, name: receiptScan.title })).toBeInTheDocument();
    expect(screen.getByText(receiptScan.problem)).toBeInTheDocument();
    receiptScan.howItWorks.forEach((step) => expect(screen.getByText(step)).toBeInTheDocument());
    receiptScan.benefits.forEach((benefit) => expect(screen.getByText(benefit)).toBeInTheDocument());
    receiptScan.detail.forEach((paragraph) =>
      expect(screen.getByText(paragraph)).toBeInTheDocument(),
    );
    expect(screen.getByText(receiptScan.example.prompt)).toBeInTheDocument();
    expect(screen.getByText(receiptScan.example.result)).toBeInTheDocument();
    // The example is an illustration, never a testimonial.
    expect(container.textContent).toMatch(/contoh tampilan/i);
  });

  it("lists the numbered how-it-works steps in order", () => {
    const { container } = render(<FeatureDetail feature={receiptScan} />);
    const section = container.querySelector("[aria-labelledby='feature-how']");
    const items = Array.from(section?.querySelectorAll("ol > li") ?? []);
    expect(items).toHaveLength(receiptScan.howItWorks.length);
    expect(items[0].textContent).toContain(receiptScan.howItWorks[0]);
  });

  it("states plan availability from the catalogue gating", () => {
    render(<FeatureDetail feature={receiptScan} />);
    expect(screen.getByText("Pro, Business")).toBeInTheDocument();
  });

  it("lists all three plans for an ungated feature", () => {
    render(<FeatureDetail feature={aiChat} />);
    expect(screen.getByText("Free, Pro, Business")).toBeInTheDocument();
    expect(screen.getByText(PLAN_LABEL.business, { exact: false })).toBeInTheDocument();
  });

  it("renders the use cases and the honest notes list", () => {
    render(<FeatureDetail feature={receiptScan} />);
    receiptScan.useCases.forEach((useCase) =>
      expect(screen.getByRole("heading", { name: useCase.title })).toBeInTheDocument(),
    );
    receiptScan.notes.forEach((note) => expect(screen.getByText(note)).toBeInTheDocument());
    expect(screen.getByRole("heading", { name: FEATURE_PAGE.notesTitle })).toBeInTheDocument();
  });

  it("cross-links the related features and back to the hub", () => {
    render(<FeatureDetail feature={receiptScan} />);
    receiptScan.relatedSlugs.forEach((slug) => {
      const related: FeaturePage = getFeature(slug)!;
      const links = screen.getAllByRole("link", { name: new RegExp(related.title) });
      expect(links.some((link) => link.getAttribute("href") === featureHref(slug))).toBe(true);
    });
    expect(screen.getByRole("link", { name: FEATURE_PAGE.allFeatures.label })).toHaveAttribute(
      "href",
      "/features",
    );
  });

  it("always offers a way back to pricing and sign-up", () => {
    render(<FeatureDetail feature={aiChat} />);
    expect(screen.getAllByRole("link", { name: FEATURE_PAGE.primaryCta.label })[0]).toHaveAttribute(
      "href",
      "/register",
    );
    expect(screen.getAllByRole("link", { name: FEATURE_PAGE.pricingCta.label })[0]).toHaveAttribute(
      "href",
      "/pricing",
    );
  });

  it("never mentions a bank-sync feature", () => {
    const { container } = render(<FeatureDetail feature={aiChat} />);
    const text = container.textContent ?? "";
    expect(text).not.toMatch(/bank_sync/i);
    expect(text).not.toMatch(/sinkronisasi bank/i);
  });
});
