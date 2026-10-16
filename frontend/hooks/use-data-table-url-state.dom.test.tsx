import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useDataTableUrlState } from "./use-data-table-url-state";

const mockReplace = vi.fn();
let currentSearch = "";

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    replace: mockReplace,
  }),
  usePathname: () => "/test-table",
  useSearchParams: () => new URLSearchParams(currentSearch),
}));

describe("useDataTableUrlState debounce", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockReplace.mockClear();
    currentSearch = "";
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("updates immediate state instantly and debounces the commit and debounced state", () => {
    const { result } = renderHook(() => useDataTableUrlState({ debounceMs: 300 }));

    act(() => {
      result.current.setFilter("type", "expense");
    });

    // Immediate state reflects change instantly
    expect(result.current.filters.type).toBe("expense");
    expect(result.current.immediate.filters.type).toBe("expense");
    // Debounced state is not yet updated
    expect(result.current.debounced.filters.type).toBeUndefined();
    expect(mockReplace).not.toHaveBeenCalled();

    // Advance halfway
    act(() => {
      vi.advanceTimersByTime(150);
    });
    expect(result.current.debounced.filters.type).toBeUndefined();
    expect(mockReplace).not.toHaveBeenCalled();

    // Advance remainder
    act(() => {
      vi.advanceTimersByTime(150);
    });
    expect(result.current.debounced.filters.type).toBe("expense");
    expect(mockReplace).toHaveBeenCalledWith("/test-table?filter_type=expense", { scroll: false });
  });

  it("resets debounce timer on rapid sort changes", () => {
    const { result } = renderHook(() => useDataTableUrlState({ debounceMs: 300 }));

    act(() => {
      result.current.setSort("amount", "asc");
    });
    expect(result.current.sort).toBe("amount");
    expect(result.current.order).toBe("asc");
    expect(result.current.debounced.sort).toBe("");

    act(() => {
      vi.advanceTimersByTime(150);
    });

    // User clicks again to invert sort
    act(() => {
      result.current.setSort("amount", "desc");
    });
    expect(result.current.sort).toBe("amount");
    expect(result.current.order).toBe("desc");

    act(() => {
      vi.advanceTimersByTime(150);
    });
    // Still not fired because timer reset
    expect(result.current.debounced.sort).toBe("");
    act(() => {
      vi.advanceTimersByTime(150);
    });
    expect(result.current.debounced.sort).toBe("amount");
    expect(result.current.debounced.order).toBe("desc");
    expect(mockReplace).toHaveBeenCalledTimes(1);
    expect(mockReplace).toHaveBeenCalledWith("/test-table?sort=amount&order=desc", { scroll: false });
  });

  it("executes page change immediately without debounce delay", () => {
    const { result } = renderHook(() => useDataTableUrlState({ debounceMs: 300 }));

    act(() => {
      result.current.setPage(3);
    });

    expect(result.current.page).toBe(3);
    expect(result.current.debounced.page).toBe(3);
    expect(mockReplace).toHaveBeenCalledWith("/test-table?page=3", { scroll: false });
  });
});
