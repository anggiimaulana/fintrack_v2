import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { SavedCardsCard } from "./SavedCardsCard";
import type { SavedCard } from "@/hooks/use-billing";

const mockUseMyCards = vi.fn();
const mockDeleteMutateAsync = vi.fn();
const mockSetDefaultMutateAsync = vi.fn();

vi.mock("@/hooks/use-billing", () => ({
  useMyCards: () => mockUseMyCards(),
  useDeleteCard: () => ({
    mutateAsync: mockDeleteMutateAsync,
    isPending: false,
  }),
  useSetDefaultCard: () => ({
    mutateAsync: mockSetDefaultMutateAsync,
    isPending: false,
  }),
}));

describe("SavedCardsCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders empty state when user has no stored cards", () => {
    mockUseMyCards.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });

    render(<SavedCardsCard />);

    expect(screen.getByText("Belum ada kartu tersimpan")).toBeInTheDocument();
    expect(screen.getByText("Perpanjangan Manual")).toBeInTheDocument();
  });

  it("renders stored cards with default badge and auto-charge indicator", () => {
    const cards: SavedCard[] = [
      {
        id: "card-1",
        gateway: "midtrans",
        masked_card: "481111-1114",
        card_type: "credit",
        bank: "bca",
        is_default: true,
        created_at: "2026-10-15T10:00:00Z",
      },
      {
        id: "card-2",
        gateway: "midtrans",
        masked_card: "520000-2222",
        card_type: "credit",
        bank: "mandiri",
        is_default: false,
        created_at: "2026-10-14T10:00:00Z",
      },
    ];

    mockUseMyCards.mockReturnValue({
      data: cards,
      isLoading: false,
      isError: false,
    });

    render(<SavedCardsCard />);

    expect(screen.getByText("Auto-Charge Aktif")).toBeInTheDocument();
    expect(screen.getByText("•••• 1114")).toBeInTheDocument();
    expect(screen.getByText("•••• 2222")).toBeInTheDocument();
    expect(screen.getByText("Utama")).toBeInTheDocument();
    expect(screen.getByText("Jadikan Utama")).toBeInTheDocument();
  });

  it("allows setting a non-default card as default", async () => {
    const cards: SavedCard[] = [
      {
        id: "card-1",
        gateway: "midtrans",
        masked_card: "481111-1114",
        is_default: true,
        bank: "bca",
        created_at: "2026-10-15T10:00:00Z",
      },
      {
        id: "card-2",
        gateway: "midtrans",
        masked_card: "520000-2222",
        is_default: false,
        bank: "mandiri",
        created_at: "2026-10-14T10:00:00Z",
      },
    ];

    mockUseMyCards.mockReturnValue({
      data: cards,
      isLoading: false,
      isError: false,
    });

    render(<SavedCardsCard />);

    const setDefaultBtn = screen.getByText("Jadikan Utama");
    fireEvent.click(setDefaultBtn);

    expect(mockSetDefaultMutateAsync).toHaveBeenCalledWith("card-2");
  });

  it("opens confirmation dialog and deletes card on confirm", async () => {
    const cards: SavedCard[] = [
      {
        id: "card-1",
        gateway: "midtrans",
        masked_card: "481111-1114",
        is_default: true,
        bank: "bca",
        created_at: "2026-10-15T10:00:00Z",
      },
    ];

    mockUseMyCards.mockReturnValue({
      data: cards,
      isLoading: false,
      isError: false,
    });

    render(<SavedCardsCard />);

    const deleteBtn = screen.getByLabelText("Hapus kartu 481111-1114");
    fireEvent.click(deleteBtn);

    expect(screen.getByText("Hapus Kartu Tersimpan")).toBeInTheDocument();
    const confirmBtn = screen.getByText("Hapus Kartu");
    fireEvent.click(confirmBtn);

    await waitFor(() => {
      expect(mockDeleteMutateAsync).toHaveBeenCalledWith("card-1");
    });
  });
});
