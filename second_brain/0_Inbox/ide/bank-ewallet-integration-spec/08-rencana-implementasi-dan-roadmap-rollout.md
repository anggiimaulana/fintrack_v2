# 08 — Rencana Implementasi & Roadmap Rollout

**Topik:** Tahapan Eksekusi Teknis, Pembagian Wave, Definition of Done, & Transisi Fase 1 ke Fase 2  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Roadmap Peluncuran Bertahap

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    ROADMAP INTEGRASI BANK & E-WALLET                        │
├──────────────────────────────────────┬──────────────────────────────────────┤
│ FASE 1: UNIVERSAL STATEMENT INGEST   │ FASE 2: OPEN FINANCE AGGREGATOR      │
│ (Rilis Target — Milik Sendiri & Zero │ (Rilis Lanjutan — Kemitraan Berizin) │
│ Biaya Vendor)                        │                                      │
├──────────────────────────────────────┼──────────────────────────────────────┤
│ - Akun Finansial & Saldo Berjalan    │ - Integrasi SDK Brick / Ayoconnect   │
│ - Adapter Parser BCA, Mandiri, Jago, │ - Sinkronisasi mutasi otomatis via   │
│   SeaBank, GoPay, DANA, ShopeePay    │   webhook berkala                    │
│ - Two-Stage Regex & AI Normalizer    │ - Multi-factor consent management    │
│ - Deduplikasi Fingerprint SHA-256    │ - Fitur toggle `bank_auto_sync`      │
│ - Rekonsiliasi Transfer Internal     │ - SLA & monitoring koneksi bank B2B  │
│ - Enriched RAG pgvector + Sankey UI  │                                      │
│ - Paywall Paket Business Dinamis     │                                      │
└──────────────────────────────────────┴──────────────────────────────────────┘
```

---

## 2. Rincian Pengerjaan Fase 1 (Siklus 7 Wave)

### Wave 1: Database Migration & Core Domain
- **Target File:**
  - `database/migrations/0044_bank_ewallet_accounts.sql`
  - `backend/internal/domain/financial_account.go`
  - `backend/internal/domain/transaction.go` (tambah field relasi akun)
- **Tugas:**
  1. Buat tipe enum `financial_account_type` (`bank`, `ewallet`, `cash`, `investment`).
  2. Tambahkan tipe `transfer` pada enum `transaction_type`.
  3. Buat tabel `financial_accounts` dan `import_batches`.
  4. Tambahkan kolom `account_id`, `import_batch_id`, `fingerprint_hash`, `clean_merchant`, `transfer_pair_id` pada tabel `transactions`.
  5. Buat indeks unik `uq_tx_fingerprint` dan indeks relasi akun.
  6. Daftarkan migrasi pada `migrate.sh` dan jalankan verifikasi skema.
- **DoD:** Migrasi sukses dijalankan di Postgres lokal, database dump diperbarui, query `SELECT` dan constraint berfungsi.

### Wave 2: Parser Factory & Native Bank Adapters
- **Target File:**
  - `backend/internal/ingestion/parser.go`
  - `backend/internal/ingestion/adapters/bca_parser.go`
  - `backend/internal/ingestion/adapters/mandiri_parser.go`
  - `backend/internal/ingestion/adapters/jago_parser.go`
  - `backend/internal/ingestion/adapters/seabank_parser.go`
  - `backend/internal/ingestion/adapters/gopay_parser.go`
  - `backend/internal/ingestion/adapters/dana_parser.go`
  - `backend/internal/ingestion/adapters/shopeepay_parser.go`
  - `backend/internal/ingestion/adapters/generic_csv_parser.go`
- **Tugas:**
  1. Bangun interface `StatementParser` dan fungsi registrasi `ParserRegistry`.
  2. Implementasikan parser untuk masing-masing dari 7 institusi target + 1 generic CSV fallback.
  3. Siapkan fixture file dummy CSV dan PDF untuk setiap format di `backend/internal/ingestion/testdata/`.
  4. Tulis unit test parser per bank dengan coverage $> 90\%$.
- **DoD:** Seluruh parser berhasil mengekstrak tanggal, nominal, jenis debit/kredit, dan deskripsi mentah dari file fixture.

### Wave 3: Ingestion Pipeline, Deduplikasi & Smart Transfer Pairing
- **Target File:**
  - `backend/internal/ingestion/normalizer.go`
  - `backend/internal/ingestion/dedup.go`
  - `backend/internal/ingestion/pairing.go`
  - `backend/internal/ingestion/service.go`
- **Tugas:**
  1. Bangun Stage 1 Regex Normalizer (membersihkan noise BI-FAST, QRIS, WSID, dll).
  2. Bangun Stage 2 AI Fallback dengan pemotongan shared token pool via `TokenMeter`.
  3. Buat engine deduplikasi SHA-256 fingerprint (`GenerateFingerprint`).
  4. Implementasikan algoritma deteksi transfer internal ($|t_A - t_B| \le 36\text{ jam}$, nominal sama).
  5. Bangun service atomik yang mengunci baris akun (`FOR UPDATE`) dan memperbarui `current_balance`.
  6. Sediakan endpoint API `POST /api/v1/accounts/:id/import-statement` dan endpoint rollback `POST /api/v1/accounts/import-batches/:id/rollback`.
- **DoD:** File mutasi berulang terbukti mengabaikan duplikat, transfer antar-rekening otomatis ter-pair, saldo akun sinkron.

### Wave 4: Dynamic Entitlement & Super Admin Management
- **Target File:**
  - `backend/internal/delivery/http/middleware/entitlement.go`
  - `backend/internal/admin/plans.go`
  - `frontend/src/app/(admin)/admin/plans/`
- **Tugas:**
  1. Semai konfigurasi awal: `bank_integration` aktif hanya di paket Business.
  2. Semai limit: `max_financial_accounts` dan `monthly_statement_imports`.
  3. Pasang middleware `RequireFeature("bank_integration")` pada seluruh handler rekening dan impor mutasi.
  4. Perbarui antarmuka Super Admin agar switch `bank_integration` dan batas akun dapat diedit interaktif.
- **DoD:** User Free dan Pro ditolak dengan status 403 saat memanggil API akun; perubahan switch di dashboard admin seketika mengubah izin tanpa restart.

### Wave 5: RAG Vector Enrichment & Hybrid Search
- **Target File:**
  - `backend/internal/infrastructure/embedding/rag_service.go`
  - `backend/internal/repository/postgres/embedding_repo.go`
- **Tugas:**
  1. Perbarui format template teks embedding agar menyertakan institusi, nama akun, tipe transfer, dan merchant bersih.
  2. Simpan metadata JSON terstruktur (`account_id`, `institution_code`, `amount_base`, `date`).
  3. Tambahkan hybrid retrieval query yang memadukan filter SQL metadata dengan cosine similarity pgvector.
  4. Uji akurasi asisten AI dalam menjawab perbandingan pengeluaran antar-bank/e-wallet.
- **DoD:** Query AI *"Berapa pengeluaran di GoPay bulan ini?"* menjawab dengan angka yang persis sesuai database.

### Wave 6: Frontend UI, Importer Wizard & Visualisasi Data
- **Target File:**
  - `frontend/src/app/(dashboard)/accounts/`
  - `frontend/src/components/accounts/`
  - `frontend/src/components/charts/`
- **Tugas:**
  1. Buat halaman `/accounts` dengan Hero Card Net Worth dan Grid Akun.
  2. Buat modal multi-step upload mutasi (Drag & Drop, Preview baris terdeteksi, Konfirmasi).
  3. Buat komponen diagram arus kas Sankey (`sankey-flow-chart.tsx`) dan bar chart Top Merchants.
  4. Buat halaman audit batch `/accounts/batches` dengan tombol batalkan impor (rollback).
  5. Buat view paywall promosi paket Business untuk user non-Business.
- **DoD:** Alur upload mutasi berjalan mulus dari browser, responsif di mobile dan desktop, dark mode sinkron, visualisasi interaktif.

### Wave 7: QA, Concurrency Stress Test & End-to-End Verification
- **Target:**
  1. Menjalankan test concurrency: 5 upload mutasi bersamaan pada akun yang sama tidak merusak saldo berjalan.
  2. Menjalankan verifikasi roll-back: 100 transaksi dari satu batch terhapus dan saldo kembali normal.
  3. Pengujian regresi penuh: `cd backend && go test ./...` dan `cd frontend && pnpm test && pnpm build`.
- **DoD:** Seluruh suite test lulus hijau tanpa kegagalan.

---

## 3. Transisi Menuju Fase 2 (Open Finance Aggregator Plugin)

Setelah Fase 1 stabil di production dan basis pengguna Business tumbuh:
1. **Pendaftaran Kemitraan:** Menandatangani perjanjian kerja sama B2B dengan agregator berlisensi (Brick atau Ayoconnect).
2. **Implementasi Widget:** Memasang SDK widget agregator di frontend `/accounts/connect-bank`.
3. **Webhook Receiver:** Membangun `AggregatorWebhookHandler` di backend untuk menerima event mutasi masuk secara otomatis setiap subuh/real-time.
4. **Zero Refactoring Core:** Karena Ingestion Engine di Fase 1 sudah memisahkan parser dari pipeline core, data dari agregator hanya perlu diubah menjadi struct `RawStatementRecord`, lalu dialirkan ke pipeline yang sama persis (Deduplikasi, Normalisasi, Pairing, Ledger, RAG).
