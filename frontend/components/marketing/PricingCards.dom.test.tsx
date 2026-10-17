import { describe, it, expect } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import { PricingCards } from "./PricingCards";
import { PRICING } from "@/lib/marketing/pricing";
import { SAMPLE_PLANS } from "@/test/fixtures/public-plans";

describe("PricingCards", () => {
  it("renders one card per plan in catalog order", () => {
    render(<PricingCards plans={SAMPLE_PLANS} />);
    ["Free", "Pro", "Business"].forEach((name) =>
      expect(screen.getByRole("heading", { name })).toBeInTheDocument(),
    );
  });

  it("starts on the monthly cycle and marks the toggle state", () => {
    render(<PricingCards plans={SAMPLE_PLANS} />);
    const group = screen.getByRole("group", { name: PRICING.cycle.groupLabel });
    expect(within(group).getByRole("button", { name: PRICING.cycle.monthly })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(within(group).getByRole("button", { name: PRICING.cycle.yearly })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
    expect(screen.getByText(/39\.000/)).toBeInTheDocument();
  });

  it("switches to yearly prices with an honest effective monthly figure", () => {
    render(<PricingCards plans={SAMPLE_PLANS} />);
    fireEvent.click(screen.getByRole("button", { name: PRICING.cycle.yearly }));

    const group = screen.getByRole("group", { name: PRICING.cycle.groupLabel });
    expect(within(group).getByRole("button", { name: PRICING.cycle.yearly })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByText(/390\.000/)).toBeInTheDocument();
    expect(screen.getAllByText(/32\.500/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Setara/).length).toBeGreaterThan(0);
  });

  it("exposes each price block as a polite live region", () => {
    const { container } = render(<PricingCards plans={SAMPLE_PLANS} />);
    expect(container.querySelectorAll('[aria-live="polite"]')).toHaveLength(SAMPLE_PLANS.length);
  });

  it("badges only the Pro plan as popular", () => {
    render(<PricingCards plans={SAMPLE_PLANS} />);
    expect(screen.getAllByText(PRICING.card.popularBadge)).toHaveLength(1);
  });

  it("labels the free plan and routes paid CTAs with the plan code", () => {
    render(<PricingCards plans={SAMPLE_PLANS} />);
    expect(screen.getByText(PRICING.card.freePriceLabel)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: PRICING.card.freeCta })).toHaveAttribute(
      "href",
      "/register",
    );
    expect(screen.getByRole("link", { name: `${PRICING.card.paidCtaPrefix} Pro` })).toHaveAttribute(
      "href",
      "/register?plan=pro",
    );
    expect(
      screen.getByRole("link", { name: `${PRICING.card.paidCtaPrefix} Business` }),
    ).toHaveAttribute("href", "/register?plan=business");
  });

  it("shows the tax line only when the catalog enables it", () => {
    const { container } = render(<PricingCards plans={SAMPLE_PLANS} />);
    // Pro and Business enable tax; Free does not.
    expect(screen.getAllByText(/PPN 11%/)).toHaveLength(2);
    expect(screen.getByText(/43\.290/)).toBeInTheDocument();
    expect(within(container.querySelectorAll("li")[0] as HTMLElement).queryByText(/PPN/)).toBeNull();
  });

  it("never renders a raw '-1' limit on a card", () => {
    const { container } = render(<PricingCards plans={SAMPLE_PLANS} />);
    expect(container.textContent).not.toMatch(/-1/);
  });
});
