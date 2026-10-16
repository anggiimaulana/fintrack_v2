import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { ConfirmActionModal } from "./ConfirmActionModal"

describe("ConfirmActionModal", () => {
  it("renders title, description, and entity name when open", () => {
    render(
      <ConfirmActionModal
        open={true}
        onOpenChange={vi.fn()}
        actionType="delete"
        title="Konfirmasi Hapus"
        description="Data akan dihapus permanen"
        entityName="Paket Pro Tahunan"
        onConfirm={vi.fn()}
      />
    )

    expect(screen.getByText("Konfirmasi Hapus")).toBeInTheDocument()
    expect(screen.getByText("Data akan dihapus permanen")).toBeInTheDocument()
    expect(screen.getByText("Paket Pro Tahunan")).toBeInTheDocument()
  })

  it("calls onConfirm when confirm button clicked", () => {
    const handleConfirm = vi.fn()
    render(
      <ConfirmActionModal
        open={true}
        onOpenChange={vi.fn()}
        actionType="suspend"
        title="Suspend Akun"
        description="Akun akan dibekukan"
        onConfirm={handleConfirm}
      />
    )

    const confirmBtn = screen.getByRole("button", { name: "Suspend Akun" })
    fireEvent.click(confirmBtn)

    expect(handleConfirm).toHaveBeenCalledTimes(1)
  })

  it("disables buttons and shows spinner when isPending is true", () => {
    render(
      <ConfirmActionModal
        open={true}
        onOpenChange={vi.fn()}
        actionType="deactivate"
        title="Nonaktifkan"
        description="Nonaktifkan provider"
        isPending={true}
        onConfirm={vi.fn()}
      />
    )

    const confirmBtn = screen.getByRole("button", { name: /Memproses/i })
    expect(confirmBtn).toBeDisabled()
  })
})
