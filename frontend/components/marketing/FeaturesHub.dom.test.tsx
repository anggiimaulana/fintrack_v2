import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { FeaturesHub } from "./FeaturesHub";
import { FEATURES_HUB } from "@/lib/marketing/copy";
import { FEATURES, featureHref } from "@/lib/marketing/features";

describe("FeaturesHub", () => {
  it("renders the single H1 and every feature card", () => {
    render(<FeaturesHub />);
    expect(screen.getByRole("heading", { level: 1, name: FEATURES_HUB.h1 })).toBeInTheDocument();
    FEATURES.forEach((feature) => {
      expect(screen.getByRole("heading", { level: 2, name: feature.title })).toBeInTheDocument();
      expect(screen.getByText(feature.tagline)).toBeInTheDocument();
    });
  });

  it("links every card to its deep-dive page", () => {
    render(<FeaturesHub />);
    FEATURES.forEach((feature) => {
      const link = screen.getByRole("link", {
        name: `${FEATURES_HUB.cardCta}: ${feature.title}`,
      });
      expect(link).toHaveAttribute("href", featureHref(feature.slug));
    });
  });

  it("exposes the primary and pricing CTAs", () => {
    render(<FeaturesHub />);
    expect(screen.getAllByRole("link", { name: /Daftar Gratis/ })[0]).toHaveAttribute(
      "href",
      "/register",
    );
    expect(screen.getAllByRole("link", { name: FEATURES_HUB.pricingCta.label })[0]).toHaveAttribute(
      "href",
      "/pricing",
    );
  });

  it("renders the 'how it connects' explainer and the breadcrumb trail", () => {
    render(<FeaturesHub />);
    expect(
      screen.getByRole("heading", { name: FEATURES_HUB.connected.title }),
    ).toBeInTheDocument();
    FEATURES_HUB.connected.paragraphs.forEach((paragraph) =>
      expect(screen.getByText(paragraph)).toBeInTheDocument(),
    );
    expect(screen.getByRole("navigation", { name: "Breadcrumb" })).toBeInTheDocument();
  });

  it("never mentions a bank-sync feature", () => {
    const { container } = render(<FeaturesHub />);
    const text = container.textContent ?? "";
    expect(text).not.toMatch(/bank_sync/i);
    expect(text).not.toMatch(/sinkronisasi bank/i);
  });
});
