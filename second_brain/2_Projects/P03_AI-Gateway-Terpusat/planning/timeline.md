# Timeline & Dependensi — P03 AI Gateway Terpusat

Dokumen ini memetakan urutan eksekusi, dependensi antar modul, dan milestone penyelesaian Fase P03.

---

## 1. Diagram Alur Dependensi (DAG)

```
[TASK-P03-001: SQL Schema & Migration]
          │
          ├──► [TASK-P03-002: Domain Entities & Interfaces]
          │             │
          │             ├──► [TASK-P03-003: SQL Repository & Key Decryption]
          │             │             │
          │             ├──► [TASK-P03-004: OpenAI-Compatible HTTP Client]
          │             │             │
          │             │             ├──► [TASK-P03-005: Circuit Breaker State Machine]
          │             │             │             │
          │             │             ├──► [TASK-P03-008: Metering Token & Estimator]
          │             │             │             │
          │             │             └─────────────┴──► [TASK-P03-006: Router & Failover Engine]
          │             │                                           │
          │             │                                           ├──► [TASK-P03-007: Health Check Worker]
          │             │                                           │
          │             │                                           ├──► [TASK-P03-009: Adapter AIUsecase]
          │             │                                           │             │
          │             │                                           └──► [TASK-P03-010: Embedder 768d Lock]
          │             │                                                         │
          │             └─────────────────────────────────────────────────────────┼──► [TASK-P03-013: Integration Tests]
          │                                                                       │
          └──► [TASK-P03-011: Deprecate BYOK Endpoint]                            │
                        │                                                         │
                        └──► [TASK-P03-012: Frontend BYOK Cleanup]                │
                                      │                                           │
                                      └───────────────────────────────────────────┴──► [TASK-P03-014: Zero-Downtime Verification]
```

---

## 2. Rincian Milestone Harian

### Hari 1: Persiapan Skema Database & Entitas Domain
- Eksekusi migrasi `0006_ai_gateway.sql`.
- Penambahan kolom arsip pada `gemini_api_keys`.
- Seed konfigurasi default (9router primer).
- Output: DB schema up-to-date, struct domain di Go siap.

### Hari 2: Lapisan Repository & Enkripsi Kredensial
- Implementasi query SQL untuk endpoint dan route.
- Implementasi enkripsi AES-GCM untuk API key provider.
- In-memory routing table cache dengan thread-safe lock.
- Output: Data rute dapat diakses cepat dari memory Go.

### Hari 3: Engine Koneksi OpenAI-Compatible
- Bangun HTTP client yang mendukung format `/chat/completions` dan `/embeddings`.
- Streaming parser untuk server-sent events (SSE).
- Ekstraksi delta text dan buffer streaming.
- Output: Komunikasi lancar dengan 9router lokal port 20128.

### Hari 4: Circuit Breaker & Resiliensi Jaringan
- State machine Closed, Open, Half-Open.
- Tracking failure threshold (5 error) dan cooldown timer (30 detik).
- Fast failover guard tanpa membebani server yang sedang down.
- Output: Unit test circuit breaker hijau.

### Hari 5: Routing Table Dinamis & Failover Engine
- Pengambilan rute berdasarkan urutan prioritas (`priority ASC`).
- Mekanisme fallback otomatis ke cloud provider jika 9router gagal.
- Heuristic token metering fallback.
- Output: `AIGatewayService` siap menerima request dari usecase.

### Hari 6: Background Health Checker
- Worker goroutine periodik setiap 60 detik.
- Ping ringan dengan timeout 5 detik.
- Update status `healthy` / `unhealthy` di database dan runtime memory.
- Output: Status kesehatan endpoint terpantau otomatis.

### Hari 7: Adapter Tipis pada `AIUsecase` & Embedder 768d
- Refactor pemanggilan chat, vision, voice di `ai_usecase.go` menuju interface gateway.
- Kunci dimensi embedding pada 768 float; tolak model dengan dimensi selain 768.
- Validasi pencarian RAG transaksi keuangan tetap presisi.
- Output: Seluruh interaksi AI di backend beralih ke gateway terpusat.

### Hari 8: Deprecate BYOK & Pembersihan Frontend
- Nonaktifkan endpoint API key user (return HTTP 410 Gone).
- Hapus menu UI, modal, dan formulir BYOK di Frontend Next.js.
- Perbaiki pesan sistem ke pengguna saat terjadi kendala upstream.
- Output: Frontend bebas dari elemen konfigurasi kunci pribadi.

### Hari 9: Pengujian Integrasi End-to-End
- Uji skenario failover: 9router dimatikan -> fallback ke cloud.
- Uji skenario circuit breaker: 5 kali error berurutan -> rute diblokir 30 detik.
- Uji stream chat interaktif dengan financial context injection.
- Output: Laporan test integrasi sukses 100%.

### Hari 10: Rollout, Feature Flag & Zero Downtime Sign-Off
- Aktivasi feature flag `AI_GATEWAY_CENTRALIZED_ENABLED`.
- Monitoring metrik latensi, memory consumption, dan error rate.
- Update dokumentasi proyek dan tracking board.
- Output: Fase P03 selesai dan siap menjadi fondasi Fase P04 (Billing & Token Ledger).

---

## 3. Matriks Dependensi Antar Fase

| Modul Terkait | Dependensi dari P03 | Keterangan Hubungan |
|---|---|---|
| **P01 (Fondasi & Auth)** | Prasyarat (Harus Selesai) | P03 memanfaatkan DB pgvector 768d, auth context user, dan crypto key yang ada di P01. |
| **P02 (Fondasi Arsitektur)** | Paralel / Bersinergi | Standarisasi logging, Redis cache invalidation, dan interface conventions. |
| **P04 (Billing & Subscriptions)** | Target Konsumsi | P04 akan membaca metadata `TokenUsage` dari P03 untuk mendebit saldo ledger token user. |
| **P06 (Super Admin)** | Target Kontrol | P06 akan menyediakan UI Backoffice untuk mengelola tabel `ai_endpoints` dan `ai_routes`. |
