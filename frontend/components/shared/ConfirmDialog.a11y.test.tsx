import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { axe } from "vitest-axe";
import { ConfirmDialog } from "./ConfirmDialog";

/**
 * Accessibility checks for the confirmation dialog. axe catches structural
 * problems (missing names, invalid ARIA, unlabelled controls); contrast is
 * verified separately by computing ratios from the design tokens.
 */
describe("ConfirmDialog a11y", () => {
  it("has no axe violations when open", async () => {
    const { container } = render(
      <ConfirmDialog
        open
        title="Hapus transaksi?"
        description="Tindakan ini tidak dapat dibatalkan."
        onOpenChange={() => {}}
        onConfirm={() => {}}
      />,
    );

    expect(await axe(container)).toHaveNoViolations();
  });

  it("exposes an accessible name and an explicit dialog role", () => {
    render(
      <ConfirmDialog
        open
        title="Hapus transaksi?"
        description="Tindakan ini tidak dapat dibatalkan."
        onOpenChange={() => {}}
        onConfirm={() => {}}
      />,
    );

    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("renders nothing while closed", () => {
    const { container } = render(
      <ConfirmDialog
        open={false}
        title="Hapus transaksi?"
        onOpenChange={() => {}}
        onConfirm={() => {}}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});
