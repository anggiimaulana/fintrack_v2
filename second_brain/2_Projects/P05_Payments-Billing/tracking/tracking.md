# Tracking Implementasi — Fase P05 (Payments & Billing)

Dokumen ini memantau status perkembangan implementasi seluruh tugas dalam fase P05.
Status ditandai dengan:
- ⬜ : Belum dimulai (*Backlog*)
- 🔄 : Sedang dikerjakan (*In Progress*)
- ✅ : Selesai dan terverifikasi (*Done*)

> **Status saat ini: BELUM DIMULAI.** Seluruh task masih ⬜ — belum ada implementasi kode.

---

## Tabel Status Tugas

| Kode Tugas | Deskripsi Singkat | Sprint | Pemilik | Status | Tautan Artefak / Verifikasi |
|---|---|---|---|:---:|---|
| `TASK-P05-001` | Migrasi Skema Payments & Invoices | Sprint 1 | Backend | ⬜ | `database/migrations/0007_create_payments_and_invoices.sql` |
| `TASK-P05-002` | Abstraksi Interface Gateway di `internal/payment` | Sprint 1 | Backend | ⬜ | `internal/payment/gateway.go` |
| `TASK-P05-003` | Driver Midtrans Snap Client | Sprint 1 | Backend | ⬜ | `internal/payment/midtrans.go` |
| `TASK-P05-004` | Webhook Handler Midtrans & Validasi Signature | Sprint 1 | Backend | ⬜ | `internal/payment/handler_webhook.go` |
| `TASK-P05-005` | Pull Verification & Idempotensi Concurrency | Sprint 1 | Backend | ⬜ | `internal/payment/service.go` |
| `TASK-P05-006` | Invoice Generator & Penomoran Deterministik | Sprint 1 | Backend | ⬜ | `internal/payment/invoice_service.go` |
| `TASK-P05-007` | Driver Manual Payment & Upload Bukti Transfer | Sprint 2 | Backend | ⬜ | `internal/payment/manual.go` |
| `TASK-P05-008` | Endpoint Review Approval Super Admin | Sprint 2 | Backend | ⬜ | `internal/payment/handler_admin.go` |
| `TASK-P05-009` | Resolver Pemilihan Gateway (Global & Override) | Sprint 2 | Backend | ⬜ | `internal/payment/resolver.go` |
| `TASK-P05-010` | Engine Perhitungan Prorata Upgrade | Sprint 2 | Backend | ⬜ | `internal/billing/proration.go` |
| `TASK-P05-011` | Manajemen Downgrade Akhir Periode | Sprint 2 | Backend | ⬜ | `internal/billing/downgrade_service.go` |
| `TASK-P05-012` | Modul Pajak PPN (Tax Engine) | Sprint 2 | Backend | ⬜ | `internal/payment/tax.go` |
| `TASK-P05-013` | Background Worker Penagihan Dunning (H-3/H-1/H-0) | Sprint 3 | Backend | ⬜ | `internal/payment/dunning_worker.go` |
| `TASK-P05-014` | Penegakan Grace Period & Past Due | Sprint 3 | Backend | ⬜ | `internal/payment/grace_worker.go` |
| `TASK-P05-015` | Integrasi Realtime SSE Status Pembayaran | Sprint 3 | Backend | ⬜ | `internal/payment/realtime_publisher.go` |
| `TASK-P05-016` | Pencatatan Audit Trail Finansial Komprehensif | Sprint 3 | Backend | ⬜ | `internal/payment/audit_interceptor.go` |
| `TASK-P05-017` | Pengujian Integrasi E2E & Stres Idempotensi | Sprint 3 | QA/Lead | ⬜ | `tests/integration/payment_test.go` |

---

## Log Riwayat Pembaruan

| Tanggal | ID Tugas | Perubahan | Oleh |
|---|---|---|---|
| 2026-10-11 | Semua | Inisialisasi daftar tugas fase P05 dengan status backlog | Tim Arsitek FinTrack v2 |
