/**
 * FAQ content: grouped, answer-first answers used by `/faq` and the landing
 * accordion. Answers start with the direct answer, then add detail — good for
 * readers and for AI citation (spec doc 05 §3.3).
 *
 * No `FAQPage` structured data is emitted (Google retired the FAQ rich result);
 * the content is rendered as plain text/`<details>` instead.
 */

export interface FaqLink {
  label: string;
  href: string;
}

export interface FaqItem {
  q: string;
  a: string;
  links?: FaqLink[];
}

export interface FaqCategory {
  /** Anchor slug for the `/faq` table of contents. */
  id: string;
  title: string;
  items: FaqItem[];
}

export const FAQ_CATEGORIES: FaqCategory[] = [
  {
    id: "umum",
    title: "Umum",
    items: [
      {
        q: "Apakah FinTrack gratis?",
        a: "Ya. Ada paket Free yang bisa dipakai tanpa batas waktu untuk mulai mencatat. Paket berbayar menambah kuota AI dan membuka fitur seperti pindai struk, input suara, dan ekspor laporan.",
        links: [
          { label: "lihat paket dan harga", href: "/pricing" },
          { label: "cara memindai struk", href: "/features/receipt-scan" },
        ],
      },
      {
        q: "Siapa yang cocok memakai FinTrack?",
        a: "FinTrack cocok untuk siapa saja yang ingin mencatat keuangan pribadi tanpa ribet: pekerja kantoran, freelancer dengan pendapatan tidak tetap, ibu rumah tangga yang banyak berbelanja, sampai pelaku UMKM yang perlu memantau arus kas usaha kecil.",
      },
      {
        q: "Apakah FinTrack berbahasa Indonesia?",
        a: "Ya. FinTrack dibuat untuk bahasa Indonesia, mulai dari antarmuka hingga pemahaman perintah dan struk belanja.",
      },
      {
        q: "Bagaimana cara mulai memakainya?",
        a: "Buat akun gratis, lalu catat transaksi pertama Anda lewat chat, suara, atau foto struk. Tidak perlu kartu kredit untuk memulai.",
        links: [{ label: "lihat cara kerja", href: "/#how-it-works" }],
      },
    ],
  },
  {
    id: "akun-langganan",
    title: "Akun & Langganan",
    items: [
      {
        q: "Bagaimana cara upgrade atau downgrade paket?",
        a: "Anda dapat naik atau turun paket dari halaman langganan di dalam aplikasi. Kenaikan paket berlaku sesuai alur pembayaran, sedangkan penurunan paket dijadwalkan berlaku pada akhir periode yang sudah Anda bayar.",
        links: [{ label: "lihat paket dan harga", href: "/pricing" }],
      },
      {
        q: "Bisakah saya berhenti berlangganan?",
        a: "Bisa. Anda dapat membatalkan langganan kapan saja, dan akses tetap berjalan sampai akhir periode yang sudah Anda bayar.",
      },
      {
        q: "Apa yang terjadi pada akhir periode?",
        a: "Pada akhir periode, langganan berhenti dan akun kembali ke paket Free. Data yang sudah Anda catat tidak dihapus; hanya fitur dan kuota yang mengikuti paket Free.",
      },
      {
        q: "Apakah data saya hilang bila turun paket?",
        a: "Tidak. Penurunan paket tidak menghapus data. Anda hanya kehilangan akses ke fitur atau kuota yang tidak termasuk dalam paket baru.",
      },
    ],
  },
  {
    id: "ai-privasi",
    title: "AI & Privasi",
    items: [
      {
        q: "Apakah saya perlu API key sendiri?",
        a: "Tidak. FinTrack memakai gateway AI terpusat, sehingga Anda tidak perlu menyiapkan API key sendiri untuk memakai fitur AI.",
        links: [{ label: "tentang asisten AI", href: "/features/ai-chat" }],
      },
      {
        q: "Bagaimana data saya diperlakukan?",
        a: "Data setiap akun terpisah dan hanya Anda yang dapat mengaksesnya. Transaksi yang Anda masukkan dipakai untuk menghasilkan catatan, ringkasan, dan analisis untuk akun Anda sendiri.",
        links: [{ label: "kebijakan privasi", href: "/privacy" }],
      },
      {
        q: "Bisakah saya mengekspor atau menghapus data?",
        a: "Bisa. Anda dapat mengekspor seluruh data keuangan Anda, dan dapat menghapus akun beserta datanya dari pengaturan akun.",
        links: [{ label: "ekspor dan laporan", href: "/features/reports" }],
      },
      {
        q: "Siapa yang bisa melihat data saya?",
        a: "Hanya Anda. Akses dibatasi per akun dengan kontrol berbasis peran, dan data antar akun diisolasi.",
      },
    ],
  },
  {
    id: "pembayaran",
    title: "Pembayaran",
    items: [
      {
        q: "Metode pembayaran apa saja yang tersedia?",
        a: "Pembayaran dapat dilakukan melalui gateway pembayaran otomatis maupun transfer manual yang diverifikasi oleh admin.",
        links: [{ label: "lihat paket dan harga", href: "/pricing" }],
      },
      {
        q: "Apakah ada pajak?",
        a: "Bila pajak berlaku, jumlahnya ditampilkan sebagai baris terpisah dan biaya akhir selalu diperlihatkan sebelum Anda membayar.",
      },
      {
        q: "Bagaimana cara transfer manual diverifikasi?",
        a: "Setelah transfer, unggah bukti pembayaran di halaman tagihan. Admin akan memverifikasinya, lalu langganan aktif setelah pembayaran disetujui.",
      },
      {
        q: "Apakah ada biaya tersembunyi?",
        a: "Tidak. Harga paket dan, bila ada, pajak ditampilkan transparan sebelum pembayaran. Tidak ada biaya yang muncul tiba-tiba setelahnya.",
      },
    ],
  },
];

/** Six concise items for the landing accordion (`#faq`). */
export const LANDING_FAQ: FaqItem[] = [
  {
    q: "Apakah FinTrack gratis?",
    a: "Ya. Ada paket Free yang bisa dipakai tanpa batas waktu untuk mulai mencatat. Paket berbayar menambah kuota AI dan fitur.",
    links: [{ label: "lihat paket dan harga", href: "/pricing" }],
  },
  {
    q: "Bagaimana AI memahami perintah saya?",
    a: "AI memproses teks, suara, dan gambar, lalu menyusunnya menjadi transaksi terstruktur.",
    links: [{ label: "tentang asisten AI", href: "/features/ai-chat" }],
  },
  {
    q: "Apakah data saya aman?",
    a: "Data tiap akun terpisah. Anda dapat mengekspor atau menghapusnya kapan saja.",
    links: [{ label: "kebijakan privasi", href: "/privacy" }],
  },
  {
    q: "Apakah perlu API key sendiri?",
    a: "Tidak. FinTrack memakai gateway AI terpusat — Anda tidak perlu API key sendiri.",
    links: [{ label: "cara mencatat lewat chat", href: "/features/ai-chat" }],
  },
  {
    q: "Apakah bisa ekspor data?",
    a: "Ya. Anda dapat menghasilkan laporan dan mengekspor data keuangan Anda.",
    links: [{ label: "ekspor dan laporan", href: "/features/reports" }],
  },
  {
    q: "Bagaimana cara berlangganan atau membatalkan?",
    a: "Anda dapat naik atau turun paket kapan saja; downgrade berlaku di akhir periode yang sudah dibayar.",
    links: [{ label: "lihat paket dan harga", href: "/pricing" }],
  },
];

/**
 * Eight concise pricing objections for `/pricing` (spec doc 04 §A.7). Answers
 * are answer-first and follow the real backend behaviour: scheduled downgrade,
 * manual transfer verification, and a transparent tax line.
 */
export const PRICING_FAQ: FaqItem[] = [
  {
    q: "Apakah ada paket gratis?",
    a: "Ya. Paket Free bisa dipakai tanpa batas waktu untuk mulai mencatat, tanpa kartu kredit.",
    links: [{ label: "daftar gratis", href: "/register" }],
  },
  {
    q: "Apa bedanya Pro dan Business?",
    a: "Business memberi kuota AI yang lebih besar daripada Pro dan menambahkan akses API. Keduanya sudah termasuk pindai struk, input suara, pencarian cerdas, anggaran lanjutan, dan ekspor laporan.",
    links: [{ label: "lihat perbandingan fitur", href: "/pricing#perbandingan" }],
  },
  {
    q: "Bagaimana penagihannya?",
    a: "Langganan ditagih per siklus sesuai masa berlaku paket. Bila pajak berlaku, jumlahnya ditampilkan sebagai baris terpisah dan total terlihat sebelum Anda membayar.",
  },
  {
    q: "Metode pembayaran apa saja yang tersedia?",
    a: "Pembayaran otomatis melalui gateway Midtrans, atau transfer manual yang diverifikasi oleh admin. Untuk transfer manual, langganan aktif setelah pembayaran disetujui.",
    links: [{ label: "hubungi tim kami", href: "/contact" }],
  },
  {
    q: "Bisakah saya beralih paket?",
    a: "Bisa. Kenaikan paket mengikuti alur pembayaran, sedangkan penurunan paket dijadwalkan berlaku pada akhir periode yang sudah Anda bayar.",
  },
  {
    q: "Bisakah saya berhenti berlangganan?",
    a: "Bisa. Anda dapat membatalkan langganan kapan saja, dan akses tetap berjalan sampai akhir periode yang sudah dibayar; setelah itu akun kembali ke paket Free.",
  },
  {
    q: "Apa yang terjadi bila kuota habis?",
    a: "Bila kuota paket Anda habis, fitur terkait dapat dibatasi sampai kuota siklus berikutnya tersedia. Anda dapat menaikkan paket untuk kuota yang lebih besar.",
    links: [{ label: "lihat paket dan harga", href: "/pricing" }],
  },
  {
    q: "Apakah ada biaya tersembunyi?",
    a: "Tidak. Harga paket dan, bila ada, pajak ditampilkan transparan sebelum pembayaran. Tidak ada biaya yang muncul setelahnya.",
  },
];
