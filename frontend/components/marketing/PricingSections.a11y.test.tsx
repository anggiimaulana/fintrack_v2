import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { axe } from "vitest-axe";
import { PricingSections } from "./PricingSections";
import { PRICING } from "@/lib/marketing/pricing";
import { SAMPLE_PLANS } from "@/test/fixtures/public-plans";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: vi.fn() }),
}));

describe("PricingSections a11y", () => {
  it("has no axe violations with a live catalog", async () => {
    const { container } = render(
      <main>
        <PricingSections plans={SAMPLE_PLANS} />
      </main>,
    );
    expect(await axe(container)).toHaveNoViolations();
  });

  it("has no axe violations on the empty-catalog fallback", async () => {
    const { container } = render(
      <main>
        <PricingSections plans={[]} />
      </main>,
    );
    expect(await axe(container)).toHaveNoViolations();
  });

  it("keeps a single H1 and never skips a heading level", () => {
    render(
      <main>
        <PricingSections plans={SAMPLE_PLANS} />
      </main>,
    );
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
    const levels = screen
      .getAllByRole("heading")
      .map((h) => Number(h.tagName.slice(1)));
    levels.forEach((level, index) => {
      if (index === 0) return;
      expect(level - levels[index - 1]).toBeLessThanOrEqual(1);
    });
  });

  it("labels the comparison tables for assistive tech", () => {
    render(
      <main>
        <PricingSections plans={SAMPLE_PLANS} />
      </main>,
    );
    const tables = screen.getAllByRole("table");
    expect(tables).toHaveLength(2);
    tables.forEach((table) => expect(table).toHaveAccessibleName());
  });

  it("titles the pricing accordion for screen readers", () => {
    render(
      <main>
        <PricingSections plans={SAMPLE_PLANS} />
      </main>,
    );
    expect(screen.getByRole("heading", { name: PRICING.faq.h2 })).toBeInTheDocument();
  });
});
