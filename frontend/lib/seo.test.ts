import { describe, it, expect } from "vitest";
import {
  absoluteUrl,
  buildPageMetadata,
  organizationGraph,
  productJsonLd,
  breadcrumbJsonLd,
  aboutPageJsonLd,
  contactPageJsonLd,
  webPageJsonLd,
} from "./seo";
import { site } from "./marketing/site";
import type { PublicPlan } from "./plans-types";

function makePlan(overrides: Partial<PublicPlan> = {}): PublicPlan {
  return {
    code: "free",
    name: "Free",
    price_monthly_idr: 0,
    price_yearly_idr: 0,
    billing_period_days: 30,
    is_public: true,
    sort_order: 1,
    fallback_gateway: "manual",
    features: {},
    limits: {},
    ...overrides,
  };
}

describe("absoluteUrl", () => {
  it("joins a site-relative path onto the origin", () => {
    expect(absoluteUrl("/pricing")).toBe(`${site.url}/pricing`);
  });

  it("passes absolute urls through", () => {
    expect(absoluteUrl("https://example.com/x")).toBe("https://example.com/x");
  });
});

describe("buildPageMetadata", () => {
  it("sets title (absolute), description, canonical, and absolute OG url", () => {
    const meta = buildPageMetadata({
      title: "Harga FinTrack",
      description: "Deskripsi.",
      path: "/pricing",
    });
    expect((meta.title as { absolute: string }).absolute).toBe("Harga FinTrack");
    expect(meta.description).toBe("Deskripsi.");
    expect(meta.alternates?.canonical).toBe("/pricing");
    expect((meta.openGraph as { url?: string }).url).toBe(`${site.url}/pricing`);
    expect((meta.twitter as { card?: string }).card).toBe("summary_large_image");
  });
});

describe("organizationGraph", () => {
  it("emits Organization + WebSite with a logo and support email", () => {
    const node = organizationGraph();
    const graph = node["@graph"] as Array<Record<string, unknown>>;
    const org = graph.find((n) => n["@type"] === "Organization");
    const website = graph.find((n) => n["@type"] === "WebSite");
    expect(org?.name).toBe("FinTrack");
    expect(org?.logo).toBe(`${site.url}/logo.png`);
    expect(website?.inLanguage).toBe("id-ID");
  });

  it("does not add a SearchAction (no internal search)", () => {
    expect(JSON.stringify(organizationGraph())).not.toContain("SearchAction");
  });
});

describe("productJsonLd", () => {
  it("builds one Offer per plan, ordered, with IDR string prices", () => {
    const plans = [
      makePlan({ code: "pro", name: "Pro", price_monthly_idr: 39000, sort_order: 2 }),
      makePlan({ code: "free", name: "Free", price_monthly_idr: 0, sort_order: 1 }),
    ];
    const node = productJsonLd(plans);
    expect(node["@type"]).toBe("Product");
    const offers = node.offers as Array<Record<string, unknown>>;
    expect(offers).toHaveLength(2);
    expect(offers[0].name).toBe("Free");
    expect(offers[0].price).toBe("0");
    expect(offers[1].price).toBe("39000");
    offers.forEach((o) => expect(o.priceCurrency).toBe("IDR"));
  });
});

describe("breadcrumbJsonLd", () => {
  it("numbers itemListElement from 1 and uses absolute urls", () => {
    const node = breadcrumbJsonLd([
      { name: "Beranda", path: "/" },
      { name: "Fitur", path: "/features" },
    ]);
    const items = node.itemListElement as Array<Record<string, unknown>>;
    expect(items).toHaveLength(2);
    expect(items[0].position).toBe(1);
    expect(items[1].item).toBe(`${site.url}/features`);
  });
});

describe("page-type nodes", () => {
  it("builds AboutPage, ContactPage, and WebPage with id-ID language", () => {
    expect(aboutPageJsonLd()["@type"]).toBe("AboutPage");
    expect(contactPageJsonLd()["@type"]).toBe("ContactPage");
    expect(webPageJsonLd("FAQ", "/faq", "x")["@type"]).toBe("WebPage");
    expect(webPageJsonLd("FAQ", "/faq", "x").inLanguage).toBe("id-ID");
  });
});

describe("forbidden structured data", () => {
  it("never emits FAQPage, Review, or AggregateRating", () => {
    const blob = JSON.stringify([
      organizationGraph(),
      productJsonLd([makePlan()]),
      breadcrumbJsonLd([{ name: "a", path: "/a" }]),
      aboutPageJsonLd(),
      contactPageJsonLd(),
      webPageJsonLd("x", "/x", "y"),
    ]);
    ["FAQPage", "AggregateRating", "Review"].forEach((banned) => {
      expect(blob).not.toContain(banned);
    });
  });
});
