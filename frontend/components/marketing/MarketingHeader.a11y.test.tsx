import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { axe } from "vitest-axe";
import { MarketingHeader } from "./MarketingHeader";

vi.mock("next/navigation", () => ({
  usePathname: () => "/",
}));

describe("MarketingHeader a11y", () => {
  it("has no axe violations", async () => {
    const { container } = render(<MarketingHeader />);
    expect(await axe(container)).toHaveNoViolations();
  });

  it("gives the menu toggle an accessible name", () => {
    render(<MarketingHeader />);
    expect(screen.getByRole("button", { name: "Buka menu" })).toBeInTheDocument();
  });

  it("exposes the primary call to action", () => {
    render(<MarketingHeader />);
    expect(screen.getAllByRole("link", { name: "Mulai Gratis" }).length).toBeGreaterThan(0);
  });
});
