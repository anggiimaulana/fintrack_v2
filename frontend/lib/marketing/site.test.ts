import { describe, it, expect } from "vitest";
import { site, mailtoHref, whatsappHref } from "./site";

describe("site config", () => {
  it("has a valid absolute url", () => {
    expect(() => new URL(site.url)).not.toThrow();
  });

  it("does not keep a trailing slash on the url", () => {
    expect(site.url.endsWith("/")).toBe(false);
  });

  it("ships a usable support email", () => {
    expect(site.email).toMatch(/^[^@\s]+@[^@\s]+\.[^@\s]+$/);
  });

  it("ships a whatsapp number that is digits only", () => {
    expect(site.whatsappNumber).toMatch(/^\d+$/);
    expect(site.whatsappNumber.length).toBeGreaterThan(8);
  });

  it("exposes non-empty support copy", () => {
    expect(site.supportHours.length).toBeGreaterThan(0);
    expect(site.responseTime.length).toBeGreaterThan(0);
  });
});

describe("contact link builders", () => {
  it("builds a mailto link with an encoded subject", () => {
    const href = mailtoHref("Bantuan akun");
    expect(href.startsWith(`mailto:${site.email}?subject=`)).toBe(true);
    expect(href).toContain(encodeURIComponent("Bantuan akun"));
  });

  it("builds a wa.me link with the digits-only number", () => {
    const href = whatsappHref("Halo");
    expect(href.startsWith(`https://wa.me/${site.whatsappNumber}`)).toBe(true);
    expect(href).toContain(encodeURIComponent("Halo"));
  });
});
