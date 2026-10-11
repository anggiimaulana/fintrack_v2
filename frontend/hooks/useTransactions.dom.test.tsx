import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
import { useUpdateTransaction, useDeleteTransaction, useCreateTransaction } from "./useTransactions";
import { queryKeys } from "@/lib/cache-keys";
import api from "@/lib/api";

vi.mock("@/lib/api", () => ({
  default: { post: vi.fn(), put: vi.fn(), delete: vi.fn(), get: vi.fn() },
}));

const listKey = [...queryKeys.transactions.all, { page: 1 }] as const;

function pageOf(items: { id: string; amount: number }[], total?: number) {
  return { data: items, total: total ?? items.length, page: 1, perPage: 20 };
}

function makeWrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

function newClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useUpdateTransaction (optimistic)", () => {
  it("patches the cached list before the request resolves", async () => {
    const client = newClient();
    client.setQueryData(listKey, pageOf([{ id: "t1", amount: 1000 }, { id: "t2", amount: 2000 }]));

    const deferred = Promise.withResolvers<unknown>();
    (api.put as ReturnType<typeof vi.fn>).mockReturnValue(deferred.promise);

    const { result } = renderHook(() => useUpdateTransaction(), { wrapper: makeWrapper(client) });
    const p = result.current.mutateAsync({ id: "t1", amount: 9999 });

    // The cache must already reflect the change while the request is in flight.
    await waitFor(() => {
      const cached = client.getQueryData<ReturnType<typeof pageOf>>(listKey);
      expect(cached?.data.find((t) => t.id === "t1")?.amount).toBe(9999);
    });
    // The other row is untouched.
    expect(client.getQueryData<ReturnType<typeof pageOf>>(listKey)?.data.find((t) => t.id === "t2")?.amount).toBe(2000);

    deferred.resolve({ data: { data: { id: "t1", amount: 9999 } } });
    await p;
  });

  it("rolls back to the previous snapshot on a 409 conflict", async () => {
    const client = newClient();
    client.setQueryData(listKey, pageOf([{ id: "t1", amount: 1000 }]));

    (api.put as ReturnType<typeof vi.fn>).mockRejectedValue({ response: { status: 409 } });

    const { result } = renderHook(() => useUpdateTransaction(), { wrapper: makeWrapper(client) });
    await expect(result.current.mutateAsync({ id: "t1", amount: 9999 })).rejects.toBeTruthy();

    await waitFor(() => {
      expect(client.getQueryData<ReturnType<typeof pageOf>>(listKey)?.data[0].amount).toBe(1000);
    });
  });

  it("rolls back on a plan-gate rejection (402)", async () => {
    const client = newClient();
    client.setQueryData(listKey, pageOf([{ id: "t1", amount: 1000 }]));

    (api.put as ReturnType<typeof vi.fn>).mockRejectedValue({
      response: { status: 402, data: { error: "limit_exceeded", limit_key: "max_transactions" } },
    });

    const { result } = renderHook(() => useUpdateTransaction(), { wrapper: makeWrapper(client) });
    await expect(result.current.mutateAsync({ id: "t1", amount: 5 })).rejects.toBeTruthy();

    await waitFor(() => {
      expect(client.getQueryData<ReturnType<typeof pageOf>>(listKey)?.data[0].amount).toBe(1000);
    });
  });

  it("rolls back on a generic failure", async () => {
    const client = newClient();
    client.setQueryData(listKey, pageOf([{ id: "t1", amount: 1000 }]));

    (api.put as ReturnType<typeof vi.fn>).mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useUpdateTransaction(), { wrapper: makeWrapper(client) });
    await expect(result.current.mutateAsync({ id: "t1", amount: 7 })).rejects.toBeTruthy();

    await waitFor(() => {
      expect(client.getQueryData<ReturnType<typeof pageOf>>(listKey)?.data[0].amount).toBe(1000);
    });
  });
});

describe("useDeleteTransaction (optimistic)", () => {
  it("removes the row and decrements total immediately", async () => {
    const client = newClient();
    client.setQueryData(listKey, pageOf([{ id: "t1", amount: 1000 }, { id: "t2", amount: 2000 }], 2));

    const deferred = Promise.withResolvers<unknown>();
    (api.delete as ReturnType<typeof vi.fn>).mockReturnValue(deferred.promise);

    const { result } = renderHook(() => useDeleteTransaction(), { wrapper: makeWrapper(client) });
    const p = result.current.mutateAsync("t1");

    await waitFor(() => {
      const cached = client.getQueryData<ReturnType<typeof pageOf>>(listKey);
      expect(cached?.data.map((t) => t.id)).toEqual(["t2"]);
      expect(cached?.total).toBe(1);
    });

    deferred.resolve({ data: {} });
    await p;
  });

  it("restores the removed row on failure", async () => {
    const client = newClient();
    client.setQueryData(listKey, pageOf([{ id: "t1", amount: 1000 }, { id: "t2", amount: 2000 }], 2));

    (api.delete as ReturnType<typeof vi.fn>).mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useDeleteTransaction(), { wrapper: makeWrapper(client) });
    await expect(result.current.mutateAsync("t1")).rejects.toBeTruthy();

    await waitFor(() => {
      const cached = client.getQueryData<ReturnType<typeof pageOf>>(listKey);
      expect(cached?.data.map((t) => t.id)).toEqual(["t1", "t2"]);
      expect(cached?.total).toBe(2);
    });
  });
});

describe("useCreateTransaction (deliberately not optimistic)", () => {
  it("does not touch the cache before the request resolves", async () => {
    const client = newClient();
    const initial = pageOf([{ id: "t1", amount: 1000 }]);
    client.setQueryData(listKey, initial);

    const deferred = Promise.withResolvers<unknown>();
    (api.post as ReturnType<typeof vi.fn>).mockReturnValue(deferred.promise);

    const { result } = renderHook(() => useCreateTransaction(), { wrapper: makeWrapper(client) });
    const p = result.current.mutateAsync({ amount: 500 });

    // Give any accidental optimistic write a chance to land.
    await new Promise((r) => setTimeout(r, 20));
    expect(client.getQueryData<ReturnType<typeof pageOf>>(listKey)?.data).toEqual(initial.data);

    deferred.resolve({ data: { data: { id: "t9" } } });
    await p;
  });
});
