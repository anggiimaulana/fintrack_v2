import { describe, it, expect } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { Hero } from "./Hero";
import { LANDING } from "@/lib/marketing/copy";

describe("Hero", () => {
  it("puts the primary keyword in the single H1", () => {
    render(<Hero />);
    const h1 = screen.getByRole("heading", { level: 1 });
    expect(h1.textContent?.toLowerCase()).toContain("catat keuangan otomatis");
  });

  it("highlights the accent word with the acid chip", () => {
    render(<Hero />);
    const h1 = screen.getByRole("heading", { level: 1 });
    const highlight = within(h1).getByText(LANDING.hero.h1HighlightWord);
    expect(highlight).toHaveClass("neo-highlight");
  });

  it("shows the eyebrow and subhead", () => {
    render(<Hero />);
    expect(screen.getByText(LANDING.hero.eyebrow)).toBeInTheDocument();
    expect(screen.getByText(LANDING.hero.subhead)).toBeInTheDocument();
  });

  it("points the CTAs at real targets", () => {
    render(<Hero />);
    expect(screen.getByRole("link", { name: /Mulai Gratis/ })).toHaveAttribute("href", "/register");
    expect(screen.getByRole("link", { name: /Lihat Cara Kerja/ })).toHaveAttribute("href", "#how-it-works");
  });

  it("lists the honest trust chips", () => {
    render(<Hero />);
    LANDING.hero.trustChips.forEach((chip) => expect(screen.getByText(chip)).toBeInTheDocument());
  });

  it("embeds the interactive demo", () => {
    render(<Hero />);
    expect(screen.getByRole("tablist")).toBeInTheDocument();
  });

  it("never re-introduces the removed BYOK claim", () => {
    const { container } = render(<Hero />);
    expect(container.innerHTML).not.toMatch(/api key milik|pakai api key anda|byok|tambahkan api key/i);
  });
});
