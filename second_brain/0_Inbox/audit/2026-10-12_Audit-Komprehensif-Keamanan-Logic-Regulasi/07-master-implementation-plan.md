# Master Implementation Plan: Rencana Tindak Lanjut Temuan Audit FinTrack v2

**Kode Dokumen**: AUDIT-PLAN-01  
**Klasifikasi**: Rencana Kerja Rekayasa & Manajemen Rilis (Engineering Roadmap & Sprint Planning)  

---

## 1. Strategi & Prinsip Eksekusi

Rencana tindak lanjut ini disusun dengan prinsip **Zero Breaking Changes**, **Security By Default**, dan **Evidence-Based Engineering**. Seluruh tindakan perbaikan dikelompokkan ke dalam 2 fase sprint terstruktur:

1. **Sprint P08-Patch (Quick-Wins & Critical Fixes — Target H+1 s.d H+3)**:
   Fokus pada mitigasi kerentanan keamanan siber (XSS, SSRF, CSRF) dan perbaikan race condition rotasi token tanpa perubahan skema database besar.
2. **Sprint P09 (Functional Rollout & Scalability — Target H+4 s.d H+10)**:
   Fokus pada penyediaan rute regulasi swalayan UU PDP (`export-data` & `delete-account`), aktivasi alur checkout mandiri paket langganan (Midtrans Snap), dan optimasi koneksi PgBouncer.

---

## 2. Rincian Paket Kerja (Work Packages)

### Paket Kerja 1: Patch Keamanan Segera (Sprint P08-Patch)

| Task ID | Nama Tugas | Target Berkas | Kriteria Penerimaan (DOD) | Prioritas |
|---|---|---|---|:---:|
| **TASK-FIX-01** | Sanitasi URL Anchor Tag Markdown | `frontend/components/ai/ChatMessage.tsx` | URL dengan protokol `javascript:` atau `data:` dinonaktifkan (`#`), hanya `http:`, `https:`, dan `mailto:` yang dapat diklik. 0 celah XSS DOM. | **P0 (Kritis)** |
| **TASK-FIX-02** | Validasi Skema & Anti-SSRF pada AI BaseURL | `backend/internal/admin/aigateway.go` | Pendaftaran endpoint dengan IP loopback, link-local (169.254.169.254), atau private network ditolak HTTP 400 pada mode produksi. | **P0 (Kritis)** |
| **TASK-FIX-03** | Perbaikan Atomik Race Condition Rotasi Token | `backend/internal/repository/postgres/refresh_token_repo.go` | Di dalam `Rotate()`, jika `revokedAt != nil` pasca-row-lock, transaksi digagalkan dan seluruh token dicabut via `RevokeAllForUser`. 0 celah concurrent reuse. | **P0 (Kritis)** |
| **TASK-FIX-04** | Penguatan OriginCheck Anti-CSRF | `backend/internal/delivery/http/middleware/origin_check.go` | Request mutasi (`POST/PUT/DELETE`) yang membawa cookie sesi `ft_at` tanpa header Origin/Referer ditolak HTTP 403 Forbidden. | **P1 (Tinggi)** |
| **TASK-FIX-05** | Isolasi Delimiter XML pada RAG AI Context | `backend/internal/usecase/ai_usecase.go` | Deskripsi transaksi dibungkus tag `<user_transactions>` dengan tag-breakout sanitization. System prompt menginstruksikan teks di dalam tag murni sebagai data non-eksekusi. | **P1 (Tinggi)** |

---

### Paket Kerja 2: Fitur Regulasi & Alur Bisnis (Sprint P09)

| Task ID | Nama Tugas | Target Berkas | Kriteria Penerimaan (DOD) | Prioritas |
|---|---|---|---|:---:|
| **TASK-REG-01** | Endpoint Hapus Akun Swalayan UU PDP | `backend/internal/delivery/http/handler/` & `router.go` | Rute `DELETE /api/v1/user/account` meminta konfirmasi password terkini, mengeksekusi `AnonymizeUserData`, mencabut sesi, dan mencatat `audit_logs`. | **P1 (Tinggi)** |
| **TASK-REG-02** | Endpoint Ekspor Data Swalayan UU PDP | `backend/internal/delivery/http/handler/` & `router.go` | Rute `GET /api/v1/user/export-data` mengunduh arsip JSON berisi profil, histori transaksi, riwayat chat, dan invoice pengguna (Data Portability). | **P1 (Tinggi)** |
| **TASK-REG-03** | Menu Privasi & Regulasi di Frontend Profile | `frontend/app/(dashboard)/user/profile/` | Antarmuka pengguna menyediakan tombol "Ekspor Data Pribadi" dan "Hapus Akun Permanen" dengan dialog konfirmasi password modal. | **P1 (Tinggi)** |
| **TASK-PAY-01** | Integrasi Endpoint Checkout Pembayaran | `backend/internal/delivery/http/handler/payment_handler.go` | ✅ **SELESAI (2026-10-13)** — `POST /api/v1/payments/checkout` terdaftar, memanggil `CheckoutService.Checkout()`, mengembalikan `snap_token` / `redirect_url` / `instructions`. | **P1 (Tinggi)** |
| **TASK-PAY-02** | Tombol Beli Paket pada Subscription UI | `frontend/app/(dashboard)/user/settings/subscription/` | ✅ **SELESAI (2026-10-13)** — halaman langganan menjadi hub swalayan: katalog dinamis, upgrade, downgrade, riwayat pembayaran, tagihan, instruksi transfer, dan unggah bukti. (Popup `window.snap.pay` menyusul saat Midtrans diaktifkan; saat ini `redirect_url` dibuka di tab baru.) | **P1 (Tinggi)** |
| **TASK-INF-01** | Setup PgBouncer Connection Pooler | `docker-compose.yml` & `deploy/` | Container PgBouncer aktif di port 6432, mereduksi koneksi fisik backend ke PostgreSQL di bawah 30 koneksi pada beban multi-pod. | **P2 (Sedang)** |
| **TASK-INF-02** | Background Sweeper Koneksi Mengambang SSE | `backend/internal/realtime/hub.go` | Worker background memutus koneksi SSE yang tidak merespons heartbeat ping > 45 detik, mencegah kebocoran memori pada jaringan seluler terputus. | **P2 (Sedang)** |

---

## 3. Matriks Pengujian & Verifikasi (Verification Gates)

Setiap implementasi perbaikan wajib melewati 4 gerbang verifikasi sebelum di-merge ke branch utama:

1. **Unit Testing & Security Assertion**:
   - Test XSS Markdown: String `[evil](javascript:alert(1))` terbukti menghasilkan elemen dengan `href="#"`.
   - Test SSRF: Injeksi `http://169.254.169.254` pada pembuatan endpoint AI terbukti ditolak HTTP 400.
   - Test Concurrency Refresh: 10 goroutine yang me-refresh token yang sama serentak menghasilkan tepat 1 keberhasilan dan 9 penolakan dengan pencabutan total token pengguna.
2. **Integration Test Suite**:
   - Seluruh suite integrasi `backend/tests/` (P06, P07, P08) wajib 100% lulus.
3. **Frontend Lint & Build**:
   - `pnpm lint` wajib 0 error.
   - `pnpm test` wajib 100% lulus.
   - `pnpm build` kompilasi Next.js wajib sukses tanpa kegagalan tipe data.
4. **Audit Trail Logging**:
   - Seluruh mutasi administratif dan kepatuhan hukum wajib tercatat di tabel `audit_logs` dengan format skema yang valid.
