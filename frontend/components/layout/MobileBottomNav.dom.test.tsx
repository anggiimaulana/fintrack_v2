import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { axe } from "@/test/setup";
import { MobileBottomNav } from "./MobileBottomNav";
import { MOBILE_BOTTOM_NAV_ITEMS } from "@/lib/nav";

const pathname = vi.fn(() => "/user");
vi.mock("next/navigation", () => ({
  usePathname: () => pathname(),
}));

beforeEach(() => {
  pathname.mockReturnValue("/user");
});

describe("MobileBottomNav", () => {
  it("renders every bottom-bar item with its label", () => {
    render(<MobileBottomNav />);
    const nav = screen.getByRole("navigation", { name: "Navigasi utama" });
    for (const item of MOBILE_BOTTOM_NAV_ITEMS) {
      expect(screen.getByRole("link", { name: item.label })).toHaveAttribute("href", item.href);
    }
    // One link per bar item (the FAB is a sibling of the nav, not inside it).
    expect(nav.querySelectorAll("a")).toHaveLength(MOBILE_BOTTOM_NAV_ITEMS.length);
  });

  it("marks only the active route with aria-current", () => {
    pathname.mockReturnValue("/user/transactions");
    render(<MobileBottomNav />);
    const current = screen.getAllByRole("link").filter((l) => l.getAttribute("aria-current") === "page");
    expect(current).toHaveLength(1);
    expect(current[0]).toHaveAttribute("href", "/user/transactions");
  });

  it("shows the AI Chat FAB off the chat route", () => {
    render(<MobileBottomNav />);
    expect(screen.getByRole("link", { name: "Buka AI Chat" })).toHaveAttribute("href", "/user/ai");
  });

  it("hides the AI Chat FAB while already on the chat route", () => {
    pathname.mockReturnValue("/user/ai");
    render(<MobileBottomNav />);
    expect(screen.queryByRole("link", { name: "Buka AI Chat" })).not.toBeInTheDocument();
    // The bar itself stays, so you can always leave the chat.
    expect(screen.getByRole("navigation", { name: "Navigasi utama" })).toBeInTheDocument();
  });

  it("highlights Dashboard on a nested dashboard route", () => {
    pathname.mockReturnValue("/user");
    render(<MobileBottomNav />);
    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute("aria-current", "page");
  });

  it("has no accessibility violations", async () => {
    const { container } = render(<MobileBottomNav />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
