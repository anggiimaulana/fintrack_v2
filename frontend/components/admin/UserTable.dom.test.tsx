import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { UserTable, type UserRow } from "./UserTable"
import type { UseDataTableUrlState } from "@/hooks/use-data-table-url-state"

const mockTableState: UseDataTableUrlState = {
  page: 2,
  perPage: 20,
  sort: "created_at",
  order: "desc",
  sorts: [{ column: "created_at", order: "desc" }],
  q: "",
  filters: {},
  setPage: vi.fn(),
  setPerPage: vi.fn(),
  setSort: vi.fn(),
  setSorts: vi.fn(),
  setSearch: vi.fn(),
  setFilter: vi.fn(),
  setFilters: vi.fn(),
  reset: vi.fn(),
  immediate: {
    page: 2,
    perPage: 20,
    sort: "created_at",
    order: "desc",
    sorts: [{ column: "created_at", order: "desc" }],
    q: "",
    filters: {},
  },
  debounced: {
    page: 2,
    perPage: 20,
    sort: "created_at",
    order: "desc",
    sorts: [{ column: "created_at", order: "desc" }],
    q: "",
    filters: {},
  },
}

const mockUsers: UserRow[] = [
  {
    id: "u-1",
    full_name: "Budi Santoso",
    email: "budi@example.com",
    created_at: "2026-10-15T10:00:00Z",
    is_active: true,
    total_transactions: 14,
  },
  {
    id: "u-2",
    full_name: "Siti Rahma",
    email: "siti@example.com",
    created_at: "2026-10-14T09:00:00Z",
    is_active: false,
    total_transactions: 3,
  },
]

describe("UserTable", () => {
  it("renders absolute row numbering based on page and perPage e.g. row 0 on page 2 = 21", () => {
    render(
      <UserTable
        users={mockUsers}
        total={42}
        totalPages={3}
        tableState={mockTableState}
        onToggleStatus={vi.fn()}
      />
    )

    // On page 2 with perPage 20, index 0 is 21, index 1 is 22
    expect(screen.getByText("21")).toBeInTheDocument()
    expect(screen.getByText("22")).toBeInTheDocument()
  })

  it("renders user information and status badges accurately", () => {
    render(
      <UserTable
        users={mockUsers}
        total={42}
        totalPages={3}
        tableState={mockTableState}
        onToggleStatus={vi.fn()}
      />
    )

    expect(screen.getByText("Budi Santoso")).toBeInTheDocument()
    expect(screen.getByText("budi@example.com")).toBeInTheDocument()
    expect(screen.getByText("Aktif")).toBeInTheDocument()
    expect(screen.getByText("Suspended")).toBeInTheDocument()
  })

  it("triggers onToggleStatus when Suspend or Aktifkan action button is clicked", () => {
    const handleToggle = vi.fn()
    render(
      <UserTable
        users={mockUsers}
        total={42}
        totalPages={3}
        tableState={mockTableState}
        onToggleStatus={handleToggle}
      />
    )

    const suspendBtn = screen.getByRole("button", { name: /Suspend/i })
    fireEvent.click(suspendBtn)

    expect(handleToggle).toHaveBeenCalledWith(mockUsers[0])
  })

  it("triggers tableState.setSort when sortable header is clicked", () => {
    render(
      <UserTable
        users={mockUsers}
        total={42}
        totalPages={3}
        tableState={mockTableState}
        onToggleStatus={vi.fn()}
      />
    )

    const nameHeader = screen.getByRole("button", { name: /Nama Lengkap/i })
    fireEvent.click(nameHeader)

    expect(mockTableState.setSort).toHaveBeenCalledWith("full_name")
  })
})
