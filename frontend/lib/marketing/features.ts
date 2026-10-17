import type { LucideIcon } from "lucide-react";
import { Camera, Download, LayoutDashboard, MessageSquare, Mic, Target } from "lucide-react";

/**
 * Central content model for the six feature pages.
 *
 * One source of truth for the feature hub, the cross-links, the pricing table,
 * the landing grid, and `app/sitemap.ts` (spec doc 04 §B.2). Slugs are English
 * (URL), all copy is Indonesian. Every page must describe a distinct problem,
 * flow, benefit set, and concrete example — never templated filler.
 */

export type PlanCode = "free" | "pro" | "business";

export interface FeatureExample {
  /** What the user types / says / sends. */
  prompt: string;
  /** What FinTrack returns. */
  result: string;
  /** Optional one-line caption clarifying this is an illustration. */
  caption?: string;
}

export interface FeaturePage {
  /** URL slug, English, kebab-case. */
  slug: string;
  icon: LucideIcon;
  /** H1 / card title. */
  title: string;
  /** Short label for tables and footers. */
  navLabel: string;
  /** One-sentence summary. */
  tagline: string;
  /** Target search phrases. */
  keywords: string[];
  /** Opening paragraph under the H1. */
  heroDescription: string;
  /** The pain this feature removes. */
  problem: string;
  /** Numbered steps to use the feature. */
  howItWorks: string[];
  /** Three to four payoff points. */
  benefits: string[];
  /** A concrete interaction example (not a testimonial). */
  example: FeatureExample;
  /** Two cross-links to sibling pages. */
  relatedSlugs: string[];
  /** Plans where the feature is enabled (drives the availability badge). */
  enabledInPlans: PlanCode[];
  /** Matching backend catalog feature key, when the feature is gated. */
  catalogFeature?: string;
}

export const FEATURES: FeaturePage[] = [
  {
    slug: "ai-chat",
    icon: MessageSquare,
    title: "Catat & Tanya Lewat Chat",
    navLabel: "Asisten AI (chat)",
    tagline: "Catat dan bertanya seperti mengobrol.",
    keywords: ["catat pengeluaran via chat", "asisten keuangan AI", "chat keuangan bahasa Indonesia"],
    heroDescription:
      "Tulis transaksi seperti Anda mengirim pesan ke teman. FinTrack membaca kalimatnya, lalu menyusun jumlah, kategori, dan tanggalnya menjadi catatan yang rapi.",
    problem:
      "Menulis catatan keuangan terasa seperti mengisi formulir. Membuka aplikasi, memilih kategori, mengeklik tanggal, mengetik nominal — lalu mengulanginya untuk setiap pengeluaran. Karena merepotkan, banyak orang menunda mencatat sampai lupa sama sekali. Padahal, catatan yang lengkap justru yang paling berguna.",
    howItWorks: [
      "Buka menu AI dari dasbor dan mulailah menulis seperti biasa.",
      "Sampaikan transaksinya dengan kalimat alami, misalnya “beli kopi 25 ribu tadi siang”.",
      "FinTrack mengenali jenis transaksi, nominal, kategori, dan waktu, lalu mengusulkan satu entri terstruktur.",
      "Periksa sekilas dan simpan. Anda juga bisa bertanya tentang keuangan dan mendapat jawaban dari data Anda sendiri.",
    ],
    benefits: [
      "Tidak perlu menyesuaikan diri dengan formulir — cukup menulis seperti biasa.",
      "Kategori dan tanggal terisi otomatis, sehingga pencatatan tetap konsisten.",
      "Bisa dipakai untuk mencatat sekaligus bertanya, dalam satu tempat.",
      "Tidak memerlukan API key; FinTrack memakai gateway AI terpusatnya sendiri.",
    ],
    example: {
      prompt: "beli kopi 25 ribu tadi siang",
      result: "Pengeluaran Rp 25.000 — Kopi. Tercatat.",
      caption: "Contoh tampilan",
    },
    relatedSlugs: ["voice-input", "receipt-scan"],
    enabledInPlans: ["free", "pro", "business"],
    catalogFeature: "ai_chat",
  },
  {
    slug: "voice-input",
    icon: Mic,
    title: "Catat dengan Suara",
    navLabel: "Input suara",
    tagline: "Catat dengan suara, tanpa menghentikan aktivitas.",
    keywords: ["catat transaksi suara", "input suara Indonesia", "catat pengeluaran tanpa mengetik"],
    heroDescription:
      "Sampaikan pengeluaran dengan suara saat tangan Anda sedang sibuk — mengemudi, memasak, atau menggendong anak. FinTrack mengubah ucapan menjadi transaksi.",
    problem:
      "Tangan sibuk saat pengeluaran terjadi. Saat membayar parkir, membeli air minum di jalan, atau menambah bahan masakan, tidak ada kesempatan untuk mengetik. Transaksi kecil inilah yang paling sering luput dicatat, lalu hilang dari perhitungan akhir bulan.",
    howItWorks: [
      "Tekan tombol mikrofon pada menu AI.",
      "Ucapkan transaksinya secara alami, misalnya “makan siang empat puluh ribu”.",
      "FinTrack menyalin ucapan menjadi teks, lalu menyusunnya menjadi entri berjumlah, berkategori, dan bertanggal.",
      "Tinjau hasilnya dan simpan bila sudah sesuai.",
    ],
    benefits: [
      "Mencatat tanpa mengalihkan pandangan atau melepas apa yang sedang dipegang.",
      "Cocok untuk transaksi kecil yang terjadi sepanjang hari.",
      "Bahasa Indonesia dipahami apa adanya, tanpa perintah khusus.",
      "Kuota suara ditampilkan jelas pada label “Permintaan suara / hari”.",
    ],
    example: {
      prompt: "makan siang empat puluh ribu",
      result: "Pengeluaran Rp 40.000 — Makan. Tercatat.",
      caption: "Contoh tampilan",
    },
    relatedSlugs: ["ai-chat", "receipt-scan"],
    enabledInPlans: ["pro", "business"],
    catalogFeature: "voice_input",
  },
  {
    slug: "receipt-scan",
    icon: Camera,
    title: "Pindai Struk Otomatis",
    navLabel: "Pindai struk (OCR)",
    tagline: "Foto struk, catatannya dirapikan otomatis.",
    keywords: ["scan struk otomatis", "OCR struk belanja", "catat belanja dari foto"],
    heroDescription:
      "Foto struk belanja Anda. FinTrack membaca item beserta totalnya, lalu menyimpannya sebagai transaksi — tanpa mengetik ulang satu per satu.",
    problem:
      "Mengetik ulang deretan belanjaan dari struk itu melelahkan dan rawan salah. Satu nomor tertukar, dan total Anda tidak lagi cocok dengan kenyataan. Akibatnya banyak orang memilih tidak mencatat belanja besar sama sekali.",
    howItWorks: [
      "Buka menu AI dan pilih opsi kamera.",
      "Ambil foto struk yang cukup jelas dan tidak terpotong.",
      "FinTrack membaca daftar item, kategori, dan total belanja.",
      "Periksa hasilnya, koreksi bila perlu, lalu simpan.",
    ],
    benefits: [
      "Menghemat waktu untuk belanja dengan banyak item.",
      "Angka yang dicatat mengikuti struk, sehingga rekap tetap akurat.",
      "Kategori terisi otomatis mengikuti jenis belanja.",
      "Cocok dipadukan dengan catatan harian lewat chat dan suara.",
    ],
    example: {
      prompt: "foto struk ini",
      result: "Belanja bulanan Rp 312.500 tercatat, 8 item.",
      caption: "Contoh tampilan",
    },
    relatedSlugs: ["voice-input", "ai-chat"],
    enabledInPlans: ["pro", "business"],
    catalogFeature: "ocr_receipt",
  },
  {
    slug: "dashboard",
    icon: LayoutDashboard,
    title: "Dasbor Keuangan Interaktif",
    navLabel: "Dasbor",
    tagline: "Lihat arus kas Anda dalam satu tampilan.",
    keywords: ["dashboard keuangan", "grafik pengeluaran", "laporan arus kas"],
    heroDescription:
      "Pantau pemasukan, pengeluaran, dan trennya dalam tampilan yang mudah dibaca. Satu layar untuk menjawab “bagaimana keuangan saya bulan ini?”.",
    problem:
      "Angka berserakan, sulit melihat gambaran besarnya. Ketika catatan hanya berupa daftar panjang, Anda tahu setiap transaksi tetapi tidak tahu arah keuangannya. Pola pengeluaran yang berulang pun tidak terlihat.",
    howItWorks: [
      "Semua transaksi yang Anda catat langsung tampil di dasbor.",
      "Lihat ringkasan pemasukan, pengeluaran, dan selisihnya pada periode berjalan.",
      "Telusuri grafik tren untuk melihat perubahan dari bulan ke bulan.",
      "Klik pada kategori untuk memahami ke mana uang Anda paling banyak pergi.",
    ],
    benefits: [
      "Gambaran menyeluruh tanpa perlu membuka spreadsheet.",
      "Perbandingan tren antar periode yang mudah dibaca.",
      "Ringkasan yang ikut terbarui saat Anda menambah transaksi.",
      "Menjadi dasar untuk analisis kesehatan finansial.",
    ],
    example: {
      prompt: "berapa pengeluaran saya bulan ini?",
      result: "Pengeluaran bulan ini Rp 4.200.000, naik 8% dari bulan lalu.",
      caption: "Contoh tampilan",
    },
    relatedSlugs: ["financial-health", "reports"],
    enabledInPlans: ["free", "pro", "business"],
  },
  {
    slug: "financial-health",
    icon: Target,
    title: "Analisis Kesehatan Finansial",
    navLabel: "Kesehatan finansial",
    tagline: "Ketahui skor dan status kesehatan finansial.",
    keywords: ["analisis keuangan pribadi", "skor keuangan", "kesehatan finansial"],
    heroDescription:
      "Penilaian meninjau pola pengeluaran, tabungan, dan kebiasaan Anda, lalu memberi skor beserta status yang jelas — dari Sangat Baik hingga Kritis.",
    problem:
      "Sulit menilai apakah kebiasaan keuangan Anda membaik. Tanpa ukuran, setiap bulan terasa sama. Anda mencatat rajin, tetapi tidak tahu apakah arahnya benar atau justru memburuk perlahan.",
    howItWorks: [
      "FinTrack meninjau transaksi, tabungan, dan pola yang terbentuk dari waktu ke waktu.",
      "Metrik dihitung menjadi skor tunggal yang mudah dipahami.",
      "Skor dipetakan ke salah satu status: Sangat Baik, Baik, Cukup, Boros, atau Kritis.",
      "Anda dapat menanyakan alasannya lewat chat untuk memahami bagian yang perlu diperbaiki.",
    ],
    benefits: [
      "Ukuran yang jelas, bukan sekadar perasaan.",
      "Status disertai label teks, tidak bergantung pada warna saja.",
      "Membantu memutuskan kapan menahan diri dan kapan menabung lebih banyak.",
      "Terhubung dengan dasbor, sehingga analisis selalu memakai data terbaru.",
    ],
    example: {
      prompt: "bagaimana kondisi keuangan saya?",
      result: "Skor 85 — Sangat Baik. Rasio tabungan 30% dari pendapatan.",
      caption: "Contoh tampilan",
    },
    relatedSlugs: ["dashboard", "reports"],
    enabledInPlans: ["free", "pro", "business"],
  },
  {
    slug: "reports",
    icon: Download,
    title: "Laporan & Ekspor Data",
    navLabel: "Laporan & ekspor",
    tagline: "Hasilkan laporan, unduh data Anda.",
    keywords: ["laporan keuangan otomatis", "ekspor data keuangan", "unduh laporan keuangan"],
    heroDescription:
      "Hasilkan laporan ringkas untuk periode tertentu, lalu unduh data Anda kapan saja. Data keuangan Anda tetap milik Anda.",
    problem:
      "Data terkunci di aplikasi, sulit dipakai di tempat lain. Saat perlu melampirkan rekap untuk urusan pajak, pengajuan, atau pembukuan usaha kecil, daftar transaksi hanya bisa dilihat di layar.",
    howItWorks: [
      "Pilih periode yang ingin Anda tinjau.",
      "FinTrack menyusun ringkasan pemasukan, pengeluaran, dan kategori terbesar.",
      "Tinjau laporannya langsung di layar.",
      "Ekspor atau unduh datanya untuk disimpan dan dipakai di luar FinTrack.",
    ],
    benefits: [
      "Rekap siap pakai tanpa menyusun sendiri di spreadsheet.",
      "Data dapat diunduh kapan saja — tidak terkunci di dalam aplikasi.",
      "Membantu keperluan pembukuan pribadi maupun usaha kecil.",
      "Melengkapi dasbor dan analisis kesehatan finansial.",
    ],
    example: {
      prompt: "ekspor transaksi bulan ini",
      result: "Berkas CSV berisi 128 transaksi siap diunduh.",
      caption: "Contoh tampilan",
    },
    relatedSlugs: ["dashboard", "financial-health"],
    enabledInPlans: ["pro", "business"],
    catalogFeature: "reports_export",
  },
];

/** English slugs, in display order. Used by the sitemap and static params. */
export const FEATURE_SLUGS: string[] = FEATURES.map((f) => f.slug);

/** Look up a feature page by slug, or `undefined` for an unknown slug. */
export function getFeature(slug: string): FeaturePage | undefined {
  return FEATURES.find((f) => f.slug === slug);
}

/** Canonical href for a feature page. */
export function featureHref(slug: string): string {
  return `/features/${slug}`;
}

/** Human label for a plan code, used by the availability badge. */
export const PLAN_LABEL: Record<PlanCode, string> = {
  free: "Free",
  pro: "Pro",
  business: "Business",
};
