import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { MarketingMobileBottomNav } from "./MarketingMobileBottomNav";

const pathname = vi.fn(() => "/");
vi.mock("next/navigation", () => ({
  usePathname: () => pathname(),
}));

beforeEach(() => {
  pathname.mockReturnValue("/");
});

describe("MarketingMobileBottomNav", () => {
  it("renders all 5 primary mobile destinations", () => {
    render(<MarketingMobileBottomNav />);
    const nav = screen.getByRole("navigation", { name: "Navigasi bawah seluler" });

    expect(within(nav).getByRole("link", { name: "Beranda" })).toHaveAttribute("href", "/");
    expect(within(nav).getByRole("link", { name: "Fitur" })).toHaveAttribute("href", "/features");
    expect(within(nav).getByRole("link", { name: "Cara Kerja" })).toHaveAttribute("href", "/#how-it-works");
    expect(within(nav).getByRole("link", { name: "Harga" })).toHaveAttribute("href", "/pricing");
    expect(within(nav).getByRole("link", { name: "Jelajahi" })).toHaveAttribute("href", "/explore");
  });

  it("marks the active destination with aria-current", () => {
    pathname.mockReturnValue("/features");
    render(<MarketingMobileBottomNav />);
    const nav = screen.getByRole("navigation", { name: "Navigasi bawah seluler" });

    expect(within(nav).getByRole("link", { name: "Fitur" })).toHaveAttribute("aria-current", "page");
    expect(within(nav).getByRole("link", { name: "Beranda" })).not.toHaveAttribute("aria-current");
  });
});
