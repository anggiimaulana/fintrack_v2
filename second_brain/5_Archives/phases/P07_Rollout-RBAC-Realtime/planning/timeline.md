# Timeline & Analisis Dependensi: Fase P07

## Jadwal Eksekusi 10 Hari Kerja (Sprint Timeline)

```
Hari   | Task ID       | Deskripsi Task                                      | PIC / Tim
-------+---------------+-----------------------------------------------------+------------------
H-01   | TASK-P07-001  | Middleware Otorisasi Entitlement & Limit (Go)        | Backend Dev 1
H-01   | TASK-P07-003  | Migrasi globalCache In-Memory ke Redis Berversi     | Backend Dev 2
H-01   | TASK-P07-011  | Komponen Universal <DataTable> TanStack Table v8    | Frontend Dev 1
-------+---------------+-----------------------------------------------------+------------------
H-02   | TASK-P07-002  | Audit & Pasang Entitlement 28 Endpoint Legacy       | Backend Dev 1 & 2
H-02   | TASK-P07-004  | Hub SSE `internal/realtime/` Broker Koneksi         | Backend Dev 3
H-02   | TASK-P07-011  | Finalisasi Toolbar, Filter & Pagination DataTable   | Frontend Dev 1
-------+---------------+-----------------------------------------------------+------------------
H-03   | TASK-P07-005  | Integrasi Redis Pub/Sub Distribusi SSE Multi-Pod    | Backend Dev 3
H-03   | TASK-P07-012  | Refactor Halaman Transaksi & Anggaran ke DataTable  | Frontend Dev 1 & 2
H-03   | TASK-P07-013  | Hapus Menu & Rute BYOK Frontend, Arsip DB Keys      | Fullstack Dev
-------+---------------+-----------------------------------------------------+------------------
H-04   | TASK-P07-006  | Resiliensi SSE: Cookie Auth, 15s Heartbeat, Resume  | Backend Dev 3
H-04   | TASK-P07-007  | Standardisasi Skema JSON Event Domain SSE           | Backend Lead
H-04   | TASK-P07-012  | Refactor Halaman Kategori & User List ke DataTable  | Frontend Dev 2
-------+---------------+-----------------------------------------------------+------------------
H-05   | MILESTONE 1   | Review Akhir Minggu 1 & Integrasi Backend - DB - FE | Seluruh Tim
-------+---------------+-----------------------------------------------------+------------------
H-06   | TASK-P07-008  | Hook `useRealtimeStream` & Auto-Reconnect Resilient | Frontend Dev 3
H-06   | TASK-P07-009  | Registri Query Key & Surgical Cache Invalidation    | Frontend Lead
H-06   | TASK-P07-016  | Implementasi Dual Health Gate (`/live` & `/ready`)  | Backend Dev 1
-------+---------------+-----------------------------------------------------+------------------
H-07   | TASK-P07-010  | Optimistic Updates Transaksi & Anggaran             | Frontend Dev 1 & 2
H-07   | TASK-P07-014  | Audit Universal Cursor Pointer & Elemen Interaktif  | Frontend Dev 3
H-07   | TASK-P07-017  | Graceful Shutdown 30s Gin Server & Docker Config    | DevOps Engineer
-------+---------------+-----------------------------------------------------+------------------
H-08   | TASK-P07-015  | Audit Aksesibilitas WCAG 2.1 AA & Keyboard Nav      | Frontend Dev 3 & QA
H-08   | TASK-P07-018  | Template Migrasi Expand/Contract Skema Database     | Backend Lead & DBA
H-08   | TASK-P07-009  | Pengujian Stabilitas Surgical Invalidation Realtime | QA & Frontend
-------+---------------+-----------------------------------------------------+------------------
H-09   | INTEGRATION   | Pengujian Beban Multi-Pod & Simulasi Network Drop   | QA & DevOps
H-09   | DRILL TEST    | Simulasi Zero-Downtime Rolling Update 200 RPS       | DevOps & Backend
-------+---------------+-----------------------------------------------------+------------------
H-10   | MILESTONE 2   | Final Polish, Dokumentasi Sign-off, Freeze P07      | Seluruh Tim
```

---

## Analisis Jalur Kritis (Critical Path Analysis)

Jalur kritis (urutan ketergantungan terpanjang yang menentukan jadwal rilis fase P07) adalah sebagai berikut:

```
[TASK-P07-003: Redis Versioned Cache]
               │
               ▼
[TASK-P07-001: Middleware Entitlement Backend]
               │
               ▼
[TASK-P07-002: Pasang Entitlement Endpoint Legacy] ──┐
                                                    │
[TASK-P07-004: Hub SSE Broker]                      │
               │                                    │
               ▼                                    │
[TASK-P07-005: Redis Pub/Sub Backbone]              │
               │                                    │
               ▼                                    │
[TASK-P07-006: Resiliensi SSE & Resume Last-Event]  │
               │                                    │
               ▼                                    │
[TASK-P07-008: Frontend Hook useRealtimeStream]     │
               │                                    │
               ▼                                    │
[TASK-P07-009: Surgical Cache Invalidation] ◄───────┘
               │
               ▼
[TASK-P07-017: Graceful Shutdown & Rolling Deploy Test]
               │
               ▼
[Sign-Off Fase P07 Siap Masuk P08 Hardening]
```

### Penjelasan Ketergantungan Kritis:
1. **TASK-P07-003 harus tuntas sebelum TASK-P07-001**: Middleware otorisasi entitlement bergantung langsung pada struktur data versi integer di Redis. Tanpa Redis versioned cache, middleware tidak dapat memvalidasi status secara realtime.
2. **TASK-P07-004, 005, 006 membentuk rantai SSE backend**: Broker lokal harus selesai sebelum dihubungkan ke Redis pub/sub, dan penanganan `Last-Event-ID` ring-buffer membutuhkan koneksi Redis pub/sub yang stabil.
3. **TASK-P07-008 & TASK-P07-009 membutuhkan kontrak SSE matang**: Frontend hook tidak dapat diintegrasikan dengan TanStack Query sampai payload JSON dan mekanisme resume SSE di sisi server telah dibakukan.
4. **TASK-P07-017 adalah gerbang akhir**: Uji rolling deployment tanpa downtime hanya dapat diverifikasi saat seluruh koneksi HTTP SSE dan koneksi database pool telah menerapkan mekanisme graceful drain.

---

## Matriks Hubungan Ketergantungan Task (Dependency Matrix)

| Task ID | Nama Task | Prasyarat (Blockers) | Menjadi Prasyarat Untuk (Unblocks) |
|---|---|---|---|
| **TASK-P07-001** | Middleware Entitlement Backend | TASK-P07-003 | TASK-P07-002 |
| **TASK-P07-002** | Pasang Entitlement Endpoint Legacy | TASK-P07-001 | TASK-P07-009, P08 Go-Live |
| **TASK-P07-003** | Migrasi ke Redis Berversi | *None* | TASK-P07-001, TASK-P07-005 |
| **TASK-P07-004** | Hub SSE Broker Lokal | *None* | TASK-P07-005, TASK-P07-006 |
| **TASK-P07-005** | Redis Pub/Sub Jembatan Multi-Pod | TASK-P07-003, TASK-P07-004 | TASK-P07-006, TASK-P07-007 |
| **TASK-P07-006** | Resiliensi SSE, Heartbeat, Resume | TASK-P07-004, TASK-P07-005 | TASK-P07-008 |
| **TASK-P07-007** | Skema JSON Event SSE Standar | TASK-P07-004 | TASK-P07-008, TASK-P07-009 |
| **TASK-P07-008** | Hook Frontend useRealtimeStream | TASK-P07-006, TASK-P07-007 | TASK-P07-009 |
| **TASK-P07-009** | Surgical Cache Invalidation | TASK-P07-008, TASK-P07-002 | TASK-P07-010 |
| **TASK-P07-010** | Optimistic Update Transaksi | TASK-P07-009 | QA Uji Regresi |
| **TASK-P07-011** | Universal DataTable TanStack Table | *None* | TASK-P07-012 |
| **TASK-P07-012** | Refactor Halaman ke DataTable | TASK-P07-011 | TASK-P07-014 |
| **TASK-P07-013** | Pembersihan Tuntas Menu BYOK | *None* | P08 Hardening Security |
| **TASK-P07-014** | Audit Universal Cursor Pointer | TASK-P07-012 | TASK-P07-015 |
| **TASK-P07-015** | Audit Aksesibilitas WCAG 2.1 AA | TASK-P07-014 | Frontend Freeze P07 |
| **TASK-P07-016** | Dual Health Gate (/live & /ready) | TASK-P07-003 | TASK-P07-017 |
| **TASK-P07-017** | Graceful Shutdown & Rolling Deploy | TASK-P07-016, TASK-P07-006 | P08 Load Test |
| **TASK-P07-018** | Prosedur Migrasi Expand/Contract | *None* | Seluruh Migrasi P08+ |

---

## Rencana Mitigasi Risiko Jadwal

1. **Risiko Koneksi SSE Stagnan di Balik Reverse Proxy**:
   - *Mitigasi*: Konfigurasi Nginx proxy directive (`proxy_buffering off; proxy_read_timeout 3600s; proxy_set_header Connection '';`) diverifikasi pada lingkungan staging sejak Hari ke-3.
2. **Risiko Cache Invalidation Storm saat Broadcast Massal**:
   - *Mitigasi*: Pengujian granularitas key TanStack Query diuji dengan unit test hook mock untuk memastikan tidak ada pemanggilan root `invalidateQueries()` tanpa argumen predicate/key.
3. **Risiko Regresi Komponen Halaman Lama saat Dipasang `<DataTable>`**:
   - *Mitigasi*: Implementasi DataTable mengadopsi skema adaptif kolom yang independen dan backward-compatible dengan skema data transaksi FinTrack v2.
