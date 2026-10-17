import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AboutSections } from "./AboutSections";
import { ContactSections } from "./ContactSections";
import { FaqSections } from "./FaqSections";
import { ABOUT, CONTACT } from "@/lib/marketing/support";
import { FAQ_CATEGORIES } from "@/lib/marketing/faq";
import { FAQ_PAGE } from "@/lib/marketing/copy";
import { site } from "@/lib/marketing/site";

describe("AboutSections", () => {
  it("renders the H1, mission, and all four principles", () => {
    render(<AboutSections />);
    expect(screen.getByRole("heading", { level: 1, name: ABOUT.h1 })).toBeInTheDocument();
    expect(screen.getByText(ABOUT.mission.text)).toBeInTheDocument();
    ABOUT.principles.items.forEach((principle) =>
      expect(screen.getByRole("heading", { name: principle.title })).toBeInTheDocument(),
    );
  });

  it("renders the honest technical overview", () => {
    render(<AboutSections />);
    ABOUT.tech.paragraphs.forEach((paragraph) =>
      expect(screen.getByText(paragraph)).toBeInTheDocument(),
    );
  });

  it("links to the landing personas and to /contact", () => {
    render(<AboutSections />);
    expect(screen.getByRole("link", { name: new RegExp(ABOUT.audience.link.label) })).toHaveAttribute(
      "href",
      "/#who-its-for",
    );
    expect(screen.getByRole("link", { name: new RegExp(ABOUT.contact.link.label) })).toHaveAttribute(
      "href",
      "/contact",
    );
  });

  it("never invents a team size or makes unverifiable claims", () => {
    const { container } = render(<AboutSections />);
    const text = (container.textContent ?? "").toLowerCase();
    ["karyawan", "pendanaan", "investor", "terbaik di", "nomor 1"].forEach((claim) =>
      expect(text).not.toContain(claim),
    );
  });
});

describe("ContactSections", () => {
  it("renders the H1 and both support channels", () => {
    render(<ContactSections />);
    expect(screen.getByRole("heading", { level: 1, name: CONTACT.h1 })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: CONTACT.channels.email.title })).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: CONTACT.channels.whatsapp.title }),
    ).toBeInTheDocument();
    expect(screen.getByText(site.email)).toBeInTheDocument();
    expect(screen.getByText(site.whatsappDisplay)).toBeInTheDocument();
  });

  it("builds a mailto: link with the site address", () => {
    render(<ContactSections />);
    const email = screen.getByRole("link", { name: CONTACT.channels.email.ariaLabel });
    expect(email.getAttribute("href")).toMatch(/^mailto:halo@fintrack\.id\?subject=/);
  });

  it("builds a safe external wa.me link", () => {
    render(<ContactSections />);
    const whatsapp = screen.getByRole("link", { name: CONTACT.channels.whatsapp.ariaLabel });
    expect(whatsapp.getAttribute("href")).toMatch(/^https:\/\/wa\.me\/6281200000000\?text=/);
    expect(whatsapp).toHaveAttribute("target", "_blank");
    expect(whatsapp).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("offers the pre-contact shortcuts and the account note", () => {
    render(<ContactSections />);
    CONTACT.before.links.forEach((link) =>
      expect(screen.getByRole("link", { name: link.label })).toHaveAttribute("href", link.href),
    );
    expect(screen.getByRole("heading", { name: CONTACT.account.title })).toBeInTheDocument();
  });

  it("never mentions bank sync", () => {
    const { container } = render(<ContactSections />);
    expect(container.textContent ?? "").not.toMatch(/bank_sync|sinkronisasi bank/i);
  });
});

describe("FaqSections", () => {
  it("renders every category with its questions", () => {
    render(<FaqSections />);
    expect(screen.getByRole("heading", { level: 1, name: FAQ_PAGE.h1 })).toBeInTheDocument();
    FAQ_CATEGORIES.forEach((category) => {
      expect(screen.getByRole("heading", { name: category.title })).toBeInTheDocument();
      category.items.forEach((item) => expect(screen.getByText(item.q)).toBeInTheDocument());
    });
  });

  it("anchors the table of contents to each category section", () => {
    const { container } = render(<FaqSections />);
    FAQ_CATEGORIES.forEach((category) => {
      expect(screen.getByRole("link", { name: category.title })).toHaveAttribute(
        "href",
        `#${category.id}`,
      );
      expect(container.querySelector(`section#${category.id}`)).not.toBeNull();
    });
  });

  it("renders answers as plain content (details/summary), never as testimonial", () => {
    const { container } = render(<FaqSections />);
    expect(container.querySelectorAll("details").length).toBeGreaterThanOrEqual(
      FAQ_CATEGORIES.reduce((total, category) => total + category.items.length, 0),
    );
    expect(container.textContent ?? "").not.toMatch(/testimoni|kata pengguna/i);
  });

  it("links to /contact when the answer is missing", () => {
    render(<FaqSections />);
    expect(screen.getByRole("link", { name: new RegExp(FAQ_PAGE.contactCta.label) })).toHaveAttribute(
      "href",
      "/contact",
    );
  });
});
