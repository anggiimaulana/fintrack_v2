import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { Sidebar } from "./Sidebar";

vi.mock("next/navigation", () => ({
  usePathname: () => "/user/usage",
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}));

vi.mock("@/lib/session-context", () => ({
  useSession: () => ({
    user: { full_name: "Anggi Maulana", email: "anggi@example.com", avatar_url: "", role: "user" },
    isLoading: false,
    isAuthenticated: true,
  }),
}));

vi.mock("@/hooks/use-sidebar-collapsed", () => ({
  useSidebarCollapsed: () => ({ collapsed: false, toggle: vi.fn(), setCollapsed: vi.fn() }),
}));

vi.mock("@/hooks/use-entitlements", () => ({
  useEntitlements: () => ({
    planName: "Pro",
    planCode: "pro",
    status: "active",
  }),
}));

describe("Sidebar", () => {
  it("renders user name and active plan dynamically", () => {
    render(<Sidebar />);
    expect(screen.getByText("Anggi Maulana")).toBeInTheDocument();
    expect(screen.getByText("Pro")).toBeInTheDocument();
    expect(screen.queryByText(/^USER$/i)).not.toBeInTheDocument();
  });

  it("marks active nav item with primary background and primary-foreground text", () => {
    render(<Sidebar />);
    const activeLink = screen.getByText("Penggunaan").closest("a");
    expect(activeLink).toHaveClass("bg-primary");
    expect(activeLink).toHaveClass("text-primary-foreground");
  });
});
