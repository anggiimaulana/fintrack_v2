# Decision Log (ADR) — FinTrack v2

Catatan keputusan arsitektur & teknis signifikan. Format: `K-PXX-NN` atau `ADR-NNN`.
Setiap entri mencakup: keputusan, alasan, alternatif yang ditolak, dan konsekuensi.

---

## ADR-001 — Kunci AI terpusat di super admin (bukan per-user)

- **Tanggal:** 2026-10-11
- **Konteks:** model per-user membuat user awam harus menyiapkan kunci LLM sendiri; biaya tak terkendali.
- **Keputusan:** hapus BYOK; semua trafik lewat pool kunci terpusat (9router) yang dikelola super admin.
- **Alternatif ditolak:** tetap per-user; BYOK opsional.
- **Konsekuensi:** memperbesar blast radius → **wajib** pool + failover + circuit breaker + kill switch.
  Kunci lama diarsipkan (tidak dihapus).

## ADR-002 — Ledger token append-only dengan reserve/commit/release

- **Tanggal:** 2026-10-11
- **Konteks:** metering token harus akurat & adil meski ada concurrency.
- **Keputusan:** `token_ledger` append-only; reserve sebelum panggilan, commit jumlah nyata, release saat gagal;
  `idempotency_key` UNIQUE.
- **Alternatif ditolak:** cek-lalu-tambah (bocor saat konkuren); estimasi lokal saja.
- **Konsekuensi:** kompleksitas tambahan; memberi akurasi & anti double-charge.

## ADR-003 — Uang sebagai integer (bukan float)

- **Tanggal:** 2026-10-11
- **Konteks:** galat pembulatan float merusak invoice & pembukuan.
- **Keputusan:** semua nilai uang = integer rupiah; token = integer.
- **Alternatif ditolak:** `numeric`/float untuk kenyamanan.
- **Konsekuensi:** konversi hanya di layer presentasi.

## ADR-004 — Entitlement dipisah dari role & tidak di JWT

- **Tanggal:** 2026-10-11
- **Konteks:** perubahan hak akses harus berlaku langsung tanpa login/logout & tanpa downtime.
- **Keputusan:** role = identitas (`super_admin`/`user`); entitlement = plan, di-resolve per-request dari
  cache **berversi**.
- **Alternatif ditolak:** entitlement di JWT; cek di frontend saja.
- **Konsekuensi:** butuh bump version + invalidasi cache yang benar.

## ADR-005 — Redis untuk cache bersama & pub/sub

- **Tanggal:** 2026-10-11
- **Konteks:** cache in-memory tidak sinkron lintas instance → menghambat zero downtime multi-instance.
- **Keputusan:** tambah Redis (cache + pub/sub untuk invalidasi & event realtime).
- **Alternatif ditolak:** tetap single-instance; Postgres LISTEN/NOTIFY.
- **Konsekuensi:** satu komponen infra tambahan; memungkinkan skala & konsistensi.

## ADR-006 — Hibrida incremental + modul baru bersih

- **Tanggal:** 2026-10-11
- **Konteks:** sistem sudah berjalan; rewrite berisiko & melanggar zero downtime.
- **Keputusan:** pendekatan Strangler Fig (incremental) + modul baru ber-interface
  (`aigateway`, `billing`, `entitlement`, `payment`, `realtime`, `audit`).
- **Alternatif ditolak:** in-place murni; big-bang rewrite.
- **Konsekuensi:** sementara ada jalur adapter; batas modul jelas.

## ADR-007 — Siklus langganan per-hari (bukan bulan kalender)

- **Tanggal:** 2026-10-11
- **Konteks:** bulan 28/29/30/31 hari membuat tagihan bulanan tidak adil.
- **Keputusan:** siklus = `mulai + N hari` (TZ Asia/Jakarta); reset tiap siklus; tanpa rollover.
- **Alternatif ditolak:** bulan kalender; rollover token.
- **Konsekuensi:** mayoritas pembayaran manual → wajib reminder + grace + dunning.

## ADR-008 — Switch payment gateway berlaku untuk checkout baru

- **Tanggal:** 2026-10-11
- **Konteks:** super admin ingin ganti gateway realtime tanpa deploy.
- **Keputusan:** `settings.payment_gateway` (global) + override per-plan; **transaksi in-flight tetap**
  terikat gateway asalnya.
- **Alternatif ditolak:** re-route transaksi berjalan (berisiko pembayaran hilang).
- **Konsekuensi:** webhook gateway lama harus tetap diproses selama ada transaksi pending.

## ADR-009 — DataTable standar berbasis TanStack Table v8 + URL state

- **Tanggal:** 2026-10-11
- **Konteks:** standar tabel harus seragam di semua halaman & dapat dibagikan lewat URL.
- **Keputusan:** satu komponen `<DataTable>`; state di URL (`?page=&sort=&order=&q=&filter`).
- **Alternatif ditolak:** custom tanpa library; tabel per-halaman.
- **Konsekuensi:** konsistensi & deep-linking; perlu konvensi param bersama.

## ADR-010 — Webhook pembayaran idempoten & terverifikasi

- **Tanggal:** 2026-10-11
- **Konteks:** pembayaran tidak boleh diaktifkan dari klien atau dua kali.
- **Keputusan:** verifikasi signature + verifikasi ulang via Status API + idempotency key.
- **Alternatif ditolak:** percaya halaman sukses klien.
- **Konsekuensi:** aman terhadap replay; butuh penanganan retry gateway.

---

## ADR-011 — Border radius aktif (revisi arah visual dari sudut lancip)

- **Tanggal:** 2026-10-15
- **Konteks:** sistem desain awal menonaktifkan seluruh token radius (`--radius-*: 0px`) dan
  mengandalkan sudut lancip sebagai ciri neo-brutalist. Namun 23 file memakai `rounded-md/lg/xl`
  secara sporadis, menghasilkan UI yang tidak konsisten (sebagian lancip, sebagian tumpul).
- **Keputusan:** aktifkan border radius di seluruh UI.
  - Token `@theme` di `app/globals.css`: `--radius-sm: 4px`, `-md: 6px`, `-lg: 8px`, `-xl: 12px`,
    `-2xl: 16px`, `-3xl: 24px`, `-4xl: 32px`; `--radius: 6px`.
  - Skala pakai: `rounded-sm` (badge, chip kecil), `rounded-md` (tombol, input, select, item, kotak kecil),
    `rounded-lg` (card, dialog, panel), `rounded-full` (avatar, dot, progress).
  - Elemen persegi serba-guna di kode konsumen dibulatkan `rounded-md` (default) / `rounded-lg` (padding ≥ 16px).
  - **Garis pembatas** (`border-b-2`, `border-t-2`, `divide-y`, `<hr>`) tetap **tidak** dibulatkan.
  - Shadow keras `Npx Npx 0` (ciri brutalist) **dipertahankan**.
- **Alternatif ditolak:** kembali ke 0px dan membersihkan 23 file bocor (user memilih radius);
  radius hanya pada komponen terpilih.
- **Konsekuensi:** radius kini bagian resmi sistem desain → komponen baru **wajib** pakai `rounded-*`.
  Dokumen/referensi yang menyebut "sudut lancip" sebagai ciri desain harus dibaca dengan ADR ini.

---

## ADR-012 — Rasio token per plan + lapisan konfigurasi dinamis

- **Tanggal:** 2026-10-15
- **Konteks:** kuota plan diukur dalam token mentah, padahal satu percakapan AI nyata
  memakai ~5.000–10.000 token. Kuota Free 5.000 habis dalam satu chat. Selain itu
  sejumlah kebijakan masih di-hardcode di Go (allowlist setting, pemetaan limit→resolver,
  katalog fitur/limit, mode pembulatan, overhead estimasi, sentinel unlimited).
- **Keputusan:**
  1. **Rasio token per plan** (`plans.token_quota_ratio`): token provider dibagi rasio
     sebelum ditagih ke kuota; **biaya Rupiah tetap dari token nyata**. Rasio di-snapshot
     per siklus (`token_periods.quota_ratio`) sehingga perubahan tidak retroaktif.
  2. **Ledger menyimpan keduanya**: `amount` = unit kuota, `raw_tokens` = token provider.
     Baris lama dibaca via `RawTokensOrEntry()` (tanpa UPDATE — ledger append-only).
  3. **Tiga tabel katalog menjadi DATA**: `setting_definitions`, `metric_definitions`,
     `feature_definitions` menggantikan konstanta kode. Menambah setting/metrik/fitur
     tidak perlu deploy.
  4. **Pembulatan dinamis** (`quota_rounding_mode`) + lantai tagih (`quota_min_charge_units`).
  5. **Invalidasi event-driven** (SSE `config_changed` + `BumpVersion`), bukan TTL.
     TTL 5 menit hanya jaring terakhir.
  6. **Fallback konstanta permanen**: tabel katalog kosong tidak mematikan sistem.
- **Alternatif ditolak:** naikkan kuota saja (angka membengkak, biaya tak terkendali);
  kuota chat terpisah (tidak mengontrol model mahal); cache TTL (tidak segar, beban berulang).
- **Seam yang diakui:** cara menghitung usage (resolver), struktur rute, dan config
  bootstrap tetap di kode. Kode menyediakan **kemampuan**; database menyimpan **kebijakan**.
- **Konsekuensi:** kuota Free kini bertahan ~2 chat (sebelumnya 0). Operator dapat
  mengubah seluruh kebijakan tanpa deploy. Migrasi: `0028` (rasio) + `0029` (katalog).

---

> Keputusan fase-spesifik dicatat di `../../2_Projects/PXX_*/analisis/hasil-analisis-teknis.md`
> dengan format `K-PXX-NN`.
