import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { axe } from "vitest-axe";
import { TransactionRow, type TransactionData } from "./TransactionRow";

const base: TransactionData = {
  id: "t1",
  type: "expense",
  amount: 49000,
  category: "Makan",
  description: "Makan siang",
  date: "2026-10-11",
  source: "manual",
};

function renderRow(overrides: Partial<TransactionData> = {}) {
  return render(<TransactionRow transaction={{ ...base, ...overrides }} />);
}

/**
 * The row is the densest UI in the app (badges, amounts, action buttons), so it
 * is the most likely place for an unlabelled control to slip in.
 */
describe("TransactionRow a11y", () => {
  for (const source of ["manual", "ai", "voice", "receipt"] as const) {
    it(`has no axe violations (source=${source})`, async () => {
      const { container } = renderRow({ source });
      expect(await axe(container)).toHaveNoViolations();
    });
  }

  it("has no axe violations for income rows", async () => {
    const { container } = renderRow({ type: "income", amount: 1500000 });
    expect(await axe(container)).toHaveNoViolations();
  });

  it("gives every icon-only action button an accessible name", () => {
    const { container } = renderRow();
    const buttons = Array.from(container.querySelectorAll("button"));
    expect(buttons.length).toBeGreaterThan(0);
    for (const b of buttons) {
      const name = b.getAttribute("aria-label") ?? b.getAttribute("title") ?? b.textContent?.trim();
      expect(name, `button has no accessible name: ${b.outerHTML}`).toBeTruthy();
    }
  });
});
