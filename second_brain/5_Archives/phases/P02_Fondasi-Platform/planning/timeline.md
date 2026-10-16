# Timeline & Dependensi — P02 Fondasi Platform

Dokumen ini memetakan urutan eksekusi, jalur kritis (critical path), dan dependensi antar-task dalam Fase P02 serta keterkaitannya dengan fase lanjutan (P03 s/d P08).

---

## 1. Diagram Alur Dependensi Task (DAG)

```mermaid
graph TD
    TASK-P02-001[TASK-001: Redis Docker] --> TASK-P02-002[TASK-002: Redis Go Client]
    TASK-P02-002 --> TASK-P02-003[TASK-003: Versioned Cache Engine]
    TASK-P02-002 --> TASK-P02-004[TASK-004: Redis Pub/Sub]
    
    TASK-P02-005[TASK-005: Migrasi audit_logs] --> TASK-P02-006[TASK-006: Modul internal/audit]
    
    TASK-P02-007[TASK-007: Migrasi feature_flags] --> TASK-P02-008[TASK-008: Modul featureflag Engine]
    TASK-P02-002 --> TASK-P02-008
    
    TASK-P02-009[TASK-009: Go Pagination Parser] --> TASK-P02-010[TASK-010: Next.js URL State Hook]
    TASK-P02-010 --> TASK-P02-011[TASK-011: Generic DataTable UI]
    TASK-P02-011 --> TASK-P02-012[TASK-012: Toolbar & Pagination UI]
    
    TASK-P02-002 --> TASK-P02-013[TASK-013: Observability Metrics]
    
    TASK-P02-003 --> TASK-P02-014[TASK-014: E2E Integration Test]
    TASK-P02-006 --> TASK-P02-014
    TASK-P02-008 --> TASK-P02-014
    TASK-P02-012 --> TASK-P02-014
    TASK-P02-013 --> TASK-P02-014
```

---

## 2. Jalur Kritis (Critical Path)
Jalur kritis pengerjaan Fase P02:
1. `TASK-P02-001` (Redis Docker) -> `TASK-P02-002` (Redis Client Go) -> `TASK-P02-003` (Versioned Cache) -> `TASK-P02-008` (Feature Flag Engine) -> `TASK-P02-014` (Verifikasi Integrasi).
2. `TASK-P02-005` (Migrasi SQL Audit) -> `TASK-P02-006` (Worker Audit Go) -> `TASK-P02-014`.

Kegagalan atau keterlambatan pada penyediaan Redis client atau migrasi audit logs akan menahan modul audit dan feature flag, yang selanjutnya memblokir fase P03 dan P04.

---

## 3. Matriks Ketergantungan Lintas Fase

| Komponen P02 | Digunakan Oleh Fase | Bentuk Ketergantungan |
|---|---|---|
| **Redis Shared Pool** | P03 (AI Gateway), P04 (Billing), P06 (Super Admin) | Rate limiter AI Gateway, session cache, realtime event broadcasting. |
| **Versioned Cache** | P04 (Subscription Engine), P06 (Super Admin) | Penegakan Realtime RBAC tanpa JWT refresh (K-14), cache profile user instan terinvalidasi saat subscription berubah. |
| **Audit Log Append-Only** | P03, P04, P05, P06, P07 | Pencatatan transaksi mutasi token AI, perubahan paket subscription, penerimaan pembayaran webhook, dan perubahan konfigurasi super admin. |
| **Feature Flags** | P03, P04, P05, P08 | Canary rollout endpoint AI Gateway v2, peluncuran gateway pembayaran Midtrans/Xendit baru, kill switch AI. |
| **Generic DataTable + URL State** | P06 (Super Admin), P07 (Audit UI) | Tampilan tabel manajemen user, daftar transaksi, riwayat audit logs, log penggunaan token. |
| **Observability Baseline** | P03, P04, P05, P08 | Metrik konsumsi token AI, error rate provider AI, durasi request webhook payment, saturation server. |

---

## 4. Jadwal Rilis & Milestones

- **Milestone M2.1 (Hari 3)**: Infrastruktur Redis berjalan, driver pool Go stabil, dan engine cache berversi lulus pengujian konkurensi.
- **Milestone M2.2 (Hari 5)**: Modul audit log aktif di PostgreSQL dengan trigger append-only dan asynchronous worker Go lulus uji benchmark.
- **Milestone M2.3 (Hari 7)**: Feature flag evaluator beroperasi dengan cache Redis dan fallback fail-closed; kontrak pagination backend siap.
- **Milestone M2.4 (Hari 9)**: Komponen `<DataTable>` generik frontend terintegrasi sempurna dengan search params URL di Next.js.
- **Milestone M2.5 (Hari 10)**: Seluruh metrik Four Golden Signals terekspos; test integrasi E2E P02 lulus; siap handover ke P03 (AI Gateway).
