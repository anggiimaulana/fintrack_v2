/**
 * Canonical copy for `/pricing` (spec doc 04 §A). Kept out of the components so
 * the page, the cards, and the comparison table read one source and no string
 * drifts — same pattern as `copy.ts` / `faq.ts`.
 *
 * Honesty rules baked into the words here: the price shown is the subtotal, tax
 * is a separate line stated plainly, and payment methods describe the real flow
 * (automated gateway + manually verified transfer).
 */

export const PRICING = {
  hero: {
    h1: "Harga FinTrack",
    subhead: "Mulai gratis, naikkan paket saat butuh lebih banyak kuota AI.",
    /** Reused, factual landing chips — no new claims. */
    chips: ["Gratis untuk memulai", "Tanpa kartu kredit", "Data dapat diekspor"],
  },

  cycle: {
    groupLabel: "Pilih siklus penagihan",
    monthly: "Bulanan",
    yearly: "Tahunan",
    perMonth: "/ bulan",
    perYear: "/ tahun",
    yearlyEquivalentPrefix: "Setara",
    yearlyEquivalentSuffix: "/ bulan",
  },

  card: {
    h2: "Paket FinTrack",
    freePriceLabel: "Gratis",
    popularBadge: "Paling Populer",
    freeCta: "Mulai Gratis",
    paidCtaPrefix: "Pilih",
    /** Shown under the price so the subtotal/tax split is never ambiguous. */
    taxNote: "Harga belum termasuk pajak.",
  },

  empty: {
    heading: "Paket belum dapat dimuat",
  },

  comparison: {
    h2: "Perbandingan paket",
    featuresH3: "Perbandingan fitur",
    featuresSubhead: "Fitur yang tersedia di tiap paket, sesuai katalog resmi.",
    limitsH3: "Batas pemakaian",
    limitsSubhead: "Kuota yang dihitung dan diterapkan oleh sistem untuk tiap paket.",
    featureColumn: "Fitur",
    limitColumn: "Batas",
    /** Accessible labels for the check/strip icons. */
    available: "tersedia",
    unavailable: "tidak tersedia",
  },

  payment: {
    eyebrow: "PEMBAYARAN",
    h2: "Cara pembayaran",
    subhead: "Dua jalur yang sama-sama sah. Pilih yang paling nyaman untuk Anda.",
    methods: [
      {
        title: "Gateway otomatis",
        text: "Bayar melalui gateway pembayaran dan langganan aktif otomatis setelah pembayaran berhasil.",
      },
      {
        title: "Transfer manual",
        text: "Transfer lalu unggah bukti pembayaran. Admin memverifikasinya; langganan aktif setelah pembayaran disetujui.",
      },
    ],
    taxNote:
      "Harga paket ditampilkan belum termasuk pajak. Bila pajak berlaku, jumlahnya tampil sebagai baris terpisah dan total terlihat sebelum Anda membayar.",
  },

  faq: {
    eyebrow: "FAQ HARGA",
    h2: "Pertanyaan seputar harga",
    description: "Keberatan paling umum soal biaya, kuota, dan pembayaran — dijawab langsung.",
  },

  cta: {
    h2: "Siap mencoba?",
    subhead: "Gratis untuk memulai. Naikkan paket saat butuh lebih banyak kuota AI.",
    cta: { label: "Mulai Gratis", href: "/register" },
  },
} as const;
