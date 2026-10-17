import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { axe } from "vitest-axe";
import { AboutSections } from "./AboutSections";
import { ContactSections } from "./ContactSections";
import { FaqSections } from "./FaqSections";
import { FeatureDetail } from "./FeatureDetail";
import { FeaturesHub } from "./FeaturesHub";
import { LegalPage } from "./LegalPage";
import { getFeature } from "@/lib/marketing/features";
import { PRIVACY, TERMS } from "@/lib/marketing/legal";

async function expectNoViolations(node: HTMLElement) {
  expect(await axe(node)).toHaveNoViolations();
}

describe("marketing page a11y (axe)", () => {
  it("features hub", async () => {
    const { container } = render(
      <main>
        <FeaturesHub />
      </main>,
    );
    await expectNoViolations(container);
  });

  it("feature deep-dive", async () => {
    const { container } = render(
      <main>
        <FeatureDetail feature={getFeature("receipt-scan")!} />
      </main>,
    );
    await expectNoViolations(container);
  });

  it("about", async () => {
    const { container } = render(
      <main>
        <AboutSections />
      </main>,
    );
    await expectNoViolations(container);
  });

  it("contact", async () => {
    const { container } = render(
      <main>
        <ContactSections />
      </main>,
    );
    await expectNoViolations(container);
  });

  it("faq", async () => {
    const { container } = render(
      <main>
        <FaqSections />
      </main>,
    );
    await expectNoViolations(container);
  });

  it("privacy policy", async () => {
    const { container } = render(
      <main>
        <LegalPage doc={PRIVACY} />
      </main>,
    );
    await expectNoViolations(container);
  });

  it("terms", async () => {
    const { container } = render(
      <main>
        <LegalPage doc={TERMS} />
      </main>,
    );
    await expectNoViolations(container);
  });
});
