import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { axe } from "vitest-axe";
import { LandingSections } from "./LandingSections";
import { SAMPLE_PLANS } from "@/test/fixtures/public-plans";

describe("LandingSections a11y", () => {
  it("has no axe violations inside the main landmark", async () => {
    const { container } = render(
      <main>
        <LandingSections plans={SAMPLE_PLANS} />
      </main>,
    );
    expect(await axe(container)).toHaveNoViolations();
  });

  it("keeps every decorative marquee out of the accessibility tree", () => {
    const { container } = render(
      <main>
        <LandingSections plans={SAMPLE_PLANS} />
      </main>,
    );
    const tracks = container.querySelectorAll(".neo-marquee");
    expect(tracks.length).toBeGreaterThan(0);
    tracks.forEach((track) => expect(track).toHaveAttribute("aria-hidden", "true"));
  });
});
