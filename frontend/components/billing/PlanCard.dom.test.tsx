import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { axe } from "@/test/setup";
import { PlanCard } from "./PlanCard";
import type { PublicPlan } from "@/hooks/use-billing";

function makePlan(overrides: Partial<PublicPlan> = {}): PublicPlan {
  return {
    code: "pro",
    name: "Pro",
    price_monthly_idr: 39000,
    price_yearly_idr: 390000,
    billing_period_days: 30,
    is_public: true,
    sort_order: 2,
    fallback_gateway: "manual",
    features: { ai_chat: true },
    limits: { max_transactions: 5000 },
    ...overrides,
  };
}

const noop = () => {};

describe("PlanCard", () => {
  it("shows the tax-inclusive total when the backend supplies tax", () => {
    render(
      <PlanCard
        plan={makePlan({ tax_rate_percent: 11, tax_idr: 4290, total_idr: 43290 })}
        currentPlanCode="free"
        currentSortOrder={1}
        onUpgrade={noop}
        onDowngrade={noop}
        isBusy={false}
      />
    );
    expect(screen.getByText(/PPN 11%/)).toBeInTheDocument();
    expect(screen.getByText(/43\.290/)).toBeInTheDocument();
  });

  it("omits the PPN line when tax is zero (tax off)", () => {
    render(
      <PlanCard
        plan={makePlan({ tax_rate_percent: 0, tax_idr: 0, total_idr: 39000 })}
        currentPlanCode="free"
        currentSortOrder={1}
        onUpgrade={noop}
        onDowngrade={noop}
        isBusy={false}
      />
    );
    expect(screen.queryByText(/PPN/)).not.toBeInTheDocument();
  });

  it("does not call onUpgrade until the button is clicked", () => {
    const onUpgrade = vi.fn();
    render(
      <PlanCard
        plan={makePlan()}
        currentPlanCode="free"
        currentSortOrder={1}
        onUpgrade={onUpgrade}
        onDowngrade={noop}
        isBusy={false}
      />
    );
    // Rendering the card must not trigger any checkout.
    expect(onUpgrade).not.toHaveBeenCalled();
    fireEvent.click(screen.getByText(/Tingkatkan ke Pro/));
    expect(onUpgrade).toHaveBeenCalledWith("pro");
  });

  it("shows the active state and no upgrade action for the current plan", () => {
    render(
      <PlanCard
        plan={makePlan()}
        currentPlanCode="pro"
        currentSortOrder={2}
        onUpgrade={noop}
        onDowngrade={noop}
        isBusy={false}
      />
    );
    expect(screen.getByText("Paket Sedang Aktif")).toBeDisabled();
    expect(screen.queryByText(/Tingkatkan/)).not.toBeInTheDocument();
  });

  it("has no accessibility violations", async () => {
    const { container } = render(
      <PlanCard
        plan={makePlan({ tax_rate_percent: 11, tax_idr: 4290, total_idr: 43290 })}
        currentPlanCode="free"
        currentSortOrder={1}
        onUpgrade={noop}
        onDowngrade={noop}
        isBusy={false}
      />
    );
    expect(await axe(container)).toHaveNoViolations();
  });
});
