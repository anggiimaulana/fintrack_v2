import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ReceiptTapeMarquee } from "./ReceiptTapeMarquee";
import { TickerBand } from "./TickerBand";
import { StatCounters } from "./StatCounters";
import { LANDING } from "@/lib/marketing/copy";

describe("ReceiptTapeMarquee", () => {
  it("hides the moving track from assistive tech", () => {
    const { container } = render(<ReceiptTapeMarquee />);
    const track = container.querySelector(".neo-marquee");
    expect(track).not.toBeNull();
    expect(track).toHaveAttribute("aria-hidden", "true");
  });

  it("duplicates the track for a seamless loop and marks the clone", () => {
    const { container } = render(<ReceiptTapeMarquee />);
    expect(container.querySelectorAll(".neo-marquee-clone")).toHaveLength(1);
  });

  it("exposes the sample rows to screen readers and captions them as an illustration", () => {
    const { container } = render(<ReceiptTapeMarquee />);
    expect(screen.getByText(LANDING.receiptTape.caption)).toBeInTheDocument();
    const staticList = container.querySelector("ul.sr-only");
    expect(staticList).not.toBeNull();
    LANDING.receiptTape.rows.forEach((row) => {
      expect(staticList?.textContent).toContain(row.label);
      expect(staticList?.textContent).toContain(row.amount);
    });
  });
});

describe("TickerBand", () => {
  it("is decorative in the visual layer", () => {
    const { container } = render(<TickerBand />);
    expect(container.querySelector(".neo-marquee")).toHaveAttribute("aria-hidden", "true");
    expect(container.querySelectorAll(".neo-marquee-clone")).toHaveLength(1);
  });

  it("provides one static text equivalent", () => {
    render(<TickerBand />);
    expect(screen.getByText(LANDING.ticker.ariaLabel)).toBeInTheDocument();
  });
});

describe("StatCounters", () => {
  it("renders the final value of every honest stat", () => {
    render(<StatCounters />);
    expect(screen.getByText("Rp 0")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("labels each stat", () => {
    render(<StatCounters />);
    LANDING.stats.forEach((stat) => {
      expect(screen.getByText(stat.label)).toBeInTheDocument();
    });
  });
});
