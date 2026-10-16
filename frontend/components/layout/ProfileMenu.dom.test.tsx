import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { axe } from "@/test/setup";
import { ProfileMenu } from "./ProfileMenu";

vi.mock("@/lib/session-context", () => ({
  useSession: () => ({
    user: { full_name: "Budi Santoso", email: "budi@example.com", avatar_url: "" },
    isLoading: false,
    isAuthenticated: true,
    isAdmin: false,
    isSuperAdmin: false,
  }),
}));

vi.mock("@/hooks/use-entitlements", () => ({
  useEntitlements: () => ({
    planName: "Pro",
    planCode: "pro",
    status: "active",
  }),
}));

/** Radix menus open on pointerdown, not click. */
function openMenu(label: string) {
  const trigger = screen.getByLabelText(label);
  fireEvent.pointerDown(trigger, { button: 0, ctrlKey: false, pointerType: "mouse" });
  return trigger;
}

describe("ProfileMenu", () => {
  it("opens a menu with the user's identity and links", () => {
    render(<ProfileMenu onLogout={() => {}} />);
    openMenu("Menu profil");
    expect(screen.getByText("Budi Santoso")).toBeInTheDocument();
    expect(screen.getByText("budi@example.com")).toBeInTheDocument();
    expect(screen.getByText("Profil & Pengaturan")).toBeInTheDocument();
    expect(screen.getByText("Pro")).toBeInTheDocument();
    expect(screen.getByText(/Langganan/)).toBeInTheDocument();
  });

  it("links to the merged profile route", () => {
    render(<ProfileMenu onLogout={() => {}} />);
    openMenu("Menu profil");
    expect(screen.getByText("Profil & Pengaturan").closest("a")).toHaveAttribute("href", "/user/profile");
  });

  it("invokes onLogout when Keluar is chosen", () => {
    const onLogout = vi.fn();
    render(<ProfileMenu onLogout={onLogout} />);
    openMenu("Menu profil");
    fireEvent.click(screen.getByText("Keluar"));
    expect(onLogout).toHaveBeenCalled();
  });

  it("has no accessibility violations when open", async () => {
    const { container } = render(<ProfileMenu onLogout={() => {}} />);
    openMenu("Menu profil");
    expect(screen.getByText("Profil & Pengaturan")).toBeInTheDocument();
    expect(await axe(container)).toHaveNoViolations();
  });
});
