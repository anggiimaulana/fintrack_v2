import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useHash } from "./use-hash";

/**
 * The hash hook backs deep links such as /super-admin/settings#kuota. It must
 * return the current hash and re-render subscribers on `hashchange` without
 * writing state from an effect.
 */
describe("useHash", () => {
  beforeEach(() => {
    window.location.hash = "";
  });

  it("returns an empty string when there is no hash", () => {
    const { result } = renderHook(() => useHash());
    expect(result.current).toBe("");
  });

  it("returns the current hash without the leading #", () => {
    window.location.hash = "#kuota";
    const { result } = renderHook(() => useHash());
    expect(result.current).toBe("kuota");
  });

  it("updates when the hash changes", () => {
    window.location.hash = "#settings";
    const { result } = renderHook(() => useHash());
    expect(result.current).toBe("settings");

    act(() => {
      window.location.hash = "#kuota";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });
    expect(result.current).toBe("kuota");
  });
});
