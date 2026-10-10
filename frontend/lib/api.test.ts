import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import axios from "axios";
import { refreshSession } from "./api";

/**
 * lib/api.ts coalesces concurrent session refreshes (single-flight):
 * two callers racing to refresh must trigger exactly ONE HTTP request.
 */
describe("refreshSession single-flight", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("coalesces concurrent refreshes into one HTTP call", async () => {
    const gate = Promise.withResolvers<void>();
    const postSpy = vi.spyOn(axios, "post").mockImplementation(async () => {
      await gate.promise;
      return { data: { success: true } };
    });

    const a = refreshSession();
    const b = refreshSession();

    expect(postSpy).toHaveBeenCalledTimes(1);

    gate.resolve();
    await Promise.all([a, b]);

    expect(postSpy).toHaveBeenCalledTimes(1);
  });

  it("issues a new refresh after the previous one settles", async () => {
    const postSpy = vi.spyOn(axios, "post").mockResolvedValue({ data: { success: true } });

    await refreshSession();
    await refreshSession();

    expect(postSpy).toHaveBeenCalledTimes(2);
  });
});
