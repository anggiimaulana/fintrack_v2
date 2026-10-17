import { describe, it, expect } from "vitest";
import robots from "./robots";

describe("app/robots.ts", () => {
  it("allows the root and disallows private prefixes", () => {
    const result = robots();
    const rules = Array.isArray(result.rules) ? result.rules : [result.rules];
    expect(rules).toHaveLength(1);
    const rule = rules[0];
    expect(rule.userAgent).toBe("*");
    expect(rule.allow).toBe("/");
    expect(rule.disallow).toEqual(["/api/", "/user", "/admin", "/super-admin", "/login"]);
  });

  it("never disallows /register or marketing pages", () => {
    const result = robots();
    const rules = Array.isArray(result.rules) ? result.rules : [result.rules];
    const disallow = (rules[0].disallow as string[]) ?? [];
    ["/register", "/pricing", "/features", "/about", "/contact", "/faq", "/privacy", "/terms"].forEach(
      (path) => expect(disallow).not.toContain(path),
    );
  });

  it("points to the canonical sitemap and host", () => {
    const result = robots();
    expect(result.sitemap).toMatch(/^https?:\/\/.+\/sitemap\.xml$/);
    expect(result.host).toMatch(/^https?:\/\/.+$/);
  });
});
