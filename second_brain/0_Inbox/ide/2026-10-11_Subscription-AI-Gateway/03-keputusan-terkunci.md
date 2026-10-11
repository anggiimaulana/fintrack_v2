# 03 — Keputusan Terkunci

Keputusan di bawah ini **disetujui pemilik produk** pada sesi brainstorming 2026-10-11 dan menjadi
dasar seluruh desain. Perubahan hanya boleh lewat keputusan baru bertanggal, bukan suntingan diam-diam.

| ID | Topik | Keputusan | Alasan / Catatan |
|---|---|---|---|
| K-01 | Model langganan | **Per-user (B2C)** | FinTrack adalah aplikasi keuangan pribadi; tidak perlu `tenant_id` |
| K-02 | Model role | **`super_admin` + `user`** (admin lama dipromosikan) | Menjaga kesederhanaan; entitlement dipisah dari role |
| K-03 | Cakupan token | **Semua fitur AI** (chat, vision, voice, embedding) | Mencerminkan biaya nyata |
| K-04 | Harga token | **Dinamis**: base per 1.000 token + markup (kali & persen) + pengali per-modality | Diatur super admin sepenuhnya |
| K-05 | Siklus kuota | **Per-hari** (bukan bulan kalender), TZ Asia/Jakarta | Adil untuk bulan 28/29/30/31 |
| K-06 | Reset kuota | **Reset tiap awal siklus**, tanpa rollover | Rollover ditolak (YAGNI, mengurangi upgrade) |
| K-07 | Renewal | **Hybrid** — auto bila metode mendukung, manual + reminder bila tidak | Midtrans tidak auto-recurring untuk semua metode |
| K-08 | Upgrade | **Instan + prorated** | Standar SaaS global |
| K-09 | Downgrade | **Berlaku akhir siklus**, tanpa refund, **soft-limit** | Tidak menghapus data user |
| K-10 | Switch gateway | **Global default + override per-plan**; berlaku untuk checkout baru | Transaksi in-flight tetap terikat gateway asalnya |
| K-11 | Invoice | **Dasar, siap PPN** | Tanpa PPN di fase awal, struktur siap ditambah |
| K-12 | Entitlement | **Fitur (bool) + limit numerik** | Sesuai permintaan "centang" |
| K-13 | Penegakan hak akses | **Backend otoritatif**; FE hanya cermin UX | Klien tidak pernah dipercaya |
| K-14 | Realtime RBAC | **Lookup per-request + cache berversi** (bukan di JWT) | Zero login/logout & zero downtime |
| K-15 | Efek ubah plan | **Efektif langsung**, tanpa migrasi data user | Reversibel, aman |
| K-16 | BYOK | **Dihapus total**; kunci lama diarsipkan | Semua trafik lewat pool terpusat |
| K-17 | Topologi AI | **Pool multi-endpoint + failover + circuit breaker + kill switch** | Mengurangi blast radius |
| K-18 | Routing AI | **Tabel route per-use-case**; embedding dikunci 768d | Cocok dengan `vector(768)` yang ada |
| K-19 | Sumber angka token | **Usage provider + fallback estimasi** | Akurasi billing |
| K-20 | Transport realtime | **Surgical cache update + SSE** (perubahan eksternal) | Ringan, memakai pola SSE yang ada |
| K-21 | Infra tambahan | **Redis** (cache bersama + pub/sub) | Syarat zero downtime multi-instance |
| K-22 | DataTable | **TanStack Table v8 + state di URL** | Konsisten di semua halaman |
| K-23 | Zero downtime | **Expand/contract + rolling deploy + feature flag** | Definisi realistis |
| K-24 | Pendekatan | **Hibrida A+B** (incremental + modul baru bersih ber-interface) | Zero downtime + batas modul jelas |
| K-25 | Pembagian fase | **7 fase P02–P08** | Sesuai permintaan struktur |
| K-26 | Struktur dokumen | **4_Resources/Dokumen-Spesifikasi/{BRD,PRD,Diagram}**; 1 folder ide; 3_Areas +2 area | Mengikuti pola dagangbot/simbot |
| K-27 | Uang | **Integer rupiah** (tanpa float) | Mencegah galat pembulatan |
| K-28 | Ledger token | **Append-only + reserve/commit/release + idempotency key** | Anti race & anti double-charge |
| K-29 | Audit | **`audit_logs` append-only** untuk semua aksi super admin & billing | Non-repudiation & sengketa |
| K-30 | Compliance | Tunduk **UU PDP No. 27/2022**; kebijakan retensi tertulis | Kewajiban hukum |

## Prinsip lintas-keputusan

1. **Backend adalah satu-satunya otoritas.** Frontend hanya UX.
2. **Data finansial user tidak pernah dihapus karena alasan billing.**
3. **Setiap mutasi uang/token bersifat idempoten & tercatat.**
4. **Semua perubahan konfigurasi (plan, gateway, routing, kunci) berlaku tanpa deploy ulang.**
5. **Setiap klaim arsitektur harus punya test yang membuktikannya.**
