import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { axe } from "@/test/setup";
import { DashboardHeader } from "./DashboardHeader";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as navigation from "next/navigation";

vi.mock("next/navigation", () => ({
  useRouter: vi.fn(),
  usePathname: vi.fn(() => "/admin"),
  useSearchParams: vi.fn(() => new URLSearchParams()),
}));

vi.mock("@/lib/session-context", () => ({
  useSession: () => ({
    user: { full_name: "Admin FinTrack", email: "admin@fintrack.id", role: "admin" },
    isLoading: false,
    isAuthenticated: true,
    isAdmin: true,
    isSuperAdmin: false,
  }),
}));

vi.mock("@/providers/theme-provider", () => ({
  useTheme: () => ({
    theme: "light",
    setTheme: vi.fn(),
  }),
}));

const mockToggle = vi.fn();
vi.mock("@/hooks/use-sidebar-collapsed", () => ({
  useSidebarCollapsed: () => ({
    collapsed: false,
    toggle: mockToggle,
  }),
}));

vi.mock("@/hooks/useNotifications", () => ({
  useNotificationInbox: () => ({
    items: [],
    unread: 0,
    markAllRead: vi.fn(),
    markRead: vi.fn(),
    clear: vi.fn(),
  }),
}));

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>
  );
}

describe("DashboardHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders page title and desktop sidebar toggle", () => {
    vi.mocked(navigation.usePathname).mockReturnValue("/admin");
    renderWithClient(<DashboardHeader />);

    expect(screen.getByText("Dashboard Admin")).toBeInTheDocument();
    const toggleBtn = screen.getByLabelText("Tutup sidebar");
    expect(toggleBtn).toBeInTheDocument();

    fireEvent.click(toggleBtn);
    expect(mockToggle).toHaveBeenCalledTimes(1);
  });

  it("renders breadcrumb prefix for nested admin and super-admin routes", () => {
    vi.mocked(navigation.usePathname).mockReturnValue("/super-admin/tokens");
    renderWithClient(<DashboardHeader />);

    expect(screen.getByText("Super Admin")).toBeInTheDocument();
    expect(screen.getByText("Token & Kuota AI")).toBeInTheDocument();
  });

  it("opens the command palette dialog when Ctrl+K search button is clicked", () => {
    vi.mocked(navigation.usePathname).mockReturnValue("/admin");
    renderWithClient(<DashboardHeader />);

    const searchBtns = screen.getAllByRole("button", { name: /Cari modul atau fitur/i });
    expect(searchBtns.length).toBeGreaterThan(0);

    fireEvent.click(searchBtns[0]);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("has no accessibility violations", async () => {
    vi.mocked(navigation.usePathname).mockReturnValue("/admin");
    const { container } = renderWithClient(<DashboardHeader />);
    expect(await axe(container)).toHaveNoViolations();
  });
});
