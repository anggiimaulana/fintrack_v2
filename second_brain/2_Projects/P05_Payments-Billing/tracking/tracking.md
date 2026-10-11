# Tracking Implementasi — Fase P05 (Payments & Billing)

Dokumen ini memantau status perkembangan implementasi seluruh tugas dalam fase P05.
Status ditandai dengan:
- ⬜ : Belum dimulai (*Backlog*)
- 🔄 : Sedang dikerjakan (*In Progress*)
- ✅ : Selesai dan terverifikasi (*Done*)

> **Status: BERJALAN.** W1 (skema, gateway, midtrans, webhook) selesai. ✅ hanya setelah dibuktikan test nyata.

---

## Tabel Status Tugas

| Kode Tugas | Deskripsi Singkat | Sprint | Pemilik | Status | Tautan Artefak / Verifikasi |
|---|---|---|---|:---:|---|
| `TASK-P05-001` | Migrasi Skema Payments & Invoices | Sprint 1 | Backend | ✅ | 0009 diterapkan; 4 tabel (invoices, payments, settings, invoice_counters); CHECK amount>0 terbukti; replay OK |
| `TASK-P05-002` | Abstraksi Interface Gateway di `internal/payment` | Sprint 1 | Backend | ✅ | `payment/gateway.go` (interface Gateway + models, semua uang int64) |
| `TASK-P05-003` | Driver Midtrans Snap Client | Sprint 1 | Backend | ✅ | `payment/midtrans.go` (Snap create+basic auth, verify SHA512, check status); 7 test PASS |
| `TASK-P05-004` | Webhook Handler Midtrans & Validasi Signature | Sprint 1 | Backend | ✅ | `handler/payment_handler.go` webhook (MaxBytes 1MB); signature diverifikasi gateway |
| `TASK-P05-005` | Pull Verification & Idempotensi Concurrency | Sprint 1 | Backend | ✅ | `payment/webhook.go` (pull verification + row-lock idempoten); test replay 5x -> aktivasi 1x PASS |
| `TASK-P05-006` | Invoice Generator & Penomoran Deterministik | Sprint 1 | Backend | ✅ | `payment/invoice_service.go` + `payment_repo.go` NextInvoiceNumber (INV/YYYYMMDD/NNNNN, atomic upsert); test PASS |
| `TASK-P05-007` | Driver Manual Payment & Upload Bukti Transfer | Sprint 2 | Backend | ✅ | `payment/manual.go` (gateway + magic-byte proof validation 5MB) + `manual_service.go` upload; 6 test PASS |
| `TASK-P05-008` | Endpoint Review Approval Super Admin | Sprint 2 | Backend | ✅ | `payment/manual_service.go` approve/reject (row-lock, idempoten, reason wajib) + `AdminService` |
| `TASK-P05-009` | Resolver Pemilihan Gateway (Global & Override) | Sprint 2 | Backend | ✅ | `payment/registry.go` ResolveGateway (override per-plan > settings global > fallback manual) + `checkout.go` (lock gateway di baris payment); 4 test PASS |
| `TASK-P05-010` | Engine Perhitungan Prorata Upgrade | Sprint 2 | Backend | ✅ | `billing/subscription_management.go` CalculateUpgradeProration (integer floor) + invoice via `payment/checkout.go`; 5 test PASS (P04) + checkout tax test |
| `TASK-P05-011` | Manajemen Downgrade Akhir Periode | Sprint 2 | Backend | ✅ | `billing/subscription_management.go` ScheduleDowngrade/ApplyScheduledDowngrade (soft-limit, no data loss); test PASS |
| `TASK-P05-012` | Modul Pajak PPN (Tax Engine) | Sprint 2 | Backend | ✅ | `payment/tax.go` TaxCalculator (rate dari settings, floor); test PASS (0% & 11%) |
| `TASK-P05-013` | Background Worker Penagihan Dunning (H-3/H-1/H-0) | Sprint 3 | Backend | ✅ | `payment/dunning_worker.go` (H-3/H-1/H-0 buckets + dedup); test PASS |
| `TASK-P05-014` | Penegakan Grace Period & Past Due | Sprint 3 | Backend | ✅ | `payment/grace_worker.go` (within grace -> grace; beyond -> past_due soft-limit); test PASS |
| `TASK-P05-015` | Integrasi Realtime SSE Status Pembayaran | Sprint 3 | Backend | ✅ | `payment/realtime_publisher.go` (channel user_events:{uid}, event payment.settled); test PASS |
| `TASK-P05-016` | Pencatatan Audit Trail Finansial Komprehensif | Sprint 3 | Backend | ✅ | `payment/audit_interceptor.go` (PAYMENT_CREATE/WEBHOOK_RECEIVE/SETTLED/MANUAL_*/INVOICE_VOID); test PASS |
| `TASK-P05-017` | Pengujian Integrasi E2E & Stres Idempotensi | Sprint 3 | QA/Lead | ⬜ | `tests/integration/payment_test.go` |

---

## Log Riwayat Pembaruan

| Tanggal | ID Tugas | Perubahan | Oleh |
|---|---|---|---|
| 2026-10-11 | Semua | Inisialisasi daftar tugas fase P05 dengan status backlog | Tim Arsitek FinTrack v2 |
