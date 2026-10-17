import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ExploreHub } from "./ExploreHub";
import { EXPLORE_HUB } from "@/lib/marketing/copy";

describe("ExploreHub", () => {
  it("renders the H1 and all 5 exploration cards", () => {
    render(<ExploreHub />);
    expect(screen.getByRole("heading", { level: 1, name: EXPLORE_HUB.h1 })).toBeInTheDocument();
    EXPLORE_HUB.cards.forEach((card) => {
      expect(screen.getByRole("heading", { level: 2, name: card.title })).toBeInTheDocument();
      expect(screen.getByText(card.tagline)).toBeInTheDocument();
      expect(screen.getByText(card.badge)).toBeInTheDocument();
    });
  });

  it("links every card to its corresponding destination", () => {
    render(<ExploreHub />);
    EXPLORE_HUB.cards.forEach((card) => {
      const link = screen.getByRole("link", {
        name: `${card.cta}: ${card.title}`,
      });
      expect(link).toHaveAttribute("href", card.href);
    });
  });

  it("includes breadcrumbs back to home", () => {
    render(<ExploreHub />);
    expect(screen.getByRole("link", { name: "Beranda" })).toHaveAttribute("href", "/");
    expect(screen.getByText("Jelajahi")).toBeInTheDocument();
  });
});
