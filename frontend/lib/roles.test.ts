import { describe, it, expect } from "vitest";
import {
  normalizeRole,
  canAccessAdmin,
  isSuperAdmin,
  homePathForRole,
  decideRoute,
  type Role,
  type DecideRouteAction,
} from "./roles";

describe("normalizeRole", () => {
  const cases: Array<{ input: unknown; expected: Role }> = [
    { input: "super_admin", expected: "super_admin" },
    { input: "admin", expected: "admin" },
    { input: "user", expected: "user" },
    { input: "unknown", expected: "user" },
    { input: null, expected: "user" },
    { input: undefined, expected: "user" },
    { input: 123, expected: "user" },
    { input: {}, expected: "user" },
  ];

  cases.forEach(({ input, expected }) => {
    it(`normalizes ${JSON.stringify(input)} to ${expected}`, () => {
      expect(normalizeRole(input)).toBe(expected);
    });
  });
});

describe("canAccessAdmin", () => {
  it("allows admin", () => {
    expect(canAccessAdmin("admin")).toBe(true);
  });

  it("allows super_admin", () => {
    expect(canAccessAdmin("super_admin")).toBe(true);
  });

  it("rejects user", () => {
    expect(canAccessAdmin("user")).toBe(false);
  });
});

describe("isSuperAdmin", () => {
  it("true for super_admin", () => {
    expect(isSuperAdmin("super_admin")).toBe(true);
  });

  it("false for admin", () => {
    expect(isSuperAdmin("admin")).toBe(false);
  });

  it("false for user", () => {
    expect(isSuperAdmin("user")).toBe(false);
  });
});

describe("homePathForRole", () => {
  it("routes super_admin to /super-admin", () => {
    expect(homePathForRole("super_admin")).toBe("/super-admin");
  });

  it("routes admin to /admin", () => {
    expect(homePathForRole("admin")).toBe("/admin");
  });

  it("routes user to /user", () => {
    expect(homePathForRole("user")).toBe("/user");
  });
});

describe("decideRoute", () => {
  const cases: Array<{
    name: string;
    pathname: string;
    role?: Role;
    valid?: boolean;
    hasSession?: boolean;
    hasRefresh?: boolean;
    expected: DecideRouteAction;
  }> = [
    {
      name: "anon on protected route -> redirect-login",
      pathname: "/admin/users",
      hasSession: false,
      valid: false,
      expected: "redirect-login",
    },
    {
      name: "anon on public page -> pass",
      pathname: "/login",
      hasSession: false,
      valid: false,
      expected: "pass",
    },
    {
      name: "super_admin on /admin route -> pass",
      pathname: "/admin/users",
      role: "super_admin",
      valid: true,
      expected: "pass",
    },
    {
      name: "super_admin on /super-admin -> pass",
      pathname: "/super-admin/plans",
      role: "super_admin",
      valid: true,
      expected: "pass",
    },
    {
      name: "user on /super-admin -> redirect to /user",
      pathname: "/super-admin/plans",
      role: "user",
      valid: true,
      expected: { redirect: "/user" },
    },
    {
      name: "admin on /super-admin -> redirect to /admin",
      pathname: "/super-admin/plans",
      role: "admin",
      valid: true,
      expected: { redirect: "/admin" },
    },
    {
      name: "admin on /user -> redirect to /admin",
      pathname: "/user/dashboard",
      role: "admin",
      valid: true,
      expected: { redirect: "/admin" },
    },
    {
      name: "super_admin on /user -> redirect to /super-admin",
      pathname: "/user/dashboard",
      role: "super_admin",
      valid: true,
      expected: { redirect: "/super-admin" },
    },
    {
      name: "user on /admin -> redirect to /user",
      pathname: "/admin",
      role: "user",
      valid: true,
      expected: { redirect: "/user" },
    },
    {
      name: "user on /user -> pass",
      pathname: "/user/transactions",
      role: "user",
      valid: true,
      expected: "pass",
    },
    {
      name: "authenticated user on public page / -> pass",
      pathname: "/",
      role: "user",
      valid: true,
      expected: "pass",
    },
    {
      name: "authenticated super_admin on /login -> redirect to /super-admin",
      pathname: "/login",
      role: "super_admin",
      valid: true,
      expected: { redirect: "/super-admin" },
    },
    {
      name: "authenticated admin on /login -> redirect to /admin",
      pathname: "/login",
      role: "admin",
      valid: true,
      expected: { redirect: "/admin" },
    },
    {
      name: "anon with refresh cookie on protected route -> pass (client refreshes)",
      pathname: "/super-admin",
      hasSession: false,
      hasRefresh: true,
      valid: false,
      expected: "pass",
    },
    {
      name: "stale access token with session on protected route -> pass",
      pathname: "/super-admin",
      hasSession: true,
      role: "user",
      valid: false,
      expected: "pass",
    },
    {
      name: "expired token on /login -> pass (client refreshes)",
      pathname: "/login",
      hasSession: true,
      hasRefresh: true,
      role: "user",
      valid: false,
      expected: "pass",
    },
  ];

  cases.forEach(({ name, pathname, role, valid, hasSession = true, hasRefresh, expected }) => {
    it(name, () => {
      expect(decideRoute({ pathname, role, valid, hasSession, hasRefresh })).toEqual(expected);
    });
  });
});
