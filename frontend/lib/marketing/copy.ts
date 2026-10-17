/**
 * Canonical copy for the marketing surface: landing sections, navigation,
 * footer, microcopy, and per-page SEO titles/descriptions.
 *
 * Tone: bold visuals, calm words. No hyperbole, no unverifiable social proof,
 * no promises of features that do not exist (spec doc 13). Components read
 * strings from here so no page drifts.
 */

export interface Cta {
  label: string;
  href: string;
}

export interface NavItem {
  label: string;
  href: string;
}

export interface FooterColumn {
  title: string;
  links: NavItem[];
}

// ─── Navigation ────────────────────────────────────────────────

/** Desktop/mobile primary nav (Indonesian labels, English targets). */
export const NAV: NavItem[] = [
  { label: "Fitur", href: "/features" },
  { label: "Cara Kerja", href: "/#how-it-works" },
  { label: "Harga", href: "/pricing" },
  { label: "Jelajahi", href: "/explore" },
];

export const PRIMARY_CTA: Cta = { label: "Mulai Gratis", href: "/register" };
export const SIGN_IN_CTA: Cta = { label: "Masuk", href: "/login" };

// ─── Footer ────────────────────────────────────────────────────

export const FOOTER_COLUMNS: FooterColumn[] = [
  {
    title: "Produk",
    links: [
      { label: "Fitur", href: "/features" },
      { label: "Harga", href: "/pricing" },
      { label: "Cara Kerja", href: "/#how-it-works" },
      { label: "FAQ", href: "/faq" },
    ],
  },
  {
    title: "Perusahaan",
    links: [
      { label: "Tentang", href: "/about" },
      { label: "Kontak", href: "/contact" },
    ],
  },
  {
    title: "Sumber Daya",
    links: [
      { label: "Panduan Fitur", href: "/features" },
      { label: "Pertanyaan Umum", href: "/faq" },
    ],
  },
  {
    title: "Legal",
    links: [
      { label: "Kebijakan Privasi", href: "/privacy" },
      { label: "Syarat & Ketentuan", href: "/terms" },
    ],
  },
];

// ─── Microcopy ─────────────────────────────────────────────────

export const MICROCOPY = {
  openMenu: "Buka menu",
  closeMenu: "Tutup menu",
  demoLabel: "Contoh tampilan",
  catalogError: "Harga sedang tidak dapat dimuat. Coba lagi sebentar lagi.",
  retry: "Coba lagi",
  emptyPlans: "Detail paket belum tersedia. Lihat halaman harga.",
  viewPricing: "Lihat Harga",
  comingSoon: "Segera hadir",
} as const;

// ─── Landing ───────────────────────────────────────────────────

export const LANDING = {
  hero: {
    eyebrow: "ASISTEN KEUANGAN BERBAHASA INDONESIA",
    h1: "Catat keuangan otomatis. Kelola dengan tenang.",
    h1HighlightWord: "tenang",
    subhead:
      "Tulis, ucapkan, atau foto transaksinya. FinTrack mencatat dengan rapi dan membantu Anda memahami kondisi keuangan — tanpa spreadsheet.",
    primaryCta: PRIMARY_CTA,
    secondaryCta: { label: "Lihat Cara Kerja", href: "#how-it-works" },
    trustChips: ["Gratis untuk memulai", "Tanpa kartu kredit", "Data dapat diekspor"],
  },

  receiptTape: {
    caption: "Ilustrasi tampilan transaksi yang tercatat.",
    rows: [
      { label: "KOPI SUSU", amount: "Rp 25.000", category: "MAKANAN" },
      { label: "BELANJA BULANAN", amount: "Rp 312.500", category: "RUMAH TANGGA" },
      { label: "TOL CIKAMPEK", amount: "Rp 27.500", category: "TRANSPORTASI" },
      { label: "GAJI", amount: "+Rp 8.500.000", category: "PEMASUKAN" },
    ],
  },

  ticker: {
    words: ["CATAT", "PAHAMI", "KELOLA", "TENANG"],
    ariaLabel: "Catat, pahami, kelola, tenang",
  },

  trustBar: [
    {
      title: "Isolasi data",
      text: "Data tiap akun terpisah dan hanya Anda yang mengaksesnya.",
    },
    {
      title: "Kendali data",
      text: "Ekspor seluruh data Anda kapan saja.",
    },
    {
      title: "Hapus kapan saja",
      text: "Hapus akun beserta datanya dari pengaturan.",
    },
    {
      title: "Transparan",
      text: "Kuota dan pemakaian AI terlihat jelas di dasbor.",
    },
  ],

  stats: [
    { prefix: "", value: 3, suffix: "", label: "cara mencatat: chat, suara, foto" },
    { prefix: "", value: 5, suffix: "", label: "indikator status kesehatan keuangan" },
    { prefix: "Rp ", value: 0, suffix: "", label: "biaya untuk memulai" },
  ],

  problemSolution: {
    h2: "Mencatat keuangan tidak seharusnya menyita waktu",
    problemTitle: "Tanpa FinTrack",
    solutionTitle: "Dengan FinTrack",
    problems: [
      "Membuka spreadsheet dan mengisi kolom satu per satu.",
      "Lupa mencatat karena repot.",
      "Data menumpuk tanpa pernah ditinjau.",
    ],
    solutions: [
      "Menulis “makan siang 40 ribu” — langsung tercatat.",
      "Mencatat lewat chat, suara, atau foto struk.",
      "Mengajukan pertanyaan; mendapat skor dan status keuangan.",
    ],
  },

  featureSection: {
    eyebrow: "FITUR",
    h2: "Satu aplikasi untuk mencatat, memahami, dan mengelola",
    subhead: "Enam fitur utama yang saling melengkapi, dari mencatat sampai meninjau.",
  },

  howItWorks: {
    h2: "Mulai dalam tiga langkah",
    steps: [
      { title: "Daftar gratis", text: "Buat akun dalam hitungan detik. Tidak perlu kartu kredit." },
      { title: "Catat transaksi", text: "Pilih cara paling nyaman: chat, suara, atau foto struk." },
      { title: "Tinjau dan analisis", text: "Lihat ringkasan, lalu ajukan pertanyaan tentang keuangan Anda." },
    ],
  },

  spk: {
    eyebrow: "ANALISIS",
    h2: "Ketahui kondisi keuangan Anda",
    description:
      "Penilaian meninjau pola pengeluaran, tabungan, dan kebiasaan Anda, lalu memberi skor beserta status yang jelas.",
    statuses: ["Sangat Baik", "Baik", "Cukup", "Boros", "Kritis"],
    example: {
      score: 85,
      status: "Sangat Baik",
      note: "Kondisi keuangan Anda stabil. Rasio tabungan Anda 30% dari pendapatan.",
    },
  },

  pricingTeaser: {
    eyebrow: "HARGA",
    h2: "Harga sederhana, tanpa kejutan",
    subhead: "Mulai gratis. Naikkan paket saat butuh kuota AI lebih besar.",
    cta: { label: "Lihat Detail Paket", href: "/pricing" },
    emptyFallback: { label: "Lihat Harga", href: "/pricing" },
    popularBadge: "Paling Populer",
    hidesWhenEmpty: true,
  },

  personas: {
    eyebrow: "COCOK UNTUK SIAPA",
    h2: "Cocok untuk berbagai kebutuhan",
    items: [
      { title: "Freelancer", text: "Pendapatan tidak tetap? Lihat arus kas per bulan dan pastikan sisanya cukup." },
      { title: "Pekerja kantoran", text: "Catat gaji dan pengeluaran bulanan tanpa membuka spreadsheet." },
      { title: "Ibu rumah tangga", text: "Foto struk belanja bulanan; catatannya dirapikan otomatis." },
      { title: "Pelaku UMKM", text: "Pantau pemasukan dan pengeluaran usaha kecil dalam satu tempat." },
    ],
  },

  faq: {
    h2: "Pertanyaan umum",
    viewAll: { label: "Lihat semua FAQ", href: "/faq" },
  },

  closingCta: {
    h2: "Mulai rapikan keuangan Anda hari ini",
    subhead: "Gratis untuk memulai. Tanpa kartu kredit. Data Anda dapat diekspor atau dihapus kapan saja.",
    cta: { label: "Daftar Gratis", href: "/register" },
  },
} as const;

// ─── Feature hub & sub-pages ───────────────────────────────────

/** `/features` hub copy (spec doc 13 §7.1, doc 04 §B.7). */
export const FEATURES_HUB = {
  h1: "Fitur FinTrack",
  subhead: "Semua yang Anda butuhkan untuk mencatat, memahami, dan mengelola keuangan pribadi.",
  gridTitle: "Enam fitur utama",
  cardCta: "Pelajari lebih lanjut",
  connected: {
    title: "Bagaimana semuanya terhubung",
    paragraphs: [
      "Setiap fitur bermuara pada catatan yang sama. Transaksi yang Anda masukkan lewat chat, suara, atau foto struk langsung menjadi entri terstruktur.",
      "Dari catatan itu, dasbor menampilkan arus kas, analisis kesehatan finansial memberi skor dan status, dan laporan menutup siklusnya dengan berkas yang dapat Anda unduh.",
    ],
  },
  cta: { label: "Daftar Gratis", href: "/register" },
  pricingCta: { label: "Lihat Harga", href: "/pricing" },
} as const;

/** Section headings/labels shared by every `/features/{slug}` page. */
export const FEATURE_PAGE = {
  problemTitle: "Masalah yang diselesaikan",
  howTitle: "Cara kerjanya",
  detailTitle: "Bagaimana FinTrack melakukannya",
  benefitsTitle: "Manfaat utama",
  exampleTitle: "Contoh nyata",
  useCasesTitle: "Kapan paling berguna",
  notesTitle: "Yang perlu diketahui",
  availabilityLabel: "Tersedia di",
  availabilityNote:
    "Ketersediaan paket mengikuti katalog langganan yang berlaku. Keterangan kuota memakai label resmi di halaman harga.",
  relatedTitle: "Fitur terkait",
  allFeatures: { label: "Lihat semua fitur", href: "/features" },
  primaryCta: PRIMARY_CTA,
  pricingCta: { label: "Lihat Harga", href: "/pricing" },
  closing: {
    h2: "Mulai rapikan keuangan Anda hari ini",
    subhead: "Gratis untuk memulai. Tanpa kartu kredit. Data Anda dapat diekspor atau dihapus kapan saja.",
  },
} as const;

// ─── FAQ page ──────────────────────────────────────────────────

/** `/faq` copy (spec doc 05 §3). */
export const FAQ_PAGE = {
  h1: "Pertanyaan Umum",
  subhead:
    "Jawaban singkat dan langsung untuk pertanyaan yang paling sering muncul soal harga, langganan, AI, privasi data, dan pembayaran.",
  tocTitle: "Kategori",
  stillTitle: "Belum menemukan jawabannya?",
  stillText: "Kirim pertanyaan Anda dan kami akan membantu pada hari kerja.",
  contactCta: { label: "Hubungi Kami", href: "/contact" },
  closing: {
    h2: "Mulai rapikan keuangan Anda hari ini",
    subhead: "Gratis untuk memulai. Tanpa kartu kredit.",
  },
} as const;

// ─── Explore Hub ────────────────────────────────────────────────

export interface ExploreCard {
  title: string;
  tagline: string;
  href: string;
  badge: string;
  cta: string;
  iconName: "HelpCircle" | "Info" | "Headphones" | "ShieldCheck" | "FileText";
}

export const EXPLORE_HUB = {
  eyebrow: "JELAJAHI",
  h1: "Pusat Informasi & Eksplorasi",
  subhead:
    "Semua hal tentang FinTrack dalam satu tempat: panduan FAQ, misi pengembang, saluran bantuan resmi, hingga komitmen privasi dan ketentuan layanan.",
  gridTitle: "Daftar Halaman Eksplorasi",
  cta: {
    h2: "Siap Mengelola Keuangan dengan Tenang?",
    subhead: "Daftar dalam hitungan detik tanpa kartu kredit. Mulai catat transaksi secara otomatis dengan kecerdasan buatan.",
    primary: { label: "Mulai Gratis", href: "/register" },
    secondary: { label: "Lihat Harga", href: "/pricing" },
  },
  cards: [
    {
      title: "Pertanyaan Umum (FAQ)",
      tagline: "Jawaban komprehensif seputar fitur AI, batasan kuota gratis, keamanan isolasi data, dan integrasi pembayaran.",
      href: "/faq",
      badge: "Bantuan",
      cta: "Buka FAQ",
      iconName: "HelpCircle",
    },
    {
      title: "Tentang FinTrack",
      tagline: "Misi kami, arsitektur teknologi di balik sistem, dan prinsip privasi mutlak tanpa kompromi.",
      href: "/about",
      badge: "Profil",
      cta: "Pelajari Misi Kami",
      iconName: "Info",
    },
    {
      title: "Kontak & Dukungan",
      tagline: "Saluran komunikasi langsung dengan tim pengembang melalui WhatsApp resmi dan email untuk pertanyaan maupun kendala.",
      href: "/contact",
      badge: "Dukungan",
      cta: "Hubungi Kami",
      iconName: "Headphones",
    },
    {
      title: "Kebijakan Privasi",
      tagline: "Transparansi penuh atas pemrosesan data, hak ekspor riwayat keuangan, serta kendali penghapusan akun kapan saja.",
      href: "/privacy",
      badge: "Legalitas",
      cta: "Baca Kebijakan",
      iconName: "ShieldCheck",
    },
    {
      title: "Syarat & Ketentuan",
      tagline: "Ketentuan lisensi penggunaan platform, aturan akun pengguna, dan batasan tanggung jawab hukum FinTrack.",
      href: "/terms",
      badge: "Ketentuan",
      cta: "Baca Ketentuan",
      iconName: "FileText",
    },
  ] as ExploreCard[],
} as const;

// ─── Per-page SEO ──────────────────────────────────────────────

export interface PageMeta {
  title: string;
  description: string;
}

/** Canonical titles/descriptions (spec doc 13 §11). */
export const PAGE_META: Record<string, PageMeta> = {
  "/": {
    title: "FinTrack — Catat Keuangan Otomatis dengan AI",
    description:
      "Catat transaksi lewat chat, suara, atau foto struk. AI berbahasa Indonesia yang menyimpan rapi dan menganalisis kondisi keuangan Anda.",
  },
  "/pricing": {
    title: "Harga FinTrack — Paket Gratis, Pro, dan Business",
    description:
      "Bandingkan paket FinTrack: kuota AI, pindai struk, input suara, dan laporan. Mulai gratis, upgrade kapan saja.",
  },
  "/features": {
    title: "Fitur FinTrack — AI, OCR Struk, Suara, Laporan",
    description:
      "Enam fitur utama FinTrack: catat lewat chat, suara, dan foto struk; dasbor, analisis kesehatan finansial, serta laporan dan ekspor data.",
  },
  "/explore": {
    title: "Jelajahi FinTrack — Pusat Informasi, Bantuan & Legalitas",
    description:
      "Temukan panduan, jawaban pertanyaan umum, profil misi, saluran bantuan, serta transparansi privasi dan ketentuan FinTrack.",
  },
  "/about": {
    title: "Tentang FinTrack — Misi & Prinsip",
    description:
      "Misi FinTrack: pencatatan keuangan semudah mengobrol, dengan privasi dan transparansi kuota.",
  },
  "/contact": {
    title: "Kontak FinTrack — Bantuan & Pertanyaan",
    description:
      "Hubungi tim FinTrack lewat email atau WhatsApp. Lihat juga FAQ, harga, dan panduan fitur.",
  },
  "/faq": {
    title: "Pertanyaan Umum (FAQ) — FinTrack",
    description:
      "Jawaban atas pertanyaan umum soal FinTrack: harga, langganan, AI, privasi data, dan pembayaran.",
  },
  "/privacy": {
    title: "Kebijakan Privasi — FinTrack",
    description:
      "Bagaimana FinTrack mengumpulkan, memakai, dan melindungi data Anda; serta hak akses, ekspor, dan penghapusan.",
  },
  "/terms": {
    title: "Syarat & Ketentuan — FinTrack",
    description:
      "Ketentuan penggunaan layanan FinTrack: akun, langganan, pembayaran, kuota AI, dan tanggung jawab.",
  },
};
