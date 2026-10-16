import { render, screen } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import { DashboardShell } from "./DashboardShell"
import * as sessionContext from "@/lib/session-context"
import { useRouter } from "next/navigation"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("next/navigation", () => ({
  useRouter: vi.fn(),
  usePathname: vi.fn(() => "/admin"),
  useSearchParams: vi.fn(() => new URLSearchParams()),
}))

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

interface MockRouter {
  replace: (href: string) => void
  push: (href: string) => void
  refresh: () => void
  back: () => void
  forward: () => void
  prefetch: (href: string) => void
}

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>
  )
}

describe("DashboardShell", () => {
  const mockReplace = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useRouter).mockReturnValue({
      replace: mockReplace,
      push: vi.fn(),
      refresh: vi.fn(),
      back: vi.fn(),
      forward: vi.fn(),
      prefetch: vi.fn(),
    } as unknown as MockRouter)
  })

  it("renders sidebar and main content children", () => {
    vi.spyOn(sessionContext, "useSession").mockReturnValue({
      user: {
        id: "1",
        email: "admin@test.com",
        full_name: "Admin User",
        role: "admin",
        is_active: true,
      },
      isLoading: false,
      isAuthenticated: true,
      isAdmin: true,
      isSuperAdmin: false,
    })

    renderWithClient(
      <DashboardShell
        sidebar={<div data-testid="sidebar">Sidebar Content</div>}
      >
        <div data-testid="content">Main Body Content</div>
      </DashboardShell>
    )

    expect(screen.getByTestId("sidebar")).toBeInTheDocument()
    expect(screen.getByTestId("content")).toBeInTheDocument()
  })

  it("redirects to /user when user lacks required super_admin role", () => {
    vi.spyOn(sessionContext, "useSession").mockReturnValue({
      user: {
        id: "1",
        email: "regular@test.com",
        full_name: "Regular User",
        role: "user",
        is_active: true,
      },
      isLoading: false,
      isAuthenticated: true,
      isAdmin: false,
      isSuperAdmin: false,
    })

    renderWithClient(
      <DashboardShell
        sidebar={<div>Sidebar</div>}
        requiredRole="super_admin"
      >
        <div>Protected Content</div>
      </DashboardShell>
    )

    expect(mockReplace).toHaveBeenCalledWith("/user")
  })

  it("redirects to /login when user account is inactive/suspended", () => {
    vi.spyOn(sessionContext, "useSession").mockReturnValue({
      user: {
        id: "1",
        email: "banned@test.com",
        full_name: "Banned User",
        role: "user",
        is_active: false,
      },
      isLoading: false,
      isAuthenticated: true,
      isAdmin: false,
      isSuperAdmin: false,
    })

    renderWithClient(
      <DashboardShell
        sidebar={<div>Sidebar</div>}
      >
        <div>Content</div>
      </DashboardShell>
    )

    expect(mockReplace).toHaveBeenCalledWith("/login")
  })

  it("renders with role=user and bottomNav without authorization errors", () => {
    vi.spyOn(sessionContext, "useSession").mockReturnValue({
      user: {
        id: "1",
        email: "user@test.com",
        full_name: "Normal User",
        role: "user",
        is_active: true,
      },
      isLoading: false,
      isAuthenticated: true,
      isAdmin: false,
      isSuperAdmin: false,
    })

    renderWithClient(
      <DashboardShell
        sidebar={<div data-testid="sidebar">Sidebar</div>}
        bottomNav={<nav data-testid="bottom-nav">Bottom Nav</nav>}
        role="user"
      >
        <div data-testid="user-content">User Page</div>
      </DashboardShell>
    )

    expect(screen.getByTestId("sidebar")).toBeInTheDocument()
    expect(screen.getByTestId("bottom-nav")).toBeInTheDocument()
    expect(screen.getByTestId("user-content")).toBeInTheDocument()
    expect(mockReplace).not.toHaveBeenCalled()
  })
})
