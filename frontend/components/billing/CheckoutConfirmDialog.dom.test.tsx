import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { axe } from "@/test/setup";
import { CheckoutConfirmDialog } from "./CheckoutConfirmDialog";
import type { PublicPlan } from "@/hooks/use-billing";

const plan: PublicPlan = {
  code: "pro",
  name: "Pro",
  price_monthly_idr: 39000,
  price_yearly_idr: 390000,
  billing_period_days: 30,
  is_public: true,
  sort_order: 2,
  fallback_gateway: "manual",
  features: {},
  limits: {},
  tax_rate_percent: 11,
  tax_idr: 4290,
  total_idr: 43290,
};

describe("CheckoutConfirmDialog", () => {
  it("shows the price breakdown including PPN before any invoice is created", () => {
    render(
      <CheckoutConfirmDialog open onOpenChange={() => {}} plan={plan} currentPlanName="Free" onConfirm={() => {}} loading={false} />
    );
    expect(screen.getByText("Pro")).toBeInTheDocument();
    expect(screen.getByText("Free")).toBeInTheDocument();
    expect(screen.getByText(/PPN 11%/)).toBeInTheDocument();
    // Subtotal and grand total both rendered.
    expect(screen.getByText(/39\.000/)).toBeInTheDocument();
    expect(screen.getByText(/43\.290/)).toBeInTheDocument();
  });

  it("does not call onConfirm until the user clicks Lanjutkan", () => {
    const onConfirm = vi.fn();
    render(
      <CheckoutConfirmDialog open onOpenChange={() => {}} plan={plan} currentPlanName="Free" onConfirm={onConfirm} loading={false} />
    );
    // Opening alone must never trigger checkout.
    expect(onConfirm).not.toHaveBeenCalled();
    fireEvent.click(screen.getByText("Lanjutkan"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("disables both actions while the checkout is in flight", () => {
    render(
      <CheckoutConfirmDialog open onOpenChange={() => {}} plan={plan} currentPlanName="Free" onConfirm={() => {}} loading />
    );
    expect(screen.getByText("Membuat faktur…")).toBeDisabled();
    expect(screen.getByText("Batal")).toBeDisabled();
  });

  it("renders nothing when there is no plan to confirm", () => {
    const { container } = render(
      <CheckoutConfirmDialog open onOpenChange={() => {}} plan={null} currentPlanName="Free" onConfirm={() => {}} loading={false} />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("has no accessibility violations", async () => {
    const { container } = render(
      <CheckoutConfirmDialog open onOpenChange={() => {}} plan={plan} currentPlanName="Free" onConfirm={() => {}} loading={false} />
    );
    expect(await axe(container)).toHaveNoViolations();
  });
});
