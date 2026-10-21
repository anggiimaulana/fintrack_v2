# FinTrack v2 — Spesifikasi Arsitektur Integrasi Bank Digital & E-Wallet

**Status:** PROPOSED / SPECIFICATION COMPLETE (2026-10-20)  
**Tanggal:** 2026-10-20  
**Pemilik:** Product & Engineering (FinTrack v2 Core Architecture)  
**Cakupan:** Integrasi mutasi bank digital & e-wallet Indonesia (BCA, Mandiri, Bank Jago, SeaBank, GoPay, DANA, ShopeePay), normalisasi mutasi, rekonsiliasi transfer internal, pelacakan saldo akun berjalan, vector embedding RAG, analitik merchant & cashflow, serta monetisasi dinamis paket Business berbasis feature entitlement zero-downtime.

---

## 1. Ringkasan Eksekutif

Pengguna FinTrack v2 membutuhkan visibilitas keuangan menyeluruh (*holistic financial picture*) di mana transaksi dari berbagai rekening bank digital dan dompet digital (e-wallet) otomatis tercatat, terbaca, divisualisasikan dalam dashboard analitik, dan dapat ditanyakan melalui asisten AI berbasis Retrieval-Augmented Generation (RAG).

Hingga saat ini, FinTrack v2 mencatat transaksi secara terpusat per pengguna (`transactions`) tanpa pemisahan entitas rekening (`account_id`), mengandalkan input manual, voice, receipt OCR, atau AI agent. Untuk menghadirkan integrasi perbankan multi-tenant yang patuh regulasi hukum Indonesia (UU No. 27 Tahun 2022 tentang Pelindungan Data Pribadi / UU PDP dan regulasi Bank Indonesia terkait Standar Nasional Open API Pembayaran / SNAP BI), FinTrack v2 mengadopsi **Strategi Hybrid Bertahap**:

1. **Fase 1 (Universal Statement Ingestion & Core Account)**: Pengguna mengunggah e-statement (PDF/CSV/XLSX) resmi dari aplikasi bank/e-wallet. Sistem memprosesnya melalui Go `StatementParser` modular, melakukan normalisasi cerdas two-stage (Regex + AI Gateway fallback), deduplikasi fingerprint SHA-256, deteksi otomatis transfer internal antar-rekening sendiri, dan meng-update saldo berjalan (`current_balance`) secara atomik.
2. **Fase 2 (Open Finance Aggregator Plugin)**: Menghubungkan widget SDK resmi agregator berlisensi (seperti Brick / Ayoconnect) untuk penarikan mutasi otomatis secara real-time bagi pengguna yang mengotorisasi koneksi bank.
3. **Monetisasi Dinamis Zero-Downtime**: Fitur ini dikunci di bawah paywall paket **Business** pada awal rilis melalui tabel konfigurasi dinamis `plan_entitlements` (`feature_key = 'bank_integration'`) dan batas numerik `plan_limits` (`max_financial_accounts`, `monthly_statement_imports`). Super Admin dapat mengaktifkan atau memindahkan fitur ini ke paket lain kapan saja tanpa restart server.
4. **Shared Token Pool Ingestion**: Pemanggilan AI Gateway untuk sanitasi deskripsi mutasi dan prediksi kategori memotong kuota token pengguna yang sudah ada (`token_quota`), menjaga model biaya tetap efisien tanpa overhead langganan ganda.

---

## 2. Indeks Dokumen Spesifikasi

Spesifikasi ini dirancang secara modular dan komprehensif ke dalam 9 berkas teknis:

| No | Dokumen | Fokus Utama |
|---|---|---|
| **01** | [01-audit-kondisi-saat-ini-dan-kelayakan.md](./01-audit-kondisi-saat-ini-dan-kelayakan.md) | Analisis codebase FinTrack v2, tinjauan legalitas SNAP BI vs scraping vs e-statement vs agregator, dan kepatuhan UU PDP. |
| **02** | [02-kebutuhan-bisnis-dan-ruang-lingkup.md](./02-kebutuhan-bisnis-dan-ruang-lingkup.md) | Persona pengguna, daftar institusi target (BCA, Mandiri, Jago, SeaBank, GoPay, DANA, ShopeePay), dan batasan fungsional. |
| **03** | [03-model-domain-dan-skema-database.md](./03-model-domain-dan-skema-database.md) | DDL PostgreSQL, tabel `financial_accounts`, `import_batches`, modifikasi kolom `transactions`, tipe enum, indeks, dan constraint. |
| **04** | [04-arsitektur-backend-dan-ingestion-pipeline.md](./04-arsitektur-backend-dan-ingestion-pipeline.md) | Desain modular Go, adapter pattern `StatementParser`, normalizer regex + AI fallback, SHA-256 deduplication, pairing transfer, dan batch rollback. |
| **05** | [05-rag-embedding-dan-analisis-keuangan.md](./05-rag-embedding-dan-analisis-keuangan.md) | Pengayaan schema vector `transaction_embeddings` pgvector, hybrid semantic search, visualisasi Sankey cashflow, dan analitik merchant. |
| **06** | [06-admin-entitlement-dan-monetisasi-paket.md](./06-admin-entitlement-dan-monetisasi-paket.md) | Integrasi dynamic entitlement `plan_entitlements`, konfigurasi kuota akun & import di `plan_limits`, dan model shared token pool. |
| **07** | [07-arsitektur-frontend-dan-desain-ux.md](./07-arsitektur-frontend-dan-desain-ux.md) | Desain UI Next.js/Tailwind/shadcn, modal drag-and-drop import, kartu rekening, banner paywall/upsell, dan layar audit batch. |
| **08** | [08-rencana-implementasi-dan-roadmap-rollout.md](./08-rencana-implementasi-dan-roadmap-rollout.md) | Breakdown tahapan migrasi database, pengerjaan backend/frontend, target sprint, dan transisi Fase 1 ke Fase 2. |
| **09** | [09-strategi-pengujian-keamanan-dan-mitigasi-risiko.md](./09-strategi-pengujian-keamanan-dan-mitigasi-risiko.md) | Test plan unit & concurrency, mitigasi kebocoran kredensial, sanitasi payload, proteksi data transaksi perbankan. |

---

## 3. Alur Kerja Arsitektur Ringkas

```
[Pengguna Upload E-Statement / Mutasi] ───► [Middleware Check Entitlement: bank_integration]
                                                               │
                                                               ▼
[Audit Batch Created: import_batches] ◄─── [Go StatementParser Factory]
                                                               │
                                                               ▼
                                                  [SHA-256 Fingerprint Check]
                                                   ├── Duplikat ──► [Skip / Log]
                                                   └── Unik ──────► [Stage 1: Regex Normalizer]
                                                                            │
                                                (Jika confidence rendah)    ▼
[Shared Token Pool Debited] ◄───────────────────────────────── [Stage 2: AI Gateway Fallback]
                                                                            │
                                                                            ▼
                                                              [Internal Transfer Detection]
                                                              (BCA Out == GoPay In?)
                                                                            │
                                                                            ▼
                                                              [Postgres Transaction Runner]
                                                              ├── Insert Transactions
                                                              ├── Update Current Balance
                                                              └── Enqueue RAG pgvector Job
```

---

## 4. Panduan Implementasi

Setiap dokumen dalam direktori ini wajib dibaca secara berurutan oleh AI Agent dan Software Engineer sebelum memulai penulisan kode pada fase implementasi. Seluruh perubahan database wajib menggunakan migrasi bernomor baru (`database/migrations/00XX_*.sql`) dan diverifikasi melalui suite test kanonik:
- Backend: `cd backend && go test ./...`
- Frontend: `cd frontend && pnpm test && pnpm build`
