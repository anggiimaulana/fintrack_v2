/**
 * Copy for the trust pages `/about` and `/contact` (spec doc 05 §1–§2).
 *
 * HONESTY MARKER: no company history, team size, or funding is invented. The
 * technical description stays at a high level and only states what the product
 * actually does. Contact addresses come from `site.ts` so no page can show a
 * different email.
 */

import { site } from "./site";

export interface SupportPrinciple {
  title: string;
  text: string;
}

export interface AboutContent {
  h1: string;
  intro: string;
  mission: { title: string; text: string };
  tech: { title: string; paragraphs: string[] };
  principles: { title: string; items: SupportPrinciple[] };
  audience: { title: string; text: string; link: { label: string; href: string } };
  contact: { title: string; text: string; link: { label: string; href: string } };
  cta: { label: string; href: string };
}

export const ABOUT: AboutContent = {
  h1: "Tentang FinTrack",
  intro:
    "FinTrack adalah aplikasi pencatat keuangan berbasis AI berbahasa Indonesia. Tujuannya sederhana: membuat pencatatan keuangan terasa seringan mengobrol, tanpa mengorbankan privasi dan kejelasan.",
  mission: {
    title: "Misi kami",
    text: "Membuat pencatatan keuangan semudah mengobrol — tanpa mengorbankan privasi dan kejelasan.",
  },
  tech: {
    title: "Bagaimana FinTrack bekerja",
    paragraphs: [
      "FinTrack adalah aplikasi web yang berjalan di browser. Data keuangan Anda disimpan pada basis data PostgreSQL, dan setiap permintaan dibatasi pada akun Anda sendiri.",
      "Fitur AI — mencatat lewat chat, mengubah suara menjadi teks, dan membaca struk — diproses melalui gateway AI terpusat milik FinTrack. Karena gateway ini yang mengelola koneksi ke penyedia model, Anda tidak perlu menyiapkan atau menyimpan API key sendiri.",
      "Kami menjelaskan arsitektur pada tingkat ini saja agar tetap akurat: tidak ada klaim berlebihan tentang teknologi, dan tidak ada detail yang belum benar-benar berlaku.",
    ],
  },
  principles: {
    title: "Prinsip produk",
    items: [
      {
        title: "Akurasi",
        text: "Setiap hasil AI ditampilkan lebih dulu untuk Anda periksa, sehingga catatan akhir tetap sesuai kenyataan.",
      },
      {
        title: "Privasi",
        text: "Data tiap akun terisolasi dan hanya dapat diakses oleh pemiliknya, dengan kontrol berbasis peran.",
      },
      {
        title: "Transparansi kuota",
        text: "Pemakaian dan batas fitur AI ditampilkan jelas memakai label resmi, bukan angka yang tersembunyi.",
      },
      {
        title: "Kendali data",
        text: "Anda dapat mengekspor seluruh data Anda dan menghapus akun beserta datanya dari pengaturan akun.",
      },
    ],
  },
  audience: {
    title: "Untuk siapa FinTrack dibuat",
    text: "FinTrack ditujukan untuk siapa saja yang ingin mencatat keuangan pribadi tanpa ribet: pekerja kantoran, freelancer dengan pendapatan tidak tetap, ibu rumah tangga yang banyak berbelanja, sampai pelaku UMKM yang perlu memantau arus kas usaha kecil.",
    link: { label: "Lihat contoh penggunaannya", href: "/#who-its-for" },
  },
  contact: {
    title: "Ada pertanyaan?",
    text: `Kirim pesan lewat email atau WhatsApp. Kami membalas pada hari kerja (${site.supportHours}).`,
    link: { label: "Hubungi kami", href: "/contact" },
  },
  cta: { label: "Mulai Gratis", href: "/register" },
};

export interface ContactChannel {
  title: string;
  text: string;
  /** Accessible name for the action link (spec doc 05 §2.4). */
  ariaLabel: string;
  /** Visible button label. */
  ctaLabel: string;
}

export interface ContactContent {
  h1: string;
  intro: string;
  channels: { email: ContactChannel; whatsapp: ContactChannel };
  hours: { title: string; text: string };
  before: {
    title: string;
    text: string;
    links: Array<{ label: string; href: string }>;
  };
  account: { title: string; text: string };
  cta: { label: string; href: string };
}

export const CONTACT: ContactContent = {
  h1: "Hubungi Kami",
  intro:
    "Hubungi kami bila ada pertanyaan soal akun, langganan, pembayaran, atau cara memakai fitur FinTrack yang belum terjawab di halaman bantuan.",
  channels: {
    email: {
      title: "Email",
      text: "Cocok untuk pertanyaan yang butuh penjelasan panjang atau lampiran.",
      ariaLabel: "Kirim email ke tim FinTrack",
      ctaLabel: "Kirim Email",
    },
    whatsapp: {
      title: "WhatsApp",
      text: "Cocok untuk pertanyaan singkat dan cepat.",
      ariaLabel: "Chat WhatsApp dengan tim FinTrack",
      ctaLabel: "Chat WhatsApp",
    },
  },
  hours: {
    title: "Jam operasional",
    text: `${site.supportHours}. ${site.responseTime}`,
  },
  before: {
    title: "Sebelum menghubungi",
    text: "Beberapa pertanyaan sudah dijawab di halaman lain. Semoga ini lebih cepat.",
    links: [
      { label: "Pertanyaan umum (FAQ)", href: "/faq" },
      { label: "Paket dan harga", href: "/pricing" },
      { label: "Panduan fitur", href: "/features" },
    ],
  },
  account: {
    title: "Untuk urusan akun",
    text: "Bila pertanyaan Anda menyangkut langganan, pembayaran, atau penghapusan data, sebutkan email akun Anda saat menghubungi agar kami dapat menindaklanjuti lebih cepat.",
  },
  cta: { label: "Mulai Gratis", href: "/register" },
};
