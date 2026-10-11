# PRD — FinTrack v2: Platform Berlangganan & AI Gateway

| Field | Isi |
|---|---|
| **Nama dokumen** | Product Requirements Document (PRD) |
| **Produk** | FinTrack v2 |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Status** | Disetujui untuk perencanaan |
| **Terkait** | BRD — FinTrack v2 |
| **Konteks repo** | `D:/project/finance/fintrack_v2` |

---

## 1. Tujuan Produk

Menyediakan platform keuangan pribadi berlangganan yang: (a) memakai AI tanpa menyulitkan user,
(b) menagih secara adil berbasis pemakaian, (c) dikelola penuh oleh super admin, dan (d) dapat
beroperasi tanpa downtime dengan standar UI/UX yang konsisten.

## 2. Persona & Peran

| Peran | Deskripsi | Kebutuhan utama |
|---|---|---|
| **Super admin** | Operator platform (dulu "admin") | Kontrol penuh plan/harga/gateway, audit, kelola user & pembayaran |
| **User (tenant)** | Pengguna akhir | Catat transaksi, pakai AI, kelola langganan, transparansi kuota |
| **Guest** | Belum login | Lihat halaman pricing/landing, daftar |

## 3. Arsitektur Sistem (ringkas)

```
Browser ─▶ Next.js (proxy same-origin) ─▶ Go API (Gin) ─▶ PostgreSQL 17 (kebenaran)
                                             │  ├─▶ Redis (cache berversi + pub/sub)
                                             │  └─▶ AI Gateway ─▶ pool 9router ─▶ LLM
                                             └─▶ SSE ─▶ Browser (event realtime)
```

**Modul backend baru:** `aigateway`, `billing`, `entitlement`, `payment`, `realtime`, `audit`.
**Modul lama diadaptasi:** `ai_usecase.go` (panggil aigateway), `middleware/rbac.go`, `middleware/cache.go`.

## 4. Kebutuhan Fungsional

### 4.1 Pilar 1 — AI Gateway Terpusat

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-1.1 | Super admin dapat menambah/mengubah/menonaktifkan endpoint 9router (URL, kunci, provider, prioritas, modality) | Wajib |
| FR-1.2 | Kunci disimpan terenkripsi; hanya preview yang ditampilkan | Wajib |
| FR-1.3 | Routing per use-case (`chat`, `vision`, `voice`, `embedding`, `reasoning`) → endpoint + model + combo | Wajib |
| FR-1.4 | Prioritas & fallback: jika endpoint gagal, pindah ke berikutnya | Wajib |
| FR-1.5 | Circuit breaker: endpoint yang gagal berulang di-skip sementara (cooldown) | Wajib |
| FR-1.6 | Health check berkala; status tampil di dashboard super admin | Wajib |
| FR-1.7 | Kill switch per endpoint & per use-case | Wajib |
| FR-1.8 | Setiap panggilan AI mencatat pemakaian token (usage provider, fallback estimasi) | Wajib |
| FR-1.9 | Embedding dikunci pada model 768 dimensi (`text-embedding-004`) | Wajib |
| FR-1.10 | Menu API-key per-user dihapus; baris lama diarsipkan (tidak dihapus) | Wajib |

### 4.2 Pilar 2 — Subscription & Token

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-2.1 | Super admin CRUD plan (nama, harga, periode hari, status, urutan) | Wajib |
| FR-2.2 | Super admin mengatur entitlement fitur (centang bool) per plan | Wajib |
| FR-2.3 | Super admin mengatur limit numerik per plan | Wajib |
| FR-2.4 | Setiap user punya langganan dengan siklus `start + N hari` | Wajib |
| FR-2.5 | Kuota token per siklus, reset tiap awal siklus (tanpa rollover) | Wajib |
| FR-2.6 | Ledger token append-only (reserve/commit/release/grant/adjust) | Wajib |
| FR-2.7 | Harga token dinamis: base per 1.000 token + markup (kali & persen) + pengali modality | Wajib |
| FR-2.8 | Upgrade instan dengan prorata sisa siklus | Wajib |
| FR-2.9 | Downgrade berlaku akhir siklus (soft-limit, tanpa hapus data) | Wajib |
| FR-2.10 | Pembatalan langganan (berlaku akhir siklus) | Wajib |
| FR-2.11 | Riwayat event langganan (upgrade/downgrade/renew/cancel/expire/grace) | Wajib |

### 4.3 Pilar 3 — Pembayaran

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-3.1 | Checkout via Midtrans Snap (hosted) | Wajib |
| FR-3.2 | Pembayaran manual: unggah bukti transfer | Wajib |
| FR-3.3 | Switch payment gateway global (Midtrans/manual) realtime, tanpa deploy | Wajib |
| FR-3.4 | Override gateway per-plan | Wajib |
| FR-3.5 | Webhook Midtrans diverifikasi (signature) & idempoten | Wajib |
| FR-3.6 | Verifikasi ulang status via Status API Midtrans | Wajib |
| FR-3.7 | Invoice dasar (nomor, item, total, status, tanggal), siap PPN | Sebaiknya |
| FR-3.8 | Antrean review pembayaran manual (approve/reject oleh super admin) | Wajib |
| FR-3.9 | Reminder perpanjangan (H-3, H-1, H-0) + grace period + dunning | Wajib |
| FR-3.10 | Transaksi in-flight tetap terikat gateway asalnya saat switch | Wajib |

### 4.4 Pilar 4 — RBAC Berbasis Entitlement

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-4.1 | Role: `super_admin`, `user` | Wajib |
| FR-4.2 | Entitlement diturunkan dari plan (fitur + limit), bukan role | Wajib |
| FR-4.3 | Penegakan di backend per-request; frontend hanya cermin UX | Wajib |
| FR-4.4 | Perubahan entitlement berlaku langsung (cache berversi) tanpa login/logout | Wajib |
| FR-4.5 | Endpoint `/me/entitlements` mengembalikan hak akses efektif user | Wajib |
| FR-4.6 | Perubahan entitlement memicu event realtime ke user terkait | Wajib |

### 4.5 Pilar 5 — Standar Lintas-Halaman

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-5.1 | Komponen `<DataTable>` reusable: pagination 20, sorting, search, filter, penomoran | Wajib |
| FR-5.2 | State tabel tersinkron ke URL (`?page=&sort=&order=&q=&filter`) | Wajib |
| FR-5.3 | Caching realtime: surgical update (create/update/delete hanya fetch data terkait) | Wajib |
| FR-5.4 | TanStack Query sebagai standar server-state | Wajib |
| FR-5.5 | Chart interaktif (tooltip, legend, drill-down) | Sebaiknya |
| FR-5.6 | Semua elemen interaktif `cursor-pointer` | Sebaiknya |
| FR-5.7 | Data realtime (SSE) & zero downtime | Wajib |

### 4.6 Pilar 6 — Super Admin

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-6.1 | Dashboard metrik (user aktif, MRR, token usage, payment success, status gateway) | Wajib |
| FR-6.2 | Manajemen user (lihat, aktif/nonaktif, ubah role, lihat langganan) | Wajib |
| FR-6.3 | Manajemen plan & entitlement | Wajib |
| FR-6.4 | Manajemen subscription (lihat, extend, force downgrade, cancel) | Wajib |
| FR-6.5 | Manajemen pembayaran (review manual, riwayat, refund tercatat) | Wajib |
| FR-6.6 | Manajemen token (kuota/pemakaian per user, adjust manual + audit) | Wajib |
| FR-6.7 | Manajemen AI gateway (endpoint, routing, health, kill switch) | Wajib |
| FR-6.8 | Audit log viewer | Wajib |
| FR-6.9 | Pengaturan (switch gateway, grace period, pricing default, feature flag) | Wajib |

## 5. Alur Utama (User Flows)

### 5.1 Registrasi → Free → Upgrade

1. Guest daftar → akun `user` + langganan `free` otomatis.
2. User memakai fitur sesuai entitlement free.
3. User klik "Upgrade" → halaman pricing → pilih plan.
4. Sistem membuat invoice/payment → Midtrans Snap (atau instruksi manual).
5. Pembayaran sukses (webhook) → langganan aktif → kuota token baru → SSE ke user.

### 5.2 Pemakaian AI (metering)

1. User mengirim pesan/foto/voice.
2. Backend: cek entitlement → **reserve** token → panggil AI Gateway → **commit** (usage) / **release**.
3. Jika kuota habis → `402` + tawaran upgrade.
4. Pemakaian tercatat di ledger & ditampilkan ke user.

### 5.3 Upgrade (prorata)

1. User pilih plan lebih tinggi → sistem hitung selisih prorata sisa siklus.
2. User bayar → langganan langsung naik, kuota disesuaikan proporsional.
3. Event `upgrade` dicatat.

### 5.4 Downgrade / Cancel

1. User pilih plan lebih rendah / cancel.
2. Sistem set `cancel_at_period_end` → tetap akses sampai akhir siklus.
3. Akhir siklus → plan baru aktif (soft-limit bila pemakaian melebihi kuota baru).
4. Event `downgrade`/`cancel` dicatat.

### 5.5 Renewal (hybrid)

1. Mendekati akhir siklus → reminder H-3/H-1/H-0.
2. Metode mendukung auto → charge otomatis; jika tidak → user bayar manual.
3. Sukses → siklus baru + kuota reset. Gagal → grace → dunning → turun ke free.

### 5.6 Super admin mengubah entitlement

1. Super admin buka plan → ubah centang/limit → simpan.
2. Sistem bump `entitlement_version` → cache lama gugur.
3. Request berikutnya user memakai aturan baru (tanpa login/logout).
4. Event realtime dikirim ke user terkait.

### 5.7 Failover gateway AI

1. Panggilan ke endpoint aktif gagal (5xx/timeout/429).
2. Circuit breaker menandai endpoint; sistem pindah ke endpoint berikutnya.
3. Jika sukses → transparan ke user. Jika semua gagal → `503` dengan pesan ramah.
4. Health & failover dicatat + tampil di dashboard.

### 5.8 Switch payment gateway

1. Super admin ubah `settings.payment_gateway` (Midtrans ↔ manual) → tersimpan.
2. Checkout **baru** memakai gateway baru.
3. Transaksi pending lama tetap diproses gateway asalnya (webhook lama tetap valid).

## 6. Kebutuhan Non-Fungsional

| ID | Kebutuhan | Target |
|---|---|---|
| NFR-1 | Performa API | p95 < 300ms (non-AI); chat p95 < 3s |
| NFR-2 | Zero downtime deploy | 0 menit (rolling + expand/contract + feature flag) |
| NFR-3 | Konsistensi cache lintas instance | Redis + pub/sub |
| NFR-4 | Keamanan pembayaran | Webhook signature + idempoten; PCI-DSS SAQ-A |
| NFR-5 | Keamanan kunci | Enkripsi at-rest, rotasi, audit, least privilege |
| NFR-6 | Akurasi billing | Ledger idempoten, tanpa double-charge |
| NFR-7 | Observability | Metrik token, error gateway, payment success, failover |
| NFR-8 | Aksesibilitas | a11y dasar (label, aria, kontras, focus-visible) |
| NFR-9 | Kepatuhan | UU PDP No. 27/2022 |
| NFR-10 | Skalabilitas | Siap multi-instance backend |

## 7. Model Data (ringkas)

**Billing:** `plans`, `plan_entitlements`, `plan_limits`, `subscriptions`, `subscription_events`,
`token_ledger`, `token_periods`, `pricing_rules`.
**Pembayaran:** `payments`, `invoices`.
**AI Gateway:** `ai_endpoints`, `ai_routes`.
**Platform:** `feature_flags`, `audit_logs`, `settings`.
**Perubahan lama:** `users.role` (+`super_admin`), `gemini_api_keys` (arsip), `transaction_embeddings`
(tetap `vector(768)`).

## 8. Kontrak API (ringkas)

| Method | Path | Fungsi |
|---|---|---|
| GET | `/api/v1/me/entitlements` | Hak akses efektif user |
| GET/POST/PUT/DELETE | `/api/v1/plans` | Manajemen plan (super admin) |
| PUT | `/api/v1/plans/:id/entitlements` | Set fitur & limit |
| GET/POST | `/api/v1/subscriptions` | Langganan user / manajemen |
| POST | `/api/v1/subscriptions/:id/upgrade` | Upgrade prorata |
| POST | `/api/v1/subscriptions/:id/downgrade` | Downgrade akhir siklus |
| POST | `/api/v1/subscriptions/:id/cancel` | Batalkan |
| GET | `/api/v1/tokens/usage` | Pemakaian token user |
| POST | `/api/v1/payments/checkout` | Buat checkout (Midtrans/manual) |
| POST | `/api/v1/payments/manual/proof` | Unggah bukti transfer |
| POST | `/api/v1/payments/webhook/midtrans` | Webhook Midtrans |
| POST | `/api/v1/admin/payments/:id/approve` | Approve pembayaran manual |
| GET/POST/PUT | `/api/v1/admin/ai-endpoints` | Manajemen endpoint AI |
| PUT | `/api/v1/admin/ai-routes` | Routing AI |
| GET | `/api/v1/admin/audit-logs` | Audit log |
| GET/PUT | `/api/v1/admin/settings` | Pengaturan platform |

**Envelope respons:** `{success, message, data, meta{total,page,per_page}}` (sudah ada).

## 9. Standar UI/UX

- **DataTable** di semua tabel: pagination 20, sorting, searching, filtering, penomoran, URL state.
- **Caching realtime**: surgical update TanStack Query; SSE untuk perubahan eksternal.
- **Chart** interaktif (tooltip, legend, drill-down).
- **Cursor pointer** di semua elemen interaktif.
- **Zero downtime**: tidak ada reload/refresh manual untuk perubahan data.

## 10. Kriteria Penerimaan (contoh, per pilar)

- **AI Gateway**: mematikan 1 endpoint → trafik otomatis pindah, user tidak melihat error.
- **Token**: 20 request paralel pada kuota 5 → maksimum 5 lolos, tidak ada over-spend.
- **Pembayaran**: replay webhook 10× → 1 baris pembayaran; langganan aktif sekali.
- **Upgrade**: upgrade di tengah siklus → harga prorata benar (pembulatan ke bawah).
- **Downgrade**: data user tetap ada; hanya fitur/kuota dibatasi.
- **Entitlement**: super admin ubah centang → user langsung merasakan tanpa login/logout.
- **DataTable**: semua tabel punya `?page=` dan mempertahankan state saat reload.
- **Zero downtime**: deploy saat ada request berjalan → tidak ada request gagal.

## 11. Ketergantungan & Asumsi

- Redis ditambahkan ke stack.
- 9router tersedia (OpenAI-compatible).
- Akun Midtrans (sandbox → produksi).
- `ENCRYPTION_KEY` lama dipertahankan.

## 12. Risiko Produk & Mitigasi

Lihat BRD §9. Tambahan: kompleksitas dunning & review manual → otomatiskan reminder, sediakan SLA jelas.

## 13. Rencana Rilis

| Fase | Rilis |
|---|---|
| P02–P04 | Internal (di balik feature flag) |
| P05 | Beta terbatas (sandbox Midtrans) |
| P06–P07 | Pilot (produksi terbatas) |
| P08 | Go-live umum |

## 14. Lampiran — Peta FR ke Fase

| Fase | FR |
|---|---|
| P02 | FR-5.1..5.4, FR-5.7 (fondasi) |
| P03 | FR-1.1..FR-1.10 |
| P04 | FR-2.1..FR-2.11, FR-4.1..FR-4.5 |
| P05 | FR-3.1..FR-3.10 |
| P06 | FR-6.1..FR-6.9 |
| P07 | FR-4.6, FR-5.1..FR-5.7 (rollout) |
| P08 | NFR semua |
