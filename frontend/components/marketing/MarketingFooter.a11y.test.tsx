import { describe, it, expect } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { axe } from "vitest-axe";
import { MarketingFooter } from "./MarketingFooter";
import { FOOTER_COLUMNS } from "@/lib/marketing/copy";

describe("MarketingFooter a11y", () => {
  it("has no axe violations", async () => {
    const { container } = render(<MarketingFooter />);
    expect(await axe(container)).toHaveNoViolations();
  });

  it("gives every footer link meaningful text", () => {
    render(<MarketingFooter />);
    screen.getAllByRole("link").forEach((link) => {
      expect((link.textContent ?? "").trim().length).toBeGreaterThan(0);
    });
  });

  it("renders every footer column with its links", () => {
    render(<MarketingFooter />);
    FOOTER_COLUMNS.forEach((column) => {
      const nav = screen.getByRole("navigation", { name: column.title });
      column.links.forEach((link) => {
        expect(within(nav).getByRole("link", { name: link.label })).toHaveAttribute("href", link.href);
      });
    });
  });

  it("offers both contact channels", () => {
    render(<MarketingFooter />);
    expect(screen.getByRole("link", { name: /@/ })).toHaveAttribute("href", expect.stringContaining("mailto:"));
    expect(screen.getByRole("link", { name: /WhatsApp/ })).toHaveAttribute(
      "href",
      expect.stringContaining("https://wa.me/"),
    );
  });
});
