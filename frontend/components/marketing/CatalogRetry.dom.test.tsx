import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { axe } from "vitest-axe";
import { CatalogRetry } from "./CatalogRetry";
import { MICROCOPY } from "@/lib/marketing/copy";

const refresh = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh }),
}));

describe("CatalogRetry", () => {
  it("shows the friendly message and a retry button", () => {
    render(<CatalogRetry />);
    expect(screen.getByText(MICROCOPY.catalogError)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: MICROCOPY.retry })).toBeInTheDocument();
  });

  it("re-renders the server tree on click", () => {
    refresh.mockClear();
    render(<CatalogRetry />);
    fireEvent.click(screen.getByRole("button", { name: MICROCOPY.retry }));
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("has no axe violations", async () => {
    const { container } = render(<CatalogRetry />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
