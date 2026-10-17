/**
 * Legal content (Privacy Policy & Terms) as structured data, rendered by the
 * shared `LegalPage` component (spec doc 05 §6).
 *
 * HONESTY MARKER: `draft: true` means this text has NOT been reviewed by a
 * qualified lawyer. It must be reviewed and the flag flipped before it is
 * treated as final/authoritative. Do not claim compliance or certification that
 * has not actually been obtained.
 */

export interface LegalSection {
  /** Anchor id for the table of contents. */
  id: string;
  heading: string;
  paragraphs?: string[];
  list?: string[];
}

export interface LegalDocument {
  id: "privacy" | "terms";
  title: string;
  /** Plain-language summary shown at the top. */
  summary: string;
  /** ISO date this version takes effect. */
  effectiveDate: string;
  /** True until reviewed by legal counsel. */
  draft: boolean;
  sections: LegalSection[];
}

export const PRIVACY: LegalDocument = {
  id: "privacy",
  title: "Kebijakan Privasi",
  summary:
    "Kebijakan ini menjelaskan data apa yang FinTrack kumpulkan, untuk apa data itu dipakai, dan kendali apa yang Anda miliki atasnya.",
  effectiveDate: "2026-10-17",
  draft: true,
  sections: [
    {
      id: "ringkasan",
      heading: "Ringkasan",
      paragraphs: [
        "Kami mengumpulkan data seperlunya untuk menjalankan layanan: informasi akun, data keuangan yang Anda masukkan sendiri, data teknis untuk keamanan, dan data langganan bila Anda berpaket berbayar.",
        "Data setiap akun terpisah. Anda dapat mengekspor data Anda dan menghapus akun beserta datanya dari pengaturan akun.",
      ],
    },
    {
      id: "data-yang-dikumpulkan",
      heading: "Data yang Kami Kumpulkan",
      list: [
        "Data akun: nama, alamat email, nomor telepon (opsional), dan avatar (opsional).",
        "Data keuangan yang Anda masukkan: transaksi, kategori, anggaran, dan catatan lain yang Anda buat.",
        "Data teknis: log aktivitas, informasi perangkat, dan alamat IP untuk keamanan dan pencegahan penyalahgunaan.",
        "Data langganan dan pembayaran: paket, riwayat tagihan, dan status pembayaran. Data pembayaran diproses oleh penyedia pembayaran; kami tidak menyimpan nomor kartu penuh.",
      ],
    },
    {
      id: "tujuan-pemrosesan",
      heading: "Tujuan Pemrosesan",
      list: [
        "Menyediakan dan memelihara layanan pencatatan keuangan.",
        "Memproses permintaan AI seperti mencatat transaksi, meringkas, dan menganalisis.",
        "Memproses penagihan dan langganan.",
        "Menjaga keamanan layanan dan mencegah penyalahgunaan.",
        "Memberikan dukungan ketika Anda menghubungi kami.",
      ],
    },
    {
      id: "pemrosesan-ai",
      heading: "Pemrosesan AI",
      paragraphs: [
        "Permintaan AI diproses melalui gateway AI terpusat milik FinTrack. Anda tidak perlu menyediakan API key pribadi, dan API key pribadi tidak kami simpan.",
        "Permintaan yang Anda kirim dipakai untuk menghasilkan jawaban pada akun Anda, bukan untuk melatih model milik pihak lain.",
      ],
    },
    {
      id: "berbagi-data",
      heading: "Berbagi Data",
      paragraphs: [
        "Kami hanya membagikan data kepada penyedia yang diperlukan untuk menjalankan layanan — misalnya pemroses AI dan gateway pembayaran — dan sebatas yang dibutuhkan. Kami tidak menjual data Anda.",
        "Kami dapat mengungkapkan data bila diwajibkan oleh hukum yang berlaku.",
      ],
    },
    {
      id: "penyimpanan-retensi",
      heading: "Penyimpanan & Retensi",
      paragraphs: [
        "Data disimpan selama akun Anda aktif. Bila Anda menghapus akun, data pribadi dihapus atau dianonimkan sesuai kebutuhan operasional dan kewajiban hukum yang berlaku.",
      ],
    },
    {
      id: "hak-anda",
      heading: "Hak Anda",
      list: [
        "Mengakses data yang kami simpan tentang Anda.",
        "Mengekspor data keuangan Anda.",
        "Memperbaiki data yang tidak akurat.",
        "Menghapus akun beserta datanya.",
        "Mengajukan keberatan atas pemrosesan tertentu.",
      ],
      paragraphs: ["Untuk menggunakan hak ini, buka pengaturan akun atau hubungi kami melalui halaman kontak."],
    },
    {
      id: "keamanan",
      heading: "Keamanan",
      paragraphs: [
        "Kami menerapkan kontrol akses berbasis peran, isolasi data antar akun, dan perlindungan terhadap kredensial layanan. Meski demikian, tidak ada sistem yang sepenuhnya bebas risiko.",
      ],
    },
    {
      id: "anak-anak",
      heading: "Anak-anak",
      paragraphs: [
        "FinTrack tidak ditujukan untuk anak di bawah usia yang diizinkan oleh hukum yang berlaku. Kami tidak dengan sengaja mengumpulkan data anak-anak.",
      ],
    },
    {
      id: "perubahan-kebijakan",
      heading: "Perubahan Kebijakan",
      paragraphs: [
        "Kami dapat memperbarui kebijakan ini dari waktu ke waktu. Perubahan penting akan diberitahukan melalui layanan, dan tanggal berlaku akan diperbarui pada halaman ini.",
      ],
    },
    {
      id: "kontak-privasi",
      heading: "Kontak Privasi",
      paragraphs: ["Untuk pertanyaan tentang privasi, hubungi kami melalui halaman kontak."],
    },
  ],
};

export const TERMS: LegalDocument = {
  id: "terms",
  title: "Syarat & Ketentuan",
  summary:
    "Syarat ini mengatur penggunaan layanan FinTrack, termasuk akun, langganan, pembayaran, kuota AI, dan tanggung jawab Anda.",
  effectiveDate: "2026-10-17",
  draft: true,
  sections: [
    {
      id: "penerimaan",
      heading: "Penerimaan Syarat",
      paragraphs: ["Dengan membuat akun atau memakai FinTrack, Anda menyetujui syarat ini."],
    },
    {
      id: "akun",
      heading: "Akun",
      list: [
        "Anda harus memenuhi usia minimum yang diizinkan hukum yang berlaku untuk memakai layanan.",
        "Anda bertanggung jawab menjaga kerahasiaan kata sandi dan aktivitas pada akun Anda.",
        "Anda wajib memberikan informasi akun yang benar.",
      ],
    },
    {
      id: "penggunaan-yang-diizinkan",
      heading: "Penggunaan yang Diizinkan",
      list: [
        "Dilarang menyalahgunakan layanan, termasuk upaya mengakses data akun lain.",
        "Dilarang melakukan otomatisasi berlebihan atau upaya melewati kuota.",
        "Dilarang memakai layanan untuk tindakan yang melanggar hukum.",
      ],
    },
    {
      id: "langganan-pembayaran",
      heading: "Langganan & Pembayaran",
      paragraphs: [
        "Langganan berbayar ditagih per periode. Bila Anda membatalkan langganan, akses tetap berjalan sampai akhir periode yang sudah dibayar, lalu akun kembali ke paket Free.",
        "Bila pajak berlaku, jumlahnya ditampilkan sebagai baris terpisah dan biaya akhir diperlihatkan sebelum pembayaran.",
      ],
    },
    {
      id: "kuota-ai",
      heading: "Kuota AI",
      paragraphs: [
        "Setiap paket memiliki kuota AI, misalnya token AI per siklus, permintaan AI per siklus, transaksi, pindai struk per siklus, dan permintaan suara per hari. Bila kuota habis, fitur terkait dapat dibatasi sampai kuota berikutnya tersedia.",
      ],
    },
    {
      id: "hak-kekayaan-intelektual",
      heading: "Hak Kekayaan Intelektual",
      paragraphs: [
        "Perangkat lunak FinTrack dan mereknya milik kami. Data yang Anda masukkan tetap milik Anda.",
      ],
    },
    {
      id: "batasan-tanggung-jawab",
      heading: "Batasan Tanggung Jawab",
      paragraphs: [
        "FinTrack membantu Anda mencatat dan memahami keuangan, tetapi bukan nasihat keuangan profesional. Layanan disediakan sebagaimana adanya, tanpa jaminan yang melampaui ketentuan hukum yang berlaku.",
      ],
    },
    {
      id: "penghentian",
      heading: "Penghentian",
      paragraphs: [
        "Akun dapat ditangguhkan atau ditutup bila melanggar syarat ini atau membahayakan layanan dan pengguna lain.",
      ],
    },
    {
      id: "perubahan-syarat",
      heading: "Perubahan Syarat",
      paragraphs: [
        "Kami dapat memperbarui syarat ini. Perubahan penting akan diberitahukan melalui layanan, dan tanggal berlaku akan diperbarui pada halaman ini.",
      ],
    },
    {
      id: "hukum-kontak",
      heading: "Hukum yang Berlaku & Kontak",
      paragraphs: [
        "Syarat ini diatur oleh hukum yang berlaku di Indonesia. Untuk pertanyaan tentang syarat ini, hubungi kami melalui halaman kontak.",
      ],
    },
  ],
};

/** Required section ids per document, asserted by tests and the renderer. */
export const REQUIRED_SECTIONS: Record<LegalDocument["id"], string[]> = {
  privacy: [
    "ringkasan",
    "data-yang-dikumpulkan",
    "tujuan-pemrosesan",
    "pemrosesan-ai",
    "berbagi-data",
    "penyimpanan-retensi",
    "hak-anda",
    "keamanan",
    "anak-anak",
    "perubahan-kebijakan",
    "kontak-privasi",
  ],
  terms: [
    "penerimaan",
    "akun",
    "penggunaan-yang-diizinkan",
    "langganan-pembayaran",
    "kuota-ai",
    "hak-kekayaan-intelektual",
    "batasan-tanggung-jawab",
    "penghentian",
    "perubahan-syarat",
    "hukum-kontak",
  ],
};
