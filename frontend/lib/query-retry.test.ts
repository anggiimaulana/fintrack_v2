import { describe, it, expect } from "vitest";
import { shouldRetry } from "./query-retry";

function httpError(status: number) {
  return { response: { status } };
}

describe("shouldRetry", () => {
  it("never retries 4xx — the answer is definitive", () => {
    // A plan gate (403) or over-quota (402) must not be retried; retrying only
    // duplicated the request and delayed the upgrade CTA.
    expect(shouldRetry(0, httpError(403))).toBe(false);
    expect(shouldRetry(0, httpError(402))).toBe(false);
    expect(shouldRetry(0, httpError(401))).toBe(false);
    expect(shouldRetry(0, httpError(404))).toBe(false);
  });

  it("retries a 5xx once, then gives up", () => {
    expect(shouldRetry(0, httpError(500))).toBe(true);
    expect(shouldRetry(1, httpError(500))).toBe(false);
  });

  it("retries a network error (no response) once", () => {
    expect(shouldRetry(0, new Error("Network Error"))).toBe(true);
    expect(shouldRetry(1, new Error("Network Error"))).toBe(false);
  });
});
