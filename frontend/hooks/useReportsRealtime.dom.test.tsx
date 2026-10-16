import * as React from "react";
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useReportsRealtime } from "./useReports";
import { publishDomainEvent } from "@/lib/realtime-bus";
import { queryKeys } from "@/lib/cache-keys";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useReportsRealtime", () => {
  it("invalidates reports, budgets, spk, and dashboard caches on transaction-related usage_changed", () => {
    const queryClient = new QueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { unmount } = renderHook(() => useReportsRealtime(), {
      wrapper: createWrapper(queryClient),
    });

    publishDomainEvent({
      event: "usage_changed",
      timestamp: 1234567890,
      payload: {
        limit_keys: ["max_transactions"],
        reason: "transaction_created",
      },
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.reports.all });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.budgets.all });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.spk.all });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.dashboard.all });

    unmount();
  });

  it("invalidates on stream_reconnected and quota_reset", () => {
    const queryClient = new QueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { unmount } = renderHook(() => useReportsRealtime(), {
      wrapper: createWrapper(queryClient),
    });

    publishDomainEvent({
      event: "stream_reconnected",
      timestamp: 1234567890,
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.reports.all });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.budgets.all });

    invalidateSpy.mockClear();

    publishDomainEvent({
      event: "quota_reset",
      timestamp: 1234567891,
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.reports.all });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.budgets.all });

    unmount();
  });

  it("ignores unrelated events such as payment_status_changed", () => {
    const queryClient = new QueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { unmount } = renderHook(() => useReportsRealtime(), {
      wrapper: createWrapper(queryClient),
    });

    publishDomainEvent({
      event: "payment_status_changed",
      timestamp: 1234567892,
    });

    expect(invalidateSpy).not.toHaveBeenCalled();

    unmount();
  });
});
