# 06 — Jawaban & Rekomendasi atas Pertanyaan Terbuka

Dokumen ini **melengkapi** `04-pertanyaan-terbuka.md`. Setiap pertanyaan dijawab dengan **rekomendasi
konkret**, alasan, dan rujukan praktik industri. Nilai yang bersifat bisnis ditandai
**[USULAN — dapat diubah]**; yang teknis ditandai **[REKOMENDASI TEKNIS]**.

> Status: **draf rekomendasi** untuk direview pemilik produk. Setelah disetujui, nilai-nilai di sini
> menjadi bagian dari spesifikasi (dan dikunci lewat keputusan bertanggal).

---

## A. Bisnis & Pricing

### 1. Harga & plan awal **[USULAN]**

| Plan | Harga/siklus (30 hari) | Kuota token/siklus | Target |
|---|---:|---:|---|
| **Free** | Rp0 | 5.000 | Akuisisi & coba-coba |
| **Pro** | Rp39.000 | 150.000 | Pengguna aktif individu |
| **Business** | Rp129.000 | 1.000.000 | Power user / usaha kecil |

**Alasan:** angka bulat & mudah dipahami (praktik umum SaaS Indonesia); jarak Pro→Business (3,3×)
mendorong upsell; Free sengaja kecil agar jadi *taste*, bukan pengganti Pro.
**Rujukan:** pola freemium YNAB/Money Lover; penentuan harga berbasis nilai, bukan biaya.

### 2. Base price per 1.000 token **[USULAN]**

- `base_per_1k_idr = 300` (Rp300 / 1.000 token)
- `markup_multiplier = 1.25` (margin 25%)
- `markup_percent = 0`

**Alasan:** selaras dengan `pricing_rules` yang sudah didefinisikan di P04. Markup 25% memberi ruang
menutup biaya overhead (Redis, infra, gagal bayar). Markup **per-plan** dapat ditambah nanti (mis.
Business margin lebih tipis untuk daya tarik).
**Rujukan:** model *cost-plus markup* pada API-metered (OpenAI/Anthropic menampilkan harga per 1k token).

### 3. Free plan & trial **[USULAN]**

- **Freemium permanen** (Free selamanya, kuota kecil) — bukan trial berbatas waktu.
- **Opsional:** trial Pro 7 hari sekali per akun (`trialing`), tanpa kartu.

**Alasan:** freemium lebih baik untuk akuisisi jangka panjang; trial memicu konversi cepat. Keduanya
sudah didukung status `trialing` di model langganan.

### 4. Pengali modality **[REKOMENDASI TEKNIS]**

| Modality | Pengali | Alasan |
|---|---:|---|
| `chat` | 1.0 | Baseline teks |
| `vision` | 2.5 | Token gambar mahal (OCR struk) |
| `voice` | 3.0 | Audio lebih mahal per token |
| `embedding` | 0.5 | Embedding jauh lebih murah |
| `reasoning` | 2.0 | Model penalaran berbiaya lebih tinggi |

**Alasan:** mencerminkan biaya relatif provider; mencegah fitur mahal (voice/vision) mensubsidi silang
secara tidak adil. Nilai disimpan di `pricing_rules.modality_multipliers` (JSONB) → **dapat diubah
super admin tanpa deploy**.

### 5. Kebijakan overage **[USULAN]**

- **Tidak ada top-up token** di fase awal (YAGNI).
- Kuota habis → `402` + tawaran **upgrade**.
- Top-up dapat ditambahkan sebagai fase tersendiri jika terbukti dibutuhkan.

**Alasan:** top-up menambah kompleksitas ledger & invoice; model langganan murni lebih sederhana dan
mendorong upgrade.

---

## B. Pembayaran

### 6. Metode manual & SLA **[USULAN]**

- **Metode:** transfer bank (mis. BCA/Mandiri) + **QRIS statis**.
- **Reviewer:** super admin.
- **SLA review:** ≤ 1×24 jam kerja (target internal ≤ 4 jam pada jam kerja).
- **Bukti:** gambar/PDF, maks 5 MB; validasi tipe & ukuran.

### 7. Midtrans environment **[USULAN]**

- **Sandbox** dipakai selama P05 (pengembangan & pengujian webhook).
- **Produksi** diaktifkan di P08 (go-live), setelah checklist keamanan & chaos test lulus.
- **Pemegang akun:** pemilik produk; server key hanya di server (bukan frontend).

### 8. Refund **[USULAN]**

- Default **tidak ada refund** (downgrade berlaku akhir siklus).
- Refund hanya kasus khusus (duplikasi, kesalahan sistem) → **manual**, wajib audit.
- Lihat `../../3_Areas/billing-pembayaran/RUNBOOK_REFUND.md`.

### 9. Grace period **[USULAN]**

- **Grace 3 hari** (dapat diubah di `settings`).
- Reminder: **H-3, H-1, H-0**, lalu reminder tambahan selama grace.
- Setelah grace → `past_due` → dunning → turun ke Free (**soft-limit**, tidak hapus data).

### 10. Pajak **[USULAN]**

- PPN **ditunda** di fase awal; kolom `invoices.tax_idr` sudah ada (default 0) → siap diaktifkan.
- NPWP **opsional** (kolom tersedia untuk invoice formal nanti).
- Retensi invoice: **10 tahun** (kepatuhan pajak).

---

## C. AI Gateway

### 11. Detail 9router **[REKOMENDASI TEKNIS]**

- Format: **OpenAI-compatible** → `POST {base_url}/v1/chat/completions` (streaming SSE).
- Health: `GET {base_url}/v1/models` (atau endpoint ringan) untuk health check.
- Usage: baca `usage.prompt_tokens` + `usage.completion_tokens` (fallback: estimasi lokal).
- Embedding: `POST {base_url}/v1/embeddings` bila 9router mendukung; jika tidak, arahkan langsung ke
  Gemini `text-embedding-004`.
- Default `base_url`: `http://localhost:20128` (dapat diubah super admin).
- **Tindakan:** verifikasi endpoint & format `usage` saat P03 dimulai (uji nyata ke 9router).

### 12. Endpoint fallback langsung **[REKOMENDASI TEKNIS]**

- **Ya** — sediakan ≥1 route cadangan ke provider langsung (mis. Gemini) dengan kunci darurat terpisah.
- Aktif hanya bila seluruh endpoint 9router gagal (atau via kill switch).
- Berguna saat 9router down → layanan AI tetap hidup.

### 13. Model embedding **[REKOMENDASI TEKNIS]**

- **Tetap `text-embedding-004` (768d)** — cocok dengan `transaction_embeddings.embedding vector(768)`.
- Mengganti model non-768 = **migrasi kolom + backfill** (bukan perubahan diam-diam). Keputusan tertulis.

### 14. Kuota per kunci **[REKOMENDASI TEKNIS]**

- Pantau `ai_endpoints.last_error` & status `quota_exceeded` (429).
- Health check menandai endpoint yang kena kuota; failover ke endpoint lain.
- Tambahkan metrik "kuota endpoint" di dashboard super admin (P06).

---

## D. RBAC & Entitlement

### 15. Daftar `feature_key` & `limit_key` final **[USULAN]**

**Feature keys (bool):**

| Key | Arti |
|---|---|
| `ai_chat` | Chat AI |
| `ocr_receipt` | Scan struk (vision) |
| `voice_input` | Input suara |
| `rag_search` | Pencarian RAG |
| `reports_export` | Ekspor laporan |
| `advanced_budgeting` | Anggaran lanjutan |
| `api_access` | Akses API (Business) |
| `bank_sync` | Sinkron bank (masa depan) |

**Limit keys (integer; `-1` = unlimited):**

| Key | Arti |
|---|---|
| `ai_tokens_per_cycle` | Kuota token AI per siklus |
| `ai_requests_per_cycle` | Maks request AI per siklus |
| `max_transactions` | Maks transaksi tersimpan |
| `max_categories` | Maks kategori |
| `max_wallets` | Maks dompet/akun |
| `ocr_scans_per_cycle` | Maks scan struk per siklus |

### 16. Limit harian vs per-siklus **[REKOMENDASI TEKNIS]**

- Gunakan **per-siklus** (`ai_requests_per_cycle`) — konsisten dengan kuota token, lebih adil, dan
  menghindari kompleksitas reset harian.
- **Rate limit teknis** (mis. 10 req/menit) tetap ada sebagai perlindungan anti-abuse, terpisah dari
  entitlement.

### 17. Perilaku saat melewati limit **[USULAN]**

- **Blokir keras** untuk request berikutnya (`402`/`429`) + payload `upgrade_required`.
- Request yang **sedang berjalan** tidak dipotong (fairness).
- Tampilkan sisa kuota & tombol upgrade di UI.

---

## E. Operasional

### 18. Target skala awal **[USULAN]**

- 6 bulan pertama: **~100–1.000 user aktif**.
- Redis **single node** cukup; DB single instance + pool terkonfigurasi.
- Rencana skala: backend multi-replica (sudah didukung Redis) saat > ~500 concurrent.

### 19. Observability **[USULAN]**

- **Fase awal:** log terstruktur + metrik internal (four golden signals + metrik bisnis) + dashboard admin.
- **P08 (opsional):** Prometheus/Grafana bila diperlukan. Tidak wajib di awal (YAGNI).

### 20. Backup & DR **[REKOMENDASI TEKNIS]**

- **Full harian** + **WAL/inkremental per jam**.
- **RPO:** ≤ 1 jam (data billing); **RTO:** ≤ 2 jam.
- Snapshot tambahan **sebelum rilis besar**.
- Drill restore bulanan (lihat `../../3_Areas/operasional/RUNBOOK_BACKUP_RESTORE.md`).

### 21. Zona waktu **[KEPUTUSAN]**

- **`Asia/Jakarta` (WIB)** untuk semua perhitungan siklus & invoice; disimpan UTC di DB.

### 22. Retensi data **[USULAN]**

| Data | Retensi | Catatan |
|---|---|---|
| Invoice/pembayaran | 10 tahun | Kepatuhan pajak |
| `token_ledger` | 10 tahun (atau agregasi) | Data uang |
| `audit_logs` | 1–2 tahun | Non-repudiation |
| Data pribadi (PDP) | Selama akun aktif + kebijakan | Hak hapus → **anonimisasi** bila konflik retensi |

---

## F. UI/UX

### 23. Posisi menu super admin **[USULAN]**

- **Ganti** area `/admin` → **`/super-admin/*`** (karena `admin` lama dipromosikan jadi `super_admin`).
- Alasan: nama konsisten dengan role tunggal; menghindari kebingungan dua istilah.
- Alternatif (bila ingin minim perubahan URL): pertahankan `/admin` sebagai area super admin.

### 24. Onboarding plan **[USULAN]**

- **Halaman pricing publik** (landing) + **halaman upgrade in-app**.
- Alur: pilih plan → checkout → status pembayaran realtime (SSE).

### 25. Notifikasi **[USULAN]**

- **In-app (SSE)** dulu: reminder perpanjangan, pembayaran berhasil/gagal, plan berubah, kuota hampir habis.
- **Email** ditunda ke fase lanjut (butuh penyedia email + template). Struktur event sudah siap.

---

## G. Teknis

### 26. Redis **[REKOMENDASI TEKNIS]**

- Versi **Redis 7**, **single node** via Docker Compose untuk lokal & awal produksi.
- **Sentinel/Cluster** hanya bila skala menuntut (ditunda; YAGNI).
- Gunakan untuk: cache berversi (entitlement, DataTable), pub/sub invalidasi, rate limit.

### 27. Feature flag **[REKOMENDASI TEKNIS]**

- Cukup **tabel `feature_flags`** + cache Redis (tanpa layanan eksternal seperti Unleash).
- Alasan: skala & kebutuhan belum menuntut; menghindari dependensi tambahan (YAGNI).

### 28. Migrasi kunci lama **[REKOMENDASI TEKNIS]**

- **Impor** kunci per-user lama ke **pool terpusat** (dedupe by provider + hash), lalu tandai baris lama
  `status=inactive`/arsip (jangan dihapus).
- Menu API-key user dihapus; kunci lama tetap tersimpan untuk audit.

---

## Ringkasan keputusan yang perlu persetujuan pemilik produk

| # | Hal | Usulan | Butuh persetujuan? |
|---|---|---|---|
| 1 | Harga & kuota Free/Pro/Business | Rp0/Rp39.000/Rp129.000 | ✅ Ya (bisnis) |
| 2 | Base per 1k + markup | Rp300, 1.25× | ✅ Ya (bisnis) |
| 3 | Freemium + trial 7 hari | Ya | ✅ Ya (bisnis) |
| 5 | Top-up token | Tidak (fase awal) | ✅ Ya (bisnis) |
| 6 | SLA review manual | ≤ 1×24 jam | ⚪ Opsional |
| 7 | Midtrans prod | Di P08 | ⚪ Opsional |
| 15 | Daftar feature/limit key | Seperti tabel | ✅ Ya (produk) |
| 23 | `/admin` → `/super-admin` | Ya | ⚪ Opsional |
| 25 | Email notifikasi | Ditunda | ⚪ Opsional |

> Sisanya (teknis) direkomendasikan dan dapat dijalankan tanpa keputusan bisnis tambahan.

## Referensi

- `03-keputusan-terkunci.md` (K-01..K-30)
- `07-konfigurasi-seed-awal.md` (nilai konkret siap seed)
- `../../4_Resources/Dokumen-Spesifikasi/` (BRD/PRD)
