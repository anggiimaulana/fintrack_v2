import { describe, it, expect } from "vitest";
import { ABOUT, CONTACT } from "./support";
import { site } from "./site";

describe("ABOUT", () => {
  it("states the mission without asterisks or hype", () => {
    expect(ABOUT.mission.text).toContain("Membuat pencatatan keuangan semudah mengobrol");
    expect(ABOUT.mission.text).toContain("tanpa mengorbankan privasi dan kejelasan");
  });

  it("lists the four product principles", () => {
    expect(ABOUT.principles.items).toHaveLength(4);
    const titles = ABOUT.principles.items.map((item) => item.title);
    expect(titles).toEqual(["Akurasi", "Privasi", "Transparansi kuota", "Kendali data"]);
    ABOUT.principles.items.forEach((item) => expect(item.text.length).toBeGreaterThan(30));
  });

  it("describes the real architecture and the central AI gateway", () => {
    const tech = ABOUT.tech.paragraphs.join(" ");
    expect(tech).toMatch(/PostgreSQL/);
    expect(tech).toMatch(/gateway AI terpusat/i);
  });

  it("links the audience section to the landing personas and to /contact", () => {
    expect(ABOUT.audience.link.href).toBe("/#who-its-for");
    expect(ABOUT.contact.link.href).toBe("/contact");
  });

  it("never invents company history, headcount, or funding", () => {
    const text = JSON.stringify(ABOUT).toLowerCase();
    ["didirikan pada", "didirikan tahun", "jumlah karyawan", "karyawan kami", "pendanaan", "investor", "juta pengguna", "ribuan pengguna"].forEach(
      (claim) => expect(text).not.toContain(claim),
    );
  });

  it("never advertises bank sync or a bring-your-own API key", () => {
    const text = JSON.stringify(ABOUT).toLowerCase();
    ["bank_sync", "sinkronisasi bank", "api key milik", "pakai api key anda"].forEach((claim) =>
      expect(text).not.toContain(claim),
    );
  });
});

describe("CONTACT", () => {
  it("offers exactly two channels, each with an accessible action label", () => {
    expect(CONTACT.channels.email.ariaLabel).toBe("Kirim email ke tim FinTrack");
    expect(CONTACT.channels.whatsapp.ariaLabel).toBe("Chat WhatsApp dengan tim FinTrack");
    // The visible label must appear inside the accessible name (WCAG 2.5.3).
    expect(CONTACT.channels.email.ariaLabel.toLowerCase()).toContain(
      CONTACT.channels.email.ctaLabel.toLowerCase(),
    );
    expect(CONTACT.channels.whatsapp.ariaLabel.toLowerCase()).toContain(
      CONTACT.channels.whatsapp.ctaLabel.toLowerCase(),
    );
  });

  it("points the pre-contact shortcuts at real routes", () => {
    expect(CONTACT.before.links.map((link) => link.href)).toEqual([
      "/faq",
      "/pricing",
      "/features",
    ]);
  });

  it("quotes the real support window from site.ts", () => {
    expect(CONTACT.hours.text).toContain(site.supportHours);
    expect(CONTACT.hours.text).toContain(site.responseTime);
  });

  it("never advertises bank sync", () => {
    const text = JSON.stringify(CONTACT).toLowerCase();
    expect(text).not.toContain("bank_sync");
    expect(text).not.toContain("sinkronisasi bank");
  });
});
