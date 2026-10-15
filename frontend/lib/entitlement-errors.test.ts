import { describe, it, expect } from "vitest";
import { isEntitlementError, UPGRADE_URL } from "./entitlement-errors";

function apiError(status: number, body: unknown) {
  return { response: { status, data: body } };
}

describe("isEntitlementError", () => {
  it("recognises the three plan-gate error codes", () => {
    expect(isEntitlementError(apiError(403, { error: "feature_not_in_plan" }))).toBe(true);
    expect(isEntitlementError(apiError(402, { error: "limit_exceeded" }))).toBe(true);
    expect(isEntitlementError(apiError(402, { error: "limit_not_in_plan" }))).toBe(true);
  });

  it("ignores unrelated failures", () => {
    expect(isEntitlementError(apiError(500, { error: "internal" }))).toBe(false);
    expect(isEntitlementError(apiError(409, { error: "conflict" }))).toBe(false);
    expect(isEntitlementError(new Error("network"))).toBe(false);
    expect(isEntitlementError(null)).toBe(false);
    expect(isEntitlementError(undefined)).toBe(false);
  });

  it("exposes the subscription route the backend points at", () => {
    expect(UPGRADE_URL).toBe("/user/subscription");
  });
});
