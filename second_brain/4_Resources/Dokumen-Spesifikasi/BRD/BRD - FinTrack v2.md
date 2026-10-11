# BRD — FinTrack v2: Platform Berlangganan & AI Gateway

| Field | Isi |
|---|---|
| **Nama dokumen** | Business Requirements Document (BRD) |
| **Produk** | FinTrack v2 — aplikasi manajemen keuangan pribadi |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Status** | Disetujui untuk perencanaan |
| **Pemilik produk** | Pemilik produk FinTrack |
| **Disusun oleh** | Analisis brainstorming bersama pemilik produk |
| **Konteks repo** | `D:/project/finance/fintrack_v2` |

---

## 1. Ringkasan Eksekutif

FinTrack v2 hari ini adalah aplikasi keuangan pribadi yang **sudah berjalan** (auth Go-native, RAG
pgvector, dashboard user/admin, AI chat berbasis API key per-user). Dokumen ini mendefinisikan
perubahan strategis: mengubah FinTrack dari aplikasi self-host menjadi **produk berlangganan (SaaS)**
dengan **kunci AI terpusat**, **kuota token dinamis**, **pembayaran (Midtrans + manual)**, dan
**kontrol super admin penuh**.

Perubahan ini menyentuh lima area: (1) AI Gateway terpusat, (2) Subscription & sistem token,
(3) Dashboard & manajemen super admin, (4) RBAC berbasis entitlement, (5) standar teknis lintas-halaman
(data table, caching realtime, zero downtime).

## 2. Latar Belakang & Masalah Bisnis

### 2.1 Kondisi saat ini

- **Model kunci AI per-user**: setiap user harus punya kunci LLM sendiri → tidak ramah user awam,
  biaya tidak terkendali, tidak bisa ditagih.
- **Tanpa monetisasi**: tidak ada plan, subscription, kuota, atau pembayaran.
- **Kontrol admin terbatas**: belum ada manajemen plan/subscription/pembayaran/token/gateway.
- **Standar data tidak seragam**: hanya tabel transaksi yang berpaginasi; tabel lain memuat seluruh data.
- **Cache backend in-memory**: tidak sinkron lintas instance → menghambat zero-downtime multi-instance.

### 2.2 Masalah bisnis yang diselesaikan

| # | Masalah | Dampak saat ini |
|---|---|---|
| M1 | Tidak ada sumber pendapatan | Produk tidak berkelanjutan secara finansial |
| M2 | Biaya LLM tak terkendali & tak terukur | Risiko kebocoran biaya operasional |
| M3 | User awam kesulitan menyiapkan kunci LLM | Konversi & retensi rendah |
| M4 | Tidak ada kontrol platform | Tidak bisa membatasi/menagih pemakaian |
| M5 | Tidak siap skala produksi | Deploy = risiko downtime |

## 3. Tujuan Bisnis & Metrik Keberhasilan

### 3.1 Tujuan

- **G1** — Menghasilkan pendapatan berulang (MRR) dari langganan.
- **G2** — Mengendalikan biaya LLM melalui metering token & pool kunci terpusat.
- **G3** — Menyederhanakan pengalaman user (tanpa kunci LLM sendiri).
- **G4** — Memberi super admin kendali penuh & auditable atas plan, harga, dan gateway AI.
- **G5** — Menyiapkan platform untuk skala & deploy tanpa downtime.

### 3.2 Metrik keberhasilan (contoh target awal)

| Metrik | Target awal | Cara ukur |
|---|---|---|
| Konversi free → berbayar | ≥ 5% | `subscriptions` / user terdaftar |
| MRR | sesuai target bisnis | jumlah pembayaran sukses per bulan |
| Payment success rate | ≥ 95% | `payments` sukses / total |
| Biaya LLM per user berbayar | < 30% harga plan | token usage × harga dasar |
| Waktu respon AI (p95) | < 3s (chat) | metrik gateway |
| Downtime deploy | 0 menit | log deploy |
| Uptime fitur AI | ≥ 99.5% | failover & health check |

## 4. Ruang Lingkup

### 4.1 Dalam ruang lingkup

1. **AI Gateway terpusat** — pool endpoint 9router, routing per-use-case, failover, circuit breaker,
   kill switch, rotasi kunci, metering token.
2. **Subscription & token** — plan dinamis, entitlement (fitur + limit), siklus per-hari, ledger token
   (reserve/commit/release), harga dinamis (base + markup), upgrade/downgrade.
3. **Pembayaran** — Midtrans (Snap) + manual (upload bukti), switch gateway realtime, invoice, prorata,
   grace & dunning.
4. **Super admin** — dashboard + manajemen (user, plan, subscription, pembayaran, token, gateway, audit,
   settings).
5. **RBAC berbasis entitlement** — penegakan backend, cache berversi, realtime tanpa login/logout.
6. **Standar lintas-halaman** — DataTable (pagination 20, sort, search, filter, penomoran, URL state),
   caching realtime, chart interaktif, cursor pointer, SSE.

### 4.2 Di luar ruang lingkup (YAGNI — ditunda)

- Multi-tenant / organisasi / plan keluarga.
- Rollover token antar-periode.
- PPN aktif (hanya "siap" secara struktur).
- Multi-mata-uang (hanya IDR).
- Auto-recurring paksa untuk semua metode.
- Pitch deck / materi investor.
- Mobile app native.

### 4.3 Asumsi

- Stack tetap: Go + Gin, PostgreSQL 17 + pgvector, Next.js 16, Docker Compose.
- 9router tersedia dan bersifat OpenAI-compatible.
- Kunci lama (`ENCRYPTION_KEY`) dipertahankan agar data terenkripsi tetap terbaca.
- TZ perhitungan = `Asia/Jakarta`.

### 4.4 Batasan

- Tidak ada rewrite besar; pendekatan **hibrida incremental + modul bersih**.
- Zero downtime wajib: migrasi expand/contract + feature flag.
- Semua data finansial user tidak boleh hilang karena alasan billing.

## 5. Pemangku Kepentingan

| Peran | Kepentingan |
|---|---|
| **Pemilik produk** | Pendapatan, kontrol biaya, kontrol penuh atas plan/harga/gateway |
| **Super admin** | Mengelola user, plan, langganan, pembayaran, kunci AI, audit |
| **User (tenant)** | Kemudahan pakai, transparansi kuota, keadilan harga, keandalan |
| **Tim teknis** | Arsitektur bersih, dapat diuji, zero downtime |
| **Regulator (UU PDP)** | Kepatuhan data pribadi |

## 6. Kebutuhan Bisnis (Business Requirements)

| ID | Kebutuhan | Prioritas | Pilar |
|---|---|---|---|
| **BR-01** | Kunci AI dipusatkan di super admin; user tidak lagi mengelola kunci sendiri | Wajib | 1 |
| **BR-02** | Super admin dapat menambah/mengubah endpoint & kunci 9router secara dinamis (tanpa deploy) | Wajib | 1 |
| **BR-03** | Routing AI dapat dibedakan per kebutuhan (chat, vision, voice, embedding, reasoning) | Wajib | 1 |
| **BR-04** | Sistem tahan gangguan: jika satu endpoint/kunci gagal, layanan tetap jalan (failover) | Wajib | 1 |
| **BR-05** | Tersedia plan/berlangganan dinamis (CRUD oleh super admin) | Wajib | 2 |
| **BR-06** | Setiap plan punya kuota token dan batas fitur yang dapat diatur | Wajib | 2 |
| **BR-07** | Pemakaian token tercatat akurat & adil, tidak bocor saat konkuren | Wajib | 2 |
| **BR-08** | Harga token dinamis: base per 1.000 token + markup (kali & persen) + pengali modality | Wajib | 2 |
| **BR-09** | Siklus langganan per-hari; kuota direset tiap awal siklus | Wajib | 2 |
| **BR-10** | Upgrade instan (prorata), downgrade berlaku akhir siklus tanpa kehilangan data | Wajib | 2 |
| **BR-11** | Pembayaran via Midtrans dan manual (unggah bukti transfer) | Wajib | 2 |
| **BR-12** | Super admin dapat mengganti payment gateway secara realtime tanpa deploy | Wajib | 2 |
| **BR-13** | Verifikasi pembayaran aman (webhook terverifikasi + idempoten) | Wajib | 2 |
| **BR-14** | Invoice dasar, siap ditambah PPN di masa depan | Sebaiknya | 2 |
| **BR-15** | Dashboard super admin dengan metrik bisnis & operasional | Wajib | 3 |
| **BR-16** | Manajemen lengkap: user, plan, subscription, pembayaran, token, gateway, audit, settings | Wajib | 3 |
| **BR-17** | Semua aksi super admin tercatat auditable (sebelum→sesudah) | Wajib | 3 |
| **BR-18** | Hak akses fitur ditentukan oleh plan (entitlement), bukan role | Wajib | 4 |
| **BR-19** | Perubahan plan/hak akses berlaku langsung tanpa login/logout & tanpa downtime | Wajib | 4 |
| **BR-20** | Penegakan hak akses di backend (otoritatif); UI hanya cermin | Wajib | 4 |
| **BR-21** | Semua tabel: pagination 20, sorting, searching, filtering, penomoran, parameter URL | Wajib | 5 |
| **BR-22** | Caching realtime: fetch hanya data yang berubah saat create/update/delete | Wajib | 5 |
| **BR-23** | Chart/diagram interaktif & menarik | Sebaiknya | 5 |
| **BR-24** | Semua elemen interaktif memakai cursor pointer | Sebaiknya | 5 |
| **BR-25** | Data realtime & zero downtime | Wajib | 5 |

## 7. Aturan Bisnis (Business Rules)

- **RB-01** — Siklus langganan dihitung per-hari kalender: `mulai + N hari`, TZ `Asia/Jakarta`.
- **RB-02** — Kuota token direset tiap awal siklus; **tidak ada rollover**.
- **RB-03** — Upgrade: bayar selisih prorata sisa siklus, aktif instan.
- **RB-04** — Downgrade: berlaku akhir siklus, tanpa refund, **soft-limit** (tidak hapus data).
- **RB-05** — Pembayaran manual masuk antrean review; super admin menyetujui/menolak.
- **RB-06** — Pembayaran dianggap sah hanya setelah **webhook terverifikasi** (Midtrans) atau
  **approval super admin** (manual).
- **RB-07** — Grace period (mis. 3 hari) sebelum turun ke free bila perpanjangan gagal.
- **RB-08** — Semua nilai uang disimpan sebagai **integer rupiah**.
- **RB-09** — Kuota token: reserve sebelum panggilan AI, commit jumlah nyata, release bila gagal.
- **RB-10** — Setiap mutasi uang/token idempoten (kunci idempotensi unik).

## 8. Analisis Kompetitor & Pasar (ringkas)

| Produk | Model | Pelajaran |
|---|---|---|
| **YNAB** | Langganan tahunan/bulanan, tanpa free tier jangka panjang | Harga sederhana, nilai jelas |
| **Money Lover / Financisto** | Freemium dengan batas fitur | Freemium menaikkan akuisisi |
| **ChatGPT Plus / Claude Pro** | Langganan + batas pemakaian | Kuota token harus transparan |
| **API-metered (OpenAI/Anthropic)** | Bayar per pemakaian | Metering harus akurat & idempoten |

**Kesimpulan:** kombinasi **freemium + kuota token + fitur berbasis plan** adalah pola yang terbukti.
Transparansi kuota dan keadilan harga adalah kunci retensi.

## 9. Risiko Bisnis & Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Biaya LLM melebihi pendapatan | Rugi per user | Metering akurat, markup, batas keras, kill switch |
| 9router/gateway down | Fitur AI mati | Pool + failover + circuit breaker + kill switch |
| Pembayaran gagal/dunning tinggi | Churn & pendapatan hilang | Reminder, grace, dunning, opsi manual |
| Sengketa tagihan | Ketidakpercayaan | Ledger append-only, invoice, audit log |
| Kebocoran kunci AI | Biaya & reputasi | Enkripsi, rotasi, audit, least privilege |
| Pelanggaran UU PDP | Sanksi hukum | Kebijakan retensi, dasar hukum, hak subjek data |
| Cache salah → akses gratis | Kehilangan pendapatan | Cache berversi + TTL pendek + backend otoritatif |

## 10. Analisis Biaya-Manfaat (kualitatif)

**Manfaat:** pendapatan berulang, biaya LLM terkendali, pengalaman user lebih baik, kontrol penuh,
skalabilitas produksi, kepercayaan lewat transparansi & audit.

**Biaya:** pembangunan 7 fase (P02–P08), penambahan Redis, kompleksitas operasional (dunning,
review pembayaran, rotasi kunci), beban kepatuhan.

**Kesimpulan:** manfaat strategis (monetisasi + kontrol biaya) jauh melebihi biaya pembangunan,
dengan catatan disiplin zero-downtime dan keamanan pembayaran wajib dijaga.

## 11. Persyaratan Kepatuhan

- **UU No. 27/2022 (PDP)** — data billing = data pribadi: dasar hukum, retensi, hak akses/hapus.
- **PCI-DSS** — gunakan Snap/hosted Midtrans (SAQ-A); jangan menyimpan data kartu.
- **Retensi pajak** — invoice disimpan (umumnya 10 tahun); rekonsiliasi dengan hak hapus PDP
  (anonimisasi, bukan hapus, bila konflik).

## 12. Rencana Peluncuran (ringkas)

7 fase di `second_brain/2_Projects/` (P02–P08). Lihat `../PRD/PRD - FinTrack v2.md` dan
`../../0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/05-peta-jalan-awal.md`.

## 13. Persetujuan

| Peran | Nama | Tanggal | Tanda |
|---|---|---|---|
| Pemilik produk | — | 2026-10-11 | ✅ (brainstorming disetujui) |
| Arsitek teknis | — | 2026-10-11 | ✅ |

## Lampiran — Peta BR ke Fase

| Fase | BR terkait |
|---|---|
| P02 | BR-17, BR-21, BR-22, BR-25 |
| P03 | BR-01..BR-04 |
| P04 | BR-05..BR-10, BR-18..BR-20 |
| P05 | BR-11..BR-14 |
| P06 | BR-15, BR-16 |
| P07 | BR-18..BR-25 |
| P08 | semua (hardening) |
