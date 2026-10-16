import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { SuperAdminSidebar } from "./SuperAdminSidebar";

vi.mock("next/navigation", () => ({
  usePathname: () => "/super-admin",
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}));

vi.mock("@/lib/session-context", () => ({
  useSession: () => ({
    user: { full_name: "Super Admin", email: "superadmin@fintrack.id", avatar_url: "", role: "super_admin" },
    isLoading: false,
    isAuthenticated: true,
  }),
}));

vi.mock("@/hooks/use-sidebar-collapsed", () => ({
  useSidebarCollapsed: () => ({ collapsed: false, toggle: vi.fn(), setCollapsed: vi.fn() }),
}));

vi.mock("@/hooks/use-hash", () => ({
  useHash: () => "",
}));

describe("SuperAdminSidebar", () => {
  it("renders FinTrack brand with wallet icon and hides native scrollbar with no-scrollbar", () => {
    const { container } = render(<SuperAdminSidebar />);
    expect(screen.getAllByText("Super Admin").length).toBeGreaterThanOrEqual(1);
    const brandLink = container.querySelector('a[href="/super-admin"]');
    expect(brandLink?.querySelector(".lucide-wallet")).toBeInTheDocument();
    const nav = container.querySelector("nav");
    expect(nav).toHaveClass("no-scrollbar");
  });
});
