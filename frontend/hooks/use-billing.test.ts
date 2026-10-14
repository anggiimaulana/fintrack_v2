import { describe, it, expect } from "vitest";
import { billingErrorMessage } from "./use-billing";

describe("billingErrorMessage", () => {
  it("prefers the backend message from a nested error envelope", () => {
    const err = { response: { data: { message: "Paket tidak ditemukan" } } };
    expect(billingErrorMessage(err, "fallback")).toBe("Paket tidak ditemukan");
  });

  it("falls back to the Error message when there is no envelope", () => {
    expect(billingErrorMessage(new Error("boom"), "fallback")).toBe("boom");
  });

  it("returns the fallback for opaque/unknown errors", () => {
    expect(billingErrorMessage({}, "fallback")).toBe("fallback");
    expect(billingErrorMessage(null, "fallback")).toBe("fallback");
    expect(billingErrorMessage(undefined, "fallback")).toBe("fallback");
    expect(billingErrorMessage({ response: { data: { message: "" } } }, "fallback")).toBe("fallback");
  });

  it("ignores a non-string message field", () => {
    const err = { response: { data: { message: 42 } } };
    expect(billingErrorMessage(err, "fallback")).toBe("fallback");
  });
});
