# Implementation Plan (Master) — FinTrack v2 Platform Berlangganan & AI Gateway

| Field | Isi |
|---|---|
| **Dokumen** | Rencana Implementasi Tingkat Program |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Status** | Disetujui untuk eksekusi |
| **Cakupan** | Fase P02–P08 |
| **Referensi** | `../4_Resources/Dokumen-Spesifikasi/`, `../0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/` |

> **Penting:** Dokumen ini adalah rencana **eksekusi**. Sebelum fase dijalankan, status seluruh task
> adalah ⬜ (belum dimulai). Hasil uji **tidak boleh** ditulis sebelum pengujian nyata dijalankan.

---

## 1. Prinsip Eksekusi

1. **Fase sequential** — fase berikut tidak dimulai sebelum DoD fase sebelumnya terpenuhi.
2. **Zero downtime** — setiap perubahan migrasi bersifat **aditif** (expand → migrate → contract);
   fitur baru di balik **feature flag** (default OFF).
3. **Backend otoritatif** — keputusan hak akses & uang selalu di backend.
4. **Uang = integer rupiah**; token = integer; ledger **append-only**; mutasi **idempoten**.
5. **Test membuktikan klaim** — setiap klaim arsitektur punya test (ledger, failover, idempotensi, entitlement).
6. **Tidak ada data user yang hilang** karena alasan billing (soft-limit).
7. **Commit atomik** per task/sub-task; catat di `1_Digest/Tugas-Rutin/log_aktivitas.md`.

## 2. Aturan Migrasi (WAJIB)

- Migrasi bernomor: `database/migrations/000N_*.sql` (mulai `0005`), **replayable**.
- Perubahan **aditif** dulu: `CREATE TABLE IF NOT EXISTS`, kolom nullable/DEFAULT, `CREATE INDEX CONCURRENTLY`.
- **Dilarang** dalam satu langkah: `NOT NULL` tanpa default pada tabel besar, `ALTER TYPE` langsung, `DROP` yang dipakai kode berjalan.
- Setelah schema berubah → regenerate `database/dump/`.

## 3. Alokasi Nomor Migrasi (rencana, dapat disesuaikan)

| Migrasi | Fase | Isi |
|---|---|---|
| `0005_create_audit_logs.sql` | P02 | `audit_logs` append-only + trigger blok UPDATE/DELETE |
| `0006_create_feature_flags.sql` | P02 | `feature_flags` |
| `0007_create_ai_gateway.sql` | P03 | `ai_endpoints`, `ai_routes` |
| `0008_create_subscription_token_core.sql` | P04 | `plans`, `plan_entitlements`, `plan_limits`, `subscriptions`, `subscription_events`, `token_periods`, `token_ledger`, `pricing_rules` |
| `0009_create_payments_invoices.sql` | P05 | `payments`, `invoices` |
| `0010_*` | P06 | `settings` (+ kolom pendukung dashboard) |
| `0011_*` | P07 | (bila perlu) index/kolom untuk rollout |
| `0012_*` | P08 | (bila perlu) index performa/retensi |

## 4. Ringkasan Fase

| Fase | Nama | Fokus | Bergantung | Estimasi |
|---|---|---|---|---|
| **P02** | Fondasi Platform | Redis, cache berversi, audit log, feature flag, DataTable, observability | — | 2 sprint |
| **P03** | AI Gateway Terpusat | Pool 9router, routing, circuit breaker, kill switch, metering | P02 | 2 sprint |
| **P04** | Subscription & Token Core | Plan, entitlement, subscription, token ledger, enforcement | P03 | 3 sprint |
| **P05** | Payments & Billing | Midtrans + manual, webhook, invoice, prorata, dunning | P04 | 3 sprint |
| **P06** | Super Admin Dashboard | Manajemen & metrik | P04, P05 | 3 sprint |
| **P07** | Rollout RBAC, Realtime & Standar | Entitlement + SSE + DataTable ke semua halaman | P02–P06 | 2 sprint |
| **P08** | Hardening & Go-Live | Load/chaos test, security, compliance, runbook | semua | 2 sprint |

## 5. Definition of Done (per task)

- [ ] Kode & test ditulis; `cd backend && go vet ./... && go test ./... && go build ./...` hijau.
- [ ] `cd frontend && pnpm build && pnpm lint && pnpm test` hijau.
- [ ] Migrasi (bila ada) replayable; dump di-regenerate.
- [ ] Fitur baru di balik feature flag (default OFF).
- [ ] Audit log mencatat mutasi sensitif.
- [ ] `tracking/tracking.md` diperbarui; `log_aktivitas.md` dicatat.

## 6. Definition of Done (per fase)

- [ ] Semua task fase ✅ & terverifikasi.
- [ ] Kriteria Sukses fase terpenuhi (lihat README fase).
- [ ] Tidak ada regresi fase sebelumnya.
- [ ] Zero downtime terverifikasi.
- [ ] Dokumentasi fase akurat & diperbarui.

## 7. Gate Kualitas (Quality Gates)

| Gate | Kriteria | Fase |
|---|---|---|
| G1 | Redis & cache berversi terbukti lintas-instance | P02 |
| G2 | Audit log append-only (trigger menolak UPDATE/DELETE) | P02 |
| G3 | DataTable + URL state jalan & dapat di-bookmark | P02 |
| G4 | Failover gateway: matikan 1 endpoint → tanpa error user | P03 |
| G5 | Ledger: 20 request paralel kuota 5 → ≤ 5 lolos | P04 |
| G6 | Webhook replay 10× → 1 baris | P05 |
| G7 | Entitlement berubah → berlaku tanpa login/logout | P04/P07 |
| G8 | Deploy saat beban → 0 request gagal | P07/P08 |
| G9 | Kepatuhan PCI-DSS SAQ-A & UU PDP terverifikasi | P08 |

## 8. Urutan Eksekusi & Dependensi

```
P02 ──▶ P03 ──▶ P04 ──▶ P05 ──▶ P06
 │                              │
 └──────────────▶ P07 ◀─────────┘
                  │
                  ▼
                 P08
```

## 9. Risiko Program & Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Redis jadi SPOF | Layanan lambat | Fallback bypass cache + `maxmemory-policy` + healthcheck |
| Concurrency ledger bocor | Keuangan salah | Row lock + idempotency + test `-race` |
| 9router down | Fitur AI mati | Pool + failover + circuit breaker + fallback langsung |
| Webhook dipalsukan | Pembayaran palsu | Signature + Status API + idempoten + audit |
| Migrasi memutus sesi | Downtime | Expand/contract + feature flag + health gate |
| Scope melebar | Terlambat | YAGNI; gate per fase; review pemilik produk |

## 10. Praktik Kerja

- Branch: `feat/<topik>`; `main` selalu hijau.
- Commit: `feat(scope): ...`, `fix(scope): ...`, `docs(scope): ...`.
- Setiap perubahan schema → migrasi bernomor → regenerate dump.
- Dokumentasi hidup diperbarui di `3_Areas/`.

## 11. Rencana per Fase (dokumen detail)

Semua fase memiliki rencana detail:

| Fase | Dokumen |
|---|---|
| P02 | `P02_Fondasi-Platform/planning/implementation-plan.md` ← **mulai dari sini** |
| P03 | `P03_AI-Gateway-Terpusat/planning/implementation-plan.md` |
| P04 | `P04_Subscription-Token-Core/planning/implementation-plan.md` |
| P05 | `P05_Payments-Billing/planning/implementation-plan.md` |
| P06 | `P06_Super-Admin-Dashboard/planning/implementation-plan.md` |
| P07 | `P07_Rollout-RBAC-Realtime/planning/implementation-plan.md` |
| P08 | `P08_Hardening-GoLive/planning/implementation-plan.md` |

Setiap rencana fase memuat: tujuan & DoD, urutan wave, detail task (target file, langkah, verifikasi,
bukti), matriks bukti, perintah verifikasi, rollback, dan handoff ke fase berikutnya.

## Referensi

- BRD: `../4_Resources/Dokumen-Spesifikasi/BRD/BRD - FinTrack v2.md`
- PRD: `../4_Resources/Dokumen-Spesifikasi/PRD/PRD - FinTrack v2.md`
- Keputusan: `../0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/03-keputusan-terkunci.md`
- Seed awal: `../0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/07-konfigurasi-seed-awal.md`
