import type { LucideIcon } from "lucide-react";
import { Camera, Download, LayoutDashboard, MessageSquare, Mic, Target } from "lucide-react";

/**
 * Central content model for the six feature pages.
 *
 * One source of truth for the feature hub, the cross-links, the pricing table,
 * the landing grid, and `app/sitemap.ts` (spec doc 04 §B.2). Slugs are English
 * (URL), all copy is Indonesian. Every page describes a distinct problem, flow,
 * benefit set, and concrete example — never templated filler (spec doc 04 §B.1).
 *
 * `featureWordCount`/`featureUniqueWordCount` power the completeness test that
 * enforces the "≥ 300 words per page" rule from spec doc 04 §B.1.
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

/** A concrete situation where the feature helps (not a testimonial). */
export interface FeatureUseCase {
  title: string;
  text: string;
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
  /** SEO meta description (~150 chars) that contains a target keyword. */
  metaDescription: string;
  /** Opening paragraph under the H1. */
  heroDescription: string;
  /** How the feature works under the hood, in plain language. */
  detail: string[];
  /** The pain this feature removes. */
  problem: string;
  /** Numbered steps to use the feature. */
  howItWorks: string[];
  /** Three to four payoff points. */
  benefits: string[];
  /** Real situations the feature fits, with a short explanation each. */
  useCases: FeatureUseCase[];
  /** Honest limits and things worth knowing before relying on it. */
  notes: string[];
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
    metaDescription:
      "Catat pengeluaran via chat dan tanyakan kondisi keuangan Anda. Asisten keuangan AI berbahasa Indonesia yang menyusun kalimat menjadi transaksi rapi.",
    heroDescription:
      "Tulis transaksi seperti Anda mengirim pesan ke teman. FinTrack membaca kalimatnya, lalu menyusun jumlah, kategori, dan tanggalnya menjadi catatan yang rapi.",
    detail: [
      "Di balik layar, FinTrack memetakan kalimat Anda ke struktur transaksi: jenis pemasukan atau pengeluaran, nominal, kategori, catatan, dan tanggal. Bila ada bagian yang ambigu — misalnya nominal yang disebut tanpa keterangan waktu — FinTrack memilih tafsiran yang paling masuk akal, menampilkannya lebih dulu, lalu menunggu Anda memeriksa sebelum menyimpan.",
      "Karena tersimpan sebagai entri terstruktur, catatan lewat chat langsung dapat dipakai untuk ringkasan di dasbor, grafik tren, dan analisis kesehatan finansial. Anda tidak perlu memindahkan angka ke tempat lain atau merapikannya secara manual.",
      "FinTrack menyimpan konteks percakapan singkat selama sesi berjalan, sehingga pertanyaan lanjutan seperti “kalau yang tadi bagaimana?” tetap dipahami tanpa mengulang seluruh kalimat. Konteks ini terikat pada akun Anda dan tidak dipakai untuk keperluan lain.",
      "Mencatat pemasukan mengikuti pola yang sama: tulis “gaji masuk 8,5 juta” dan FinTrack menandainya sebagai pemasukan, bukan pengeluaran. Pemisahan ini penting agar ringkasan arus kas tetap benar.",
    ],
    problem:
      "Menulis catatan keuangan terasa seperti mengisi formulir. Membuka aplikasi, memilih kategori, mengeklik tanggal, mengetik nominal — lalu mengulanginya untuk setiap pengeluaran. Karena merepotkan, banyak orang menunda mencatat sampai lupa sama sekali. Padahal, catatan yang lengkap justru yang paling berguna.",
    howItWorks: [
      "Buka menu AI dari dasbor dan mulailah menulis seperti biasa.",
      "Sampaikan transaksinya dengan kalimat alami, misalnya “beli kopi 25 ribu tadi siang”.",
      "FinTrack mengenali jenis transaksi, nominal, kategori, dan waktu, lalu mengusulkan satu entri terstruktur.",
      "Periksa sekilas dan simpan. Anda juga dapat bertanya tentang keuangan dan mendapat jawaban dari data Anda sendiri.",
    ],
    benefits: [
      "Tidak perlu menyesuaikan diri dengan formulir — cukup menulis seperti yang Anda biasa lakukan.",
      "Kategori dan tanggal terisi otomatis, sehingga pencatatan tetap konsisten dari hari ke hari.",
      "Bisa dipakai untuk mencatat sekaligus bertanya, dalam satu tempat yang sama.",
      "Tidak memerlukan API key; FinTrack memakai gateway AI terpusatnya sendiri.",
    ],
    useCases: [
      {
        title: "Mencatat cepat setelah membayar",
        text: "Begitu transaksi selesai, tulis satu kalimat dan lanjutkan aktivitas. Tidak ada formulir yang harus diisi berurutan.",
      },
      {
        title: "Bertanya tentang data sendiri",
        text: "Ajukan pertanyaan seperti “berapa pengeluaran saya minggu ini?” dan dapatkan jawaban yang dihitung dari transaksi Anda, bukan perkiraan.",
      },
      {
        title: "Merapikan catatan yang salah",
        text: "Bila ada entri yang keliru kategori atau nominal, perbaiki lewat percakapan yang sama tanpa mencari-cari menu.",
      },
      {
        title: "Mencatat pemasukan",
        text: "Pendapatan dari gaji, proyek lepas, atau penjualan ditulis dengan pola kalimat yang sama seperti pengeluaran.",
      },
      {
        title: "Meninjau tanpa berpindah menu",
        text: "Setelah beberapa transaksi, minta ringkasannya pada percakapan yang sama agar Anda tidak perlu membuka layar lain.",
      },
    ],
    notes: [
      "Jawaban dihitung dari catatan yang Anda masukkan, sehingga kelengkapan catatan menentukan ketepatan jawabannya.",
      "FinTrack menampilkan hasil untuk diperiksa lebih dulu; keputusan akhir menyimpan tetap di tangan Anda.",
      "Tidak perlu menyiapkan API key pribadi. Semua permintaan AI diproses melalui gateway terpusat FinTrack.",
      "Chat tersedia pada semua paket, dengan kuota Token AI / siklus dan Permintaan AI / siklus yang tampil jelas di dasbor.",
      "Chat memahami kalimat, bukan sekadar kata kunci, tetapi tetap dapat salah menafsirkan ungkapan yang sangat ambigu.",
      "Beberapa transaksi dapat ditulis dalam satu pesan; FinTrack memisahkannya menjadi beberapa entri untuk Anda periksa.",
      "Permintaan AI hanya menghasilkan jawaban untuk akun Anda dan tidak dipakai untuk melatih model pihak lain.",
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
    metaDescription:
      "Catat transaksi suara tanpa mengetik. Input suara Indonesia yang mengubah ucapan menjadi entri rapi — berguna saat tangan Anda sedang sibuk.",
    heroDescription:
      "Sampaikan pengeluaran dengan suara saat tangan Anda sedang sibuk — mengemudi, memasak, atau menggendong anak. FinTrack mengubah ucapan menjadi transaksi.",
    detail: [
      "Ucapan Anda diubah menjadi teks lebih dahulu, lalu diproses dengan cara yang sama seperti catatan lewat chat. Hasilnya tetap berupa entri terstruktur dengan nominal, kategori, dan tanggal yang dapat Anda telusuri kembali.",
      "FinTrack dirancang untuk bahasa Indonesia sehari-hari. Sapaan, angka yang diucapkan, serta istilah umum seperti “makan siang”, “bensin”, atau “parkir” tetap dikenali tanpa daftar perintah khusus.",
      "Karena hasil transkripsi tetap melewati pemeriksaan yang sama seperti input chat, Anda dapat memperbaiki kata yang salah dengar langsung di kolom teks sebelum menyimpan. Tidak ada yang tersimpan otomatis tanpa persetujuan Anda.",
      "Untuk nominal besar, menyebutkan angka dengan jelas — misalnya “satu juta dua ratus ribu” — membantu FinTrack menangkap jumlah yang tepat sejak awal, sehingga koreksi menjadi jarang diperlukan.",
      "Input suara juga berguna ketika Anda ingin menambah beberapa transaksi sekaligus dalam perjalanan. Ucapkan satu per satu; setiap ucapan diproses menjadi entri terpisah sehingga Anda tetap dapat memeriksa dan mengoreksinya sebelum menyimpan semuanya.",
    ],
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
      "Cocok untuk transaksi kecil yang terjadi sepanjang hari dan mudah terlupa.",
      "Bahasa Indonesia dipahami apa adanya, tanpa perintah atau kata kunci khusus.",
      "Kuota suara ditampilkan jelas pada label “Permintaan suara / hari”.",
    ],
    useCases: [
      {
        title: "Saat berada di perjalanan",
        text: "Biaya parkir, tol, atau bahan bakar bisa dicatat tanpa mengetik sambil mengemudi atau menunggu kendaraan.",
      },
      {
        title: "Sambil mengerjakan hal lain",
        text: "Di dapur, saat menggendong anak, atau ketika kedua tangan penuh, cukup ucapkan transaksinya sekilas.",
      },
      {
        title: "Menangkap transaksi kecil",
        text: "Jajan, air minum, dan ongkos memang kecil, tetapi bila tercatat semuanya, ringkasan bulan Anda menjadi utuh.",
      },
      {
        title: "Mencatat saat makan",
        text: "Ucapkan biaya makanan begitu selesai membayar, sebelum detailnya terlupa di tengah kesibukan berikutnya.",
      },
      {
        title: "Menambah keterangan singkat",
        text: "Tambahkan catatan seperti “untuk arisan” atau “bayar listrik” pada transaksi yang baru diucapkan.",
      },
    ],
    notes: [
      "Kualitas mikrofon dan keramaian di sekitar memengaruhi ketepatan pengenalan suara, jadi ucapkan dengan wajar dan cukup jelas.",
      "Kuota suara dihitung per hari dan ditampilkan pada label “Permintaan suara / hari”.",
      "Input suara tersedia pada paket Pro dan Business.",
      "Bila pengenalan suara kurang tepat, Anda selalu dapat memperbaiki teksnya atau beralih mencatat lewat chat.",
      "Angka yang diucapkan dengan jelas lebih akurat daripada yang disebut terburu-buru.",
      "Bila transkripsi meleset, ubah teksnya seperti biasa; tidak ada data yang tersimpan tanpa tinjauan.",
      "Sebut satu transaksi dalam satu ucapan agar pemisahan entri tetap bersih.",
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
    metaDescription:
      "Scan struk otomatis dari foto belanja Anda. OCR struk belanja membaca item dan total, lalu menyimpannya menjadi transaksi tanpa mengetik ulang.",
    heroDescription:
      "Foto struk belanja Anda. FinTrack membaca item beserta totalnya, lalu menyimpannya sebagai transaksi — tanpa mengetik ulang satu per satu.",
    detail: [
      "Ketika struk difoto, FinTrack membaca teks pada gambar, mengenali baris item, harga, jumlah, dan total, lalu menyusunnya menjadi transaksi. Struk yang rata, cukup terang, dan tidak terpotong memberi hasil paling baik.",
      "Anda tetap dapat memperbaiki item yang salah baca sebelum menyimpan. Dengan begitu, rekap akhir tetap mengikuti struk asli dan tidak melenceng dari kenyataan.",
      "FinTrack membaca struktur struk pada gambar, bukan mencocokkannya dengan daftar harga toko tertentu. Karena itu, struk dari merchant berbeda tetap dapat dibaca selama teksnya cukup jelas.",
      "Setelah dibaca, item disimpan sebagai transaksi beserta kategorinya. Anda dapat menggabungkan atau memisahkannya sesuai kebutuhan sebelum menyimpan.",
      "Untuk hasil terbaik, letakkan struk di permukaan datar dan pastikan seluruh bagiannya masuk ke dalam bingkai. Bila struk panjang, Anda dapat memotretnya secara bertahap lalu menggabungkan hasilnya menjadi satu transaksi belanja.",
    ],
    problem:
      "Mengetik ulang deretan belanjaan dari struk itu melelahkan dan rawan salah. Satu nomor tertukar, dan total Anda tidak lagi cocok dengan kenyataan. Akibatnya banyak orang memilih tidak mencatat belanja besar sama sekali.",
    howItWorks: [
      "Buka menu AI dan pilih opsi kamera.",
      "Ambil foto struk yang cukup jelas dan tidak terpotong.",
      "FinTrack membaca daftar item, kategori, dan total belanja.",
      "Periksa hasilnya, koreksi bila perlu, lalu simpan.",
    ],
    benefits: [
      "Menghemat waktu untuk belanja dengan banyak item sekaligus.",
      "Angka yang dicatat mengikuti struk, sehingga rekap tetap akurat.",
      "Kategori terisi otomatis mengikuti jenis belanja yang terbaca.",
      "Cocok dipadukan dengan catatan harian lewat chat dan suara.",
    ],
    useCases: [
      {
        title: "Belanja bulanan",
        text: "Sekali foto, seluruh daftar belanja beserta totalnya tercatat tanpa mengetik satu baris pun.",
      },
      {
        title: "Pembelian bernilai besar",
        text: "Struk dengan banyak item tetap rapi, dan totalnya mudah dicocokkan dengan bukti pembayaran.",
      },
      {
        title: "Pengeluaran usaha kecil",
        text: "Bukti belanja usaha tersimpan sebagai transaksi, sehingga pembukuan sederhana tidak menumpuk di akhir bulan.",
      },
      {
        title: "Mencocokkan dengan bukti bayar",
        text: "Setelah struk tercatat, bandingkan totalnya dengan mutasi atau bukti pembayaran untuk memastikan keduanya cocok.",
      },
      {
        title: "Merapikan struk yang menumpuk",
        text: "Struk yang sempat terkumpul dapat dipindai satu per satu saat senggang, tanpa menghitung ulang totalnya.",
      },
    ],
    notes: [
      "Struk yang pudar, terlipat, atau kurang cahaya dapat menurunkan ketepatan pembacaan; ambil ulang bila hasilnya jauh dari struk.",
      "Kuota pindai dihitung per siklus dan ditampilkan pada label “Pindai struk / siklus”.",
      "Pindai struk tersedia pada paket Pro dan Business.",
      "Hasil pindai selalu dapat diedit, jadi Anda tidak terpaku pada pembacaan otomatis.",
      "Satu foto sebaiknya memuat satu struk agar batas antar baris tidak tercampur.",
      "Jumlah item dapat berbeda dari struk bila ada baris yang terbaca sebagian; periksa daftarnya sebelum menyimpan.",
      "Gunakan pencahayaan yang cukup dan hindari bayangan tangan yang menutupi angka.",
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
    metaDescription:
      "Dashboard keuangan dengan grafik pengeluaran dan ringkasan arus kas. Lihat pemasukan, pengeluaran, dan tren bulanan dalam satu tampilan.",
    heroDescription:
      "Pantau pemasukan, pengeluaran, dan trennya dalam tampilan yang mudah dibaca. Satu layar untuk menjawab “bagaimana keuangan saya bulan ini?”.",
    detail: [
      "Setiap transaksi yang Anda simpan langsung memperbarui ringkasan di dasbor: total pemasukan, total pengeluaran, dan selisihnya pada periode berjalan. Grafik tren menunjukkan perubahan dari bulan ke bulan, sementara rincian per kategori membantu Anda melihat ke mana uang paling banyak pergi.",
      "Semua angka berasal dari catatan Anda sendiri. Tidak ada rata-rata industri atau asumsi dari luar yang disisipkan, sehingga yang Anda lihat adalah keadaan keuangan Anda sebenarnya.",
      "Dasbor memisahkan periode dengan jelas, sehingga Anda dapat membandingkan bulan berjalan dengan bulan sebelumnya tanpa menghitung sendiri. Bila ada bulan dengan pengeluaran tidak biasa, selisihnya langsung terlihat.",
      "Rincian kategori menampilkan porsi terbesar lebih dahulu, sehingga Anda dapat memusatkan perhatian pada pos yang paling memengaruhi anggaran.",
      "Dasbor juga menandai transaksi terbaru yang baru Anda tambahkan, sehingga Anda dapat memastikan hasil pencatatan lewat chat, suara, atau struk masuk ke ringkasan pada periode yang benar.",
    ],
    problem:
      "Angka berserakan, sulit melihat gambaran besarnya. Ketika catatan hanya berupa daftar panjang, Anda tahu setiap transaksi tetapi tidak tahu arah keuangannya. Pola pengeluaran yang berulang pun tidak terlihat.",
    howItWorks: [
      "Semua transaksi yang Anda catat langsung tampil di dasbor.",
      "Lihat ringkasan pemasukan, pengeluaran, dan selisihnya pada periode berjalan.",
      "Telusuri grafik tren untuk melihat perubahan dari bulan ke bulan.",
      "Klik kategori untuk memahami ke mana uang Anda paling banyak pergi.",
    ],
    benefits: [
      "Gambaran menyeluruh tanpa perlu membuka spreadsheet.",
      "Perbandingan tren antar periode yang mudah dibaca sekilas.",
      "Ringkasan ikut terbarui setiap kali Anda menambah transaksi.",
      "Menjadi dasar untuk analisis kesehatan finansial.",
    ],
    useCases: [
      {
        title: "Menutup bulan",
        text: "Lihat sekilas apakah pengeluaran bulan ini naik atau turun dibanding bulan sebelumnya, tanpa menghitung manual.",
      },
      {
        title: "Menemukan kebocoran halus",
        text: "Telusuri kategori terbesar dan temukan pengeluaran berulang yang sebelumnya tidak Anda sadari.",
      },
      {
        title: "Menjawab pertanyaan cepat",
        text: "Saat ingin tahu sisa uang bulan ini atau total pengeluaran mingguan, jawabannya ada di satu layar.",
      },
      {
        title: "Memantau anggaran berjalan",
        text: "Lihat kategori mana yang sudah mendekati batas, sehingga penyesuaian bisa dilakukan sebelum akhir bulan.",
      },
      {
        title: "Melihat dampak transaksi besar",
        text: "Setelah pembelian penting, periksa bagaimana perubahan itu memengaruhi ringkasan bulan berjalan.",
      },
    ],
    notes: [
      "Dasbor hanya menampilkan apa yang sudah Anda catat; transaksi yang belum dimasukkan tidak akan muncul.",
      "Angka di dasbor dapat berbeda dari saldo rekening bila ada transaksi yang belum tercatat.",
      "Dasbor tersedia pada semua paket, termasuk yang gratis.",
      "Pilih periode yang ingin ditinjau agar perbandingan tetap relevan dengan kebutuhan Anda.",
      "Periode mengikuti tanggal transaksi, bukan tanggal pencatatan, sehingga susunan tetap konsisten bila Anda mengisi catatan terlambat.",
      "Bila kategori terasa kurang tepat, perbaiki transaksi terkait dan ringkasan ikut menyesuaikan.",
      "Grafik memerlukan beberapa periode data sebelum polanya terlihat bermakna.",
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
    metaDescription:
      "Analisis keuangan pribadi yang memberi skor keuangan dan status kesehatan finansial — dari Sangat Baik sampai Kritis, dihitung dari data Anda.",
    heroDescription:
      "Penilaian meninjau pola pengeluaran, tabungan, dan kebiasaan Anda, lalu memberi skor beserta status yang jelas — dari Sangat Baik hingga Kritis.",
    detail: [
      "Analisis meninjau beberapa hal sekaligus: seberapa besar pengeluaran dibanding pemasukan, seberapa konsisten Anda menabung, serta bagaimana polanya dari waktu ke waktu. Hasilnya dirangkum menjadi satu skor yang mudah diingat.",
      "Skor dipetakan ke salah satu status: Sangat Baik, Baik, Cukup, Boros, atau Kritis. Karena memakai data terbaru dari catatan Anda, status ini ikut berubah ketika kebiasaan mencatat dan pola pengeluaran Anda berubah.",
      "Skor memberi satu angka untuk diingat, sedangkan status memberi konteks yang mudah dibaca. Keduanya bukan penilaian mutlak; tujuannya membantu Anda melihat perubahan dari waktu ke waktu.",
      "Bila ingin tahu mengapa skor bergerak, tanyakan lewat chat. Jawabannya merujuk pada pola yang terlihat pada data Anda, lalu menyarankan bagian yang paling wajar diperbaiki lebih dahulu.",
      "Status dirancang untuk dibaca cepat: satu skor, satu label, dan penjelasan singkat tentang faktor yang paling berpengaruh. Dari sana Anda dapat memutuskan langkah kecil berikutnya, misalnya menekan satu kategori atau menambah alokasi tabungan.",
    ],
    problem:
      "Sulit menilai apakah kebiasaan keuangan Anda membaik. Tanpa ukuran, setiap bulan terasa sama. Anda mencatat rajin, tetapi tidak tahu apakah arahnya benar atau justru memburuk perlahan.",
    howItWorks: [
      "FinTrack meninjau transaksi, tabungan, dan pola yang terbentuk dari waktu ke waktu.",
      "Metrik dihitung menjadi skor tunggal yang mudah dipahami.",
      "Skor dipetakan ke salah satu status: Sangat Baik, Baik, Cukup, Boros, atau Kritis.",
      "Anda dapat menanyakan alasannya lewat chat untuk memahami bagian yang perlu diperbaiki.",
    ],
    benefits: [
      "Ukuran yang jelas, bukan sekadar perasaan tiap akhir bulan.",
      "Status disertai label teks, tidak bergantung pada warna saja.",
      "Membantu memutuskan kapan menahan diri dan kapan menabung lebih banyak.",
      "Terhubung dengan dasbor, sehingga analisis selalu memakai data terbaru.",
    ],
    useCases: [
      {
        title: "Pemeriksaan rutin bulanan",
        text: "Jadikan skor dan status sebagai kebiasaan tiap akhir bulan untuk melihat arah keuangan Anda.",
      },
      {
        title: "Sebelum keputusan besar",
        text: "Ketahui kondisi Anda lebih dulu sebelum mengambil cicilan, kredit, atau pengeluaran besar lainnya.",
      },
      {
        title: "Memahami penyebab skor",
        text: "Tanyakan alasannya lewat chat untuk mengetahui kebiasaan mana yang paling menekan skor Anda.",
      },
      {
        title: "Menetapkan target sederhana",
        text: "Jadikan status bulan ini titik awal, lalu lihat apakah status itu membaik pada pemeriksaan berikutnya.",
      },
      {
        title: "Membandingkan sebelum dan sesudah",
        text: "Setelah mengubah kebiasaan menabung, pantau apakah perubahan itu tercermin pada skor.",
      },
    ],
    notes: [
      "Status selalu disertai label teks, bukan hanya warna, agar tetap terbaca oleh semua orang.",
      "Skor adalah alat bantu refleksi, bukan pengganti nasihat keuangan profesional.",
      "Analisis kesehatan finansial tersedia pada semua paket.",
      "Karena dihitung dari catatan Anda, semakin lengkap catatannya semakin mewakili kondisi sebenarnya.",
      "Skor memerlukan riwayat catatan; akun yang baru mulai akan memperoleh gambaran lebih baik setelah beberapa periode.",
      "Rasio tabungan dihitung dari data yang Anda masukkan, jadi pastikan pemasukan juga tercatat.",
      "Gunakan analisis ini bersama dasbor untuk melihat angka pendukung di balik status yang muncul.",
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
    metaDescription:
      "Laporan keuangan otomatis dan ekspor data keuangan. Pilih periode, tinjau ringkasannya, lalu unduh data Anda kapan saja tanpa terkunci di aplikasi.",
    heroDescription:
      "Hasilkan laporan ringkas untuk periode tertentu, lalu unduh data Anda kapan saja. Data keuangan Anda tetap milik Anda.",
    detail: [
      "Pilih periode yang ingin ditinjau, dan FinTrack menyusun ringkasan pemasukan, pengeluaran, kategori terbesar, serta daftar transaksinya. Laporan dapat dibaca langsung di layar, lalu diunduh untuk disimpan atau dibagikan.",
      "Karena datanya milik Anda, tidak ada batas untuk memakainya di luar FinTrack. Anda dapat melampirkannya pada pembukuan usaha kecil, menyimpannya sebagai arsip pribadi, atau mengolahnya kembali sesuai kebutuhan.",
      "Laporan menampilkan angka ringkas di bagian atas, diikuti daftar transaksi yang menyusunnya. Dengan begitu Anda dapat menelusuri kembali dari mana setiap total berasal.",
      "Berkas yang dihasilkan memakai kolom yang rapi — tanggal, jenis, kategori, keterangan, dan nominal — sehingga mudah dibuka kembali di aplikasi tabel atau diserahkan kepada pihak lain.",
      "Anda dapat menghasilkan laporan untuk kebutuhan berbeda dari data yang sama: rekap bulanan untuk pemantauan rutin, atau rentang khusus ketika ada keperluan tertentu. Tidak ada format yang dikunci, sehingga berkasnya mudah disesuaikan di luar FinTrack.",
      "Sebelum mengunduh, tinjau angka ringkasannya lebih dahulu. Bila ada transaksi yang tampak janggal, perbaiki catatannya lalu hasilkan ulang laporan agar berkas yang Anda simpan sudah final.",
    ],
    problem:
      "Data terkunci di aplikasi, sulit dipakai di tempat lain. Saat perlu melampirkan rekap untuk urusan pajak, pengajuan, atau pembukuan usaha kecil, daftar transaksi hanya bisa dilihat di layar.",
    howItWorks: [
      "Pilih periode yang ingin Anda tinjau.",
      "FinTrack menyusun ringkasan pemasukan, pengeluaran, dan kategori terbesar.",
      "Tinjau laporannya langsung di layar.",
      "Ekspor atau unduh datanya untuk disimpan dan dipakai di luar FinTrack.",
    ],
    benefits: [
      "Rekap siap pakai tanpa menyusunnya sendiri di spreadsheet.",
      "Data dapat diunduh kapan saja — tidak terkunci di dalam aplikasi.",
      "Membantu keperluan pembukuan pribadi maupun usaha kecil.",
      "Melengkapi dasbor dan analisis kesehatan finansial.",
    ],
    useCases: [
      {
        title: "Persiapan pajak",
        text: "Kumpulkan rekap pengeluaran dan pemasukan satu periode dalam satu berkas yang mudah diserahkan.",
      },
      {
        title: "Pembukuan usaha kecil",
        text: "Serahkan ringkasan bulanan kepada pembukuan tanpa perlu menghitung atau menyalin ulang angkanya.",
      },
      {
        title: "Arsip pribadi",
        text: "Simpan salinan data keuangan Anda sendiri sebagai cadangan, terpisah dari aplikasi.",
      },
      {
        title: "Meninjau satu kuartal",
        text: "Pilih rentang tiga bulan untuk melihat pola musiman, misalnya pengeluaran yang membengkak pada periode tertentu.",
      },
      {
        title: "Menyiapkan berkas untuk pihak lain",
        text: "Saat perlu menyerahkan rekap kepada mitra atau pembukuan, cukup unduh periode yang relevan.",
      },
      {
        title: "Membandingkan dua periode",
        text: "Hasilkan dua rentang berbeda untuk melihat perubahan pengeluaran antar bulan secara berdampingan.",
      },
    ],
    notes: [
      "Ketersediaan laporan dan ekspor mengikuti paket; keduanya tersedia pada Pro dan Business.",
      "Periksa kembali data yang diekspor sebelum dipakai untuk keperluan resmi atau pajak.",
      "Berkas yang diunduh berisi catatan Anda sendiri dan tidak mencakup data akun lain.",
      "Struktur berkas dibuat sederhana agar mudah dibuka di aplikasi tabel mana pun.",
      "Rentang periode menentukan isi berkas, jadi pilih rentang yang benar sebelum mengunduh.",
      "Data yang sudah diekspor tidak lagi terikat pada akun Anda; simpan berkasnya di tempat yang aman.",
      "Ekspor tidak mengubah data di dalam FinTrack; catatan aslinya tetap utuh.",
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

/** Every prose string that counts toward a feature page's word total. */
function contentStrings(f: FeaturePage): string[] {
  return [
    f.title,
    f.navLabel,
    f.tagline,
    f.heroDescription,
    ...f.keywords,
    ...f.detail,
    f.problem,
    ...f.howItWorks,
    ...f.benefits,
    ...f.useCases.flatMap((useCase) => [useCase.title, useCase.text]),
    ...f.notes,
    f.example.prompt,
    f.example.result,
  ];
}

/** Split prose into comparable lowercase word tokens. */
function tokenize(strings: string[]): string[] {
  return strings
    .join(" ")
    .toLowerCase()
    .replace(/[^a-z0-9\s]+/g, " ")
    .split(/\s+/)
    .filter((token) => token.length > 1);
}

/** Every prose token of a feature page, in order (used by the content tests). */
export function featureWords(feature: FeaturePage): string[] {
  return tokenize(contentStrings(feature));
}

/** Total word count of a feature page's prose. */
export function featureWordCount(feature: FeaturePage): number {
  return featureWords(feature).length;
}

/** Number of distinct words on a feature page (spec doc 04 §B.1: "kata unik"). */
export function featureUniqueWordCount(feature: FeaturePage): number {
  return new Set(featureWords(feature)).size;
}
