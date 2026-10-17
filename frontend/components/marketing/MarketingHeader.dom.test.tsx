import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import { MarketingHeader } from "./MarketingHeader";
import { NAV, PRIMARY_CTA, SIGN_IN_CTA } from "@/lib/marketing/copy";

const pathname = vi.fn(() => "/");
vi.mock("next/navigation", () => ({
  usePathname: () => pathname(),
}));

beforeEach(() => {
  pathname.mockReturnValue("/");
});

describe("MarketingHeader", () => {
  it("renders the desktop navigation with real routes", () => {
    render(<MarketingHeader />);
    const nav = screen.getByRole("navigation", { name: "Navigasi utama" });

    expect(within(nav).getByRole("link", { name: "Fitur" })).toHaveAttribute("href", "/features");
    expect(within(nav).getByRole("link", { name: "Harga" })).toHaveAttribute("href", "/pricing");
    expect(within(nav).getByRole("link", { name: "Jelajahi" })).toHaveAttribute("href", "/explore");
    expect(within(nav).getByRole("link", { name: "Cara Kerja" })).toHaveAttribute("href", "/#how-it-works");
  });

  it("never uses an empty '#' link", () => {
    render(<MarketingHeader />);
    const links = screen.getAllByRole("link");
    links.forEach((link) => expect(link.getAttribute("href")).not.toBe("#"));
  });

  it("starts with the mobile panel collapsed", () => {
    render(<MarketingHeader />);
    const toggle = screen.getByRole("button", { name: "Buka menu" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("navigation", { name: "Navigasi seluler" })).not.toBeInTheDocument();
  });

  it("opens and closes the mobile panel with the toggle button", () => {
    render(<MarketingHeader />);
    fireEvent.click(screen.getByRole("button", { name: "Buka menu" }));

    expect(screen.getByRole("button", { name: "Tutup menu" })).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("navigation", { name: "Navigasi seluler" })).toBeInTheDocument();
    // Drawer focuses purely on Account actions (primary navigation is in bottom nav and /explore).
    const drawer = document.getElementById("marketing-mobile-menu")!;
    expect(within(drawer).getByRole("link", { name: SIGN_IN_CTA.label })).toHaveAttribute("href", SIGN_IN_CTA.href);
    expect(within(drawer).getByRole("link", { name: PRIMARY_CTA.label })).toHaveAttribute("href", PRIMARY_CTA.href);
    expect(within(drawer).getByText("FinTrack © 2026")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Tutup menu" }));
    expect(screen.queryByRole("navigation", { name: "Navigasi seluler" })).not.toBeInTheDocument();
  });

  it("closes the mobile panel on Escape", () => {
    render(<MarketingHeader />);
    fireEvent.click(screen.getByRole("button", { name: "Buka menu" }));
    expect(screen.getByRole("navigation", { name: "Navigasi seluler" })).toBeInTheDocument();

    fireEvent.keyDown(window, { key: "Escape" });
    expect(screen.queryByRole("navigation", { name: "Navigasi seluler" })).not.toBeInTheDocument();
  });

  it("marks the active route with aria-current", () => {
    pathname.mockReturnValue("/features/receipt-scan");
    render(<MarketingHeader />);
    const nav = screen.getByRole("navigation", { name: "Navigasi utama" });
    expect(within(nav).getByRole("link", { name: "Fitur" })).toHaveAttribute("aria-current", "page");
  });
});
