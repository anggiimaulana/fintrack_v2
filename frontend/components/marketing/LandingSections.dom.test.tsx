import { describe, it, expect } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { LandingSections } from "./LandingSections";
import { PricingTeaser } from "./PricingTeaser";
import { LANDING } from "@/lib/marketing/copy";
import { LANDING_FAQ } from "@/lib/marketing/faq";
import { FEATURES } from "@/lib/marketing/features";
import { SAMPLE_PLANS } from "@/test/fixtures/public-plans";

const SECTION_ANCHORS = [
  "#hero",
  "#solution",
  "#features",
  "#how-it-works",
  "#insights",
  "#pricing",
  "#who-its-for",
  "#faq",
];

describe("LandingSections", () => {
  it("renders every in-page anchor the nav and CTAs point at", () => {
    const { container } = render(<LandingSections plans={SAMPLE_PLANS} />);
    SECTION_ANCHORS.forEach((id) => {
      expect(container.querySelector(id)).not.toBeNull();
    });
  });

  it("never ships a dead '#' link", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    screen.getAllByRole("link").forEach((link) => expect(link.getAttribute("href")).not.toBe("#"));
  });

  it("avoids unverifiable social proof, fake names, and the removed BYOK claim", () => {
    const { container } = render(<LandingSections plans={SAMPLE_PLANS} />);
    const text = container.textContent ?? "";
    ["ribuan pengguna", "Budi Santoso", "Siti Aminah", "Andi Wijaya"].forEach((bad) => {
      expect(text).not.toContain(bad);
    });
    expect(container.innerHTML).not.toMatch(/api key milik|pakai api key anda|byok|tambahkan api key/i);
  });

  it("shows exactly one H1", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
  });

  it("renders the four factual trust points", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    LANDING.trustBar.forEach((point) => expect(screen.getByText(point.text)).toBeInTheDocument());
  });

  it("contrasts problems with solutions", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    expect(screen.getByRole("heading", { name: LANDING.problemSolution.h2 })).toBeInTheDocument();
    LANDING.problemSolution.problems.forEach((p) => expect(screen.getByText(p)).toBeInTheDocument());
    LANDING.problemSolution.solutions.forEach((s) => expect(screen.getByText(s)).toBeInTheDocument());
  });

  it("links each of the six features to its deep-dive page", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    const links = screen.getAllByRole("link", { name: /Pelajari lebih lanjut/ });
    expect(links).toHaveLength(FEATURES.length);
    FEATURES.forEach((feature) => {
      expect(screen.getByRole("link", { name: new RegExp(feature.title) })).toHaveAttribute(
        "href",
        `/features/${feature.slug}`,
      );
    });
  });

  it("renders three how-it-works steps without the old API-key step", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    const section = document.querySelector("#how-it-works") as HTMLElement;
    expect(within(section).getAllByRole("listitem")).toHaveLength(3);
    LANDING.howItWorks.steps.forEach((step) => expect(within(section).getByText(step.title)).toBeInTheDocument());
    expect(section.textContent?.toLowerCase()).not.toContain("api key");
  });

  it("lists the five financial-health statuses with text labels", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    const section = document.querySelector("#insights") as HTMLElement;
    LANDING.spk.statuses.forEach((status) =>
      expect(within(section).getAllByText(status).length).toBeGreaterThan(0),
    );
    expect(within(section).getByText(String(LANDING.spk.example.score))).toBeInTheDocument();
  });

  it("uses honest personas instead of testimonials", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    const section = document.querySelector("#who-its-for") as HTMLElement;
    LANDING.personas.items.forEach((item) => {
      expect(within(section).getByText(item.title)).toBeInTheDocument();
      expect(within(section).getByText(item.text)).toBeInTheDocument();
    });
  });

  it("renders a native-details accordion with every landing FAQ", () => {
    const { container } = render(<LandingSections plans={SAMPLE_PLANS} />);
    const faq = container.querySelector("#faq") as HTMLElement;
    expect(faq.querySelectorAll("details")).toHaveLength(LANDING_FAQ.length);
    LANDING_FAQ.forEach((item) => expect(within(faq).getByText(item.q)).toBeInTheDocument());
  });

  it("closes with the register CTA", () => {
    render(<LandingSections plans={SAMPLE_PLANS} />);
    expect(screen.getByRole("heading", { name: LANDING.closingCta.h2 })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Daftar Gratis/ })).toHaveAttribute("href", "/register");
  });
});

describe("PricingTeaser", () => {
  it("renders one card per plan with the popular badge on Pro", () => {
    render(<PricingTeaser plans={SAMPLE_PLANS} />);
    SAMPLE_PLANS.forEach((plan) => expect(screen.getByRole("heading", { name: plan.name })).toBeInTheDocument());
    expect(screen.getByText(LANDING.pricingTeaser.popularBadge)).toBeInTheDocument();
    expect(screen.getByText("Gratis")).toBeInTheDocument();
  });

  it("links every card to the pricing page", () => {
    render(<PricingTeaser plans={SAMPLE_PLANS} />);
    screen
      .getAllByRole("link", { name: LANDING.pricingTeaser.cta.label })
      .forEach((link) => expect(link).toHaveAttribute("href", "/pricing"));
  });

  it("degrades to a single link when the catalog is empty", () => {
    render(<PricingTeaser plans={[]} />);
    const fallback = screen.getByRole("link", { name: LANDING.pricingTeaser.emptyFallback.label });
    expect(fallback).toHaveAttribute("href", "/pricing");
    expect(screen.queryByText(LANDING.pricingTeaser.popularBadge)).not.toBeInTheDocument();
  });
});
