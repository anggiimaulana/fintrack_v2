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
})
