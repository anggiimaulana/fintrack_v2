import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
import { axe } from "@/test/setup";
import { NotificationBell } from "./NotificationBell";
import api from "@/lib/api";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
}));

const pushMock = vi.fn();

vi.mock("@/lib/api", () => ({
  default: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

const get = vi.mocked(api.get);
const post = vi.mocked(api.post);

function makeWrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
}

/**
 * Wire the two endpoints the bell reads, using the REAL response shapes:
 *   GET /me/notifications       -> { data: <ARRAY>, meta: { total, page, per_page } }
 *   GET /me/notifications/stats -> { data: { total, unread, read } }
 * The previous mock used a `{ data: { items, unread } }` shape the backend never
 * sends, which is exactly how the bell's empty-badge bug slipped through.
 */
function mockInbox(items: unknown[], unread: number, total = items.length) {
  get.mockImplementation((url: string) => {
    if (url.startsWith("/me/notifications/stats")) {
      return Promise.resolve({ data: { data: { total, unread, read: total - unread } } });
    }
    if (url.startsWith("/me/notifications")) {
      return Promise.resolve({ data: { data: items, meta: { total, page: 1, per_page: 20 } } });
    }
    return Promise.resolve({ data: {} });
  });
}

/** Radix menus open on pointerdown, not click. */
function openBell() {
  const trigger = screen.getByLabelText(/Notifikasi/);
  fireEvent.pointerDown(trigger, { button: 0, ctrlKey: false, pointerType: "mouse" });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("NotificationBell", () => {
  it("shows an empty state when the server inbox is empty", async () => {
    mockInbox([], 0);
    const client = newClient();
    render(<NotificationBell />, { wrapper: makeWrapper(client) });
    await waitFor(() => expect(screen.getByLabelText("Notifikasi")).toBeInTheDocument());
    openBell();
    expect(await screen.findByText("Tidak ada notifikasi baru")).toBeInTheDocument();
  });

  it("renders server notifications and the unread badge", async () => {
    mockInbox(
      [
        {
          id: "n1",
          event: "payment_received",
          title: "Pembayaran diterima",
          body: "Pembayaran Anda sebesar Rp 39.000 telah kami terima.",
          href: "/user/subscription?tab=payment",
          read_at: null,
          created_at: new Date().toISOString(),
        },
      ],
      1,
    );
    const client = newClient();
    render(<NotificationBell />, { wrapper: makeWrapper(client) });
    expect(await screen.findByLabelText(/Notifikasi, 1 belum dibaca/)).toBeInTheDocument();
    openBell();
    expect(await screen.findByText("Pembayaran diterima")).toBeInTheDocument();
  });

  it("marks a notification read and navigates on click", async () => {
    mockInbox(
      [
        {
          id: "n2",
          event: "plan_updated",
          title: "Paket diperbarui",
          body: "Paket Anda kini Pro.",
          href: "/user/subscription",
          read_at: null,
          created_at: new Date().toISOString(),
        },
      ],
      1,
    );
    post.mockResolvedValue({ data: { success: true } });
    const client = newClient();
    render(<NotificationBell />, { wrapper: makeWrapper(client) });
    openBell();
    fireEvent.click(await screen.findByText("Paket diperbarui"));
    await waitFor(() => expect(post).toHaveBeenCalledWith("/me/notifications/n2/read"));
    expect(pushMock).toHaveBeenCalledWith("/user/subscription");
  });

  it("has no accessibility violations when open", async () => {
    mockInbox([], 0);
    const client = newClient();
    const { container } = render(<NotificationBell />, { wrapper: makeWrapper(client) });
    await waitFor(() => expect(screen.getByLabelText("Notifikasi")).toBeInTheDocument());
    openBell();
    expect(await axe(container)).toHaveNoViolations();
  });
});
