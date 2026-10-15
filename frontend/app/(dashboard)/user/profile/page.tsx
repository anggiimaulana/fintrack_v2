"use client";

import { useEffect, useState } from "react";
import { User, Shield, Bell, Palette, ShieldAlert } from "lucide-react";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { ProfileDetails } from "@/components/profile/ProfileDetails";
import { SecuritySettings } from "@/components/profile/SecuritySettings";
import { NotificationSettings } from "@/components/profile/NotificationSettings";
import { AppearanceSettings } from "@/components/profile/AppearanceSettings";
import { PrivacySettings } from "@/components/profile/PrivacySettings";
import { useHashSection } from "@/hooks/use-hash-section";

/** The accordion sections, in display order. `id` doubles as the URL anchor. */
const SECTIONS = [
  { id: "profile", title: "Profil", desc: "Foto, nama, dan kontak Anda", icon: User, Component: ProfileDetails },
  { id: "security", title: "Keamanan", desc: "Ubah password akun", icon: Shield, Component: SecuritySettings },
  { id: "notifications", title: "Notifikasi", desc: "Preferensi notifikasi", icon: Bell, Component: NotificationSettings },
  { id: "appearance", title: "Tampilan & Bahasa", desc: "Tema dan format regional", icon: Palette, Component: AppearanceSettings },
  { id: "privacy", title: "Privasi & Data", desc: "Ekspor data dan hapus akun", icon: ShieldAlert, Component: PrivacySettings },
];

const SECTION_IDS = SECTIONS.map((s) => s.id);

/**
 * Merged profile + settings page.
 *
 * The former /user/profile and /user/settings pages are combined here as
 * accordion groups so there is a single "account" surface: shorter on desktop
 * and naturally collapsible on mobile. /user/settings redirects here.
 */
export default function ProfilePage() {
  // Deep links such as /user/profile#privacy (used by the search palette and
  // notifications) open the matching section. `selected` holds an explicit user
  // choice (including "" when collapsed); until then the hash decides.
  const hashSection = useHashSection(SECTION_IDS);
  const [selected, setSelected] = useState<string | undefined>(undefined);
  const openSection = selected ?? hashSection ?? "profile";

  // Scrolling is a DOM side effect (not state), so it is safe in an effect.
  useEffect(() => {
    if (hashSection) {
      document.getElementById(hashSection)?.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }, [hashSection]);

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto">
      <div>
        <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Profil &amp; Pengaturan</h2>
        <p className="text-xs font-bold text-muted-foreground mt-1">Kelola informasi akun dan preferensi aplikasi Anda.</p>
      </div>

      <Accordion
        type="single"
        collapsible
        value={openSection}
        onValueChange={setSelected}
        className="space-y-4"
      >
        {SECTIONS.map((section) => (
          <AccordionItem key={section.id} value={section.id} id={section.id} className="shadow-[4px_4px_0px_0px] shadow-foreground">
            <AccordionTrigger>
              <span className="flex items-center gap-3">
                <span className="w-9 h-9 border-2 rounded-md border-foreground bg-primary/10 flex items-center justify-center shrink-0">
                  <section.icon className="w-4 h-4 text-primary" />
                </span>
                <span className="flex flex-col">
                  <span>{section.title}</span>
                  <span className="text-[11px] font-bold normal-case tracking-normal text-muted-foreground">{section.desc}</span>
                </span>
              </span>
            </AccordionTrigger>
            <AccordionContent>
              <section.Component />
            </AccordionContent>
          </AccordionItem>
        ))}
      </Accordion>
    </div>
  );
}
