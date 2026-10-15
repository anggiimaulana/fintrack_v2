# Voice Input Cerdas — Smart Routing, Quota-Aware Fallback & Konfigurasi Dinamis — FinTrack v2

| Field | Isi |
|---|---|
| **Dokumen** | Desain & Spesifikasi: Voice Input Cerdas (Smart Routing, Daily Limit, Dinamis via Super Admin) |
| **Tanggal** | 2026-10-15 (revisi 2 — jawaban open question) |
| **Status** | ✅ **Terimplementasi & terverifikasi live** (lihat §8) |
| **Penulis** | Tim Rekayasa FinTrack v2 |
| **Cakupan** | `frontend/components/ai/VoiceInput.tsx`, `frontend/hooks/useVoiceMode.ts`, `backend/internal/delivery/http/handler/ai_handler.go`, `backend/internal/usecase/ai_usecase.go`, `backend/internal/admin/settings.go`, `backend/internal/entitlement/*`, `backend/internal/billing/*`, `pkg/contracts/*` |
| **Ketergantungan** | F6 (usage real-time), event `usage_changed` (2026-10-15), SSE realtime, `runtimeconfig` + `settings` dinamis |

---

## 1. Latar Belakang

Saat ini ada **dua jalur "voice" yang tidak nyambung**:

1. **`VoiceInput.tsx`** (frontend) — memakai **Web Speech API browser** (`SpeechRecognition`).
   Audio **tidak pernah menyentuh backend**; transkripsi terjadi di sisi vendor browser.
   Konsekuensi: **tidak menyentuh kuota**, **tidak ter-gate** oleh `voice_input`, dan
   **hanya jalan di Chrome/Edge**.
2. **`POST /api/v1/ai/voice`** (backend) — endpoint lengkap: reserve → transkripsi via
   AI Gateway → commit ke token ledger → emit `usage_changed`. **Namun tidak dipanggil
   oleh kode frontend mana pun** → praktis **dead code**.

Selain itu ada halaman **`/user/voice`** yang juga memakai Web Speech API (bukan `/ai/voice`),
sehingga fitur `voice_input` yang dijual di paket Pro/Business **tidak benar-benar
ditegakkan** dan kuota token **tidak pernah berkurang** untuk pemakaian suara.

### 1.1 Permintaan Produk (hasil keputusan)

> Voice harus **berfungsi nyata** (bukan dead code), **mengurangi kuota real-time**,
> **cerdas menghadapi paket/kuota**, dan **dapat dikonfigurasi Super Admin secara dinamis**.

| Kode | Keputusan Produk |
|---|---|
| **Q-01** | Durasi rekam maksimum **default 60 detik**, **dapat dikonfigurasi Super Admin** (dinamis) |
| **Q-02** | Saat fallback, tawarkan **upgrade hint** — **tidak memaksa** (dismissible) |
| **Q-03** | Mode browser tersedia untuk semua paket, **dibatasi per hari** (mis. Free = 1/hari), **berbeda per paket**, **dinamis** |
| **Q-04** | Riwayat transkripsi di **server** (source of truth) **dan** lokal (cache cepat) |
| **Q-05** | **VU meter (indikator level mikrofon) wajib** saat merekam |
| **Q-06** | Bahasa default **`id`**; **mode multi-bahasa** disiapkan, **dapat diaktifkan via Super Admin** (dinamis: `id` saja atau multi) |
| **Q-07** | Halaman `/user/voice` **digabung** ke chat (hapus halaman standalone) |
| **Q-08** | **Fallback wajib** (tidak boleh dead-end) |

### 1.2 Keputusan Lanjutan (N-01 … N-05)

| Kode | Keputusan Produk |
|---|---|
| **N-01** | **Retensi riwayat** = **90 hari**, **dinamis** (setting `voice_history_retention_days`) |
| **N-02** | Pemilihan bahasa **per-rekaman** (saat `multi`), **dinamis** |
| **N-03** | Reset limit harian mengikuti **zona waktu server** — **dinamis** (`voice_daily_reset_timezone`: `UTC` atau `Asia/Jakarta`) |
| **N-04** | Transcript **dapat diedit** sebelum dikirim ke chat (perilaku chat existing); **dinamis** (`voice_allow_edit_before_send`) |
| **N-05** | **Export riwayat voice** → **fase lanjut** (belum diimplementasikan) |

> **Prinsip menyeluruh:** **SEMUA** knob voice bersifat **dinamis** dan dapat dikonfigurasi
> Super Admin tanpa redeploy (lihat §4.1). Tidak ada nilai voice yang hardcode.

---

## 2. Ringkasan Eksekutif (TL;DR)

> **Inti solusi: satu komponen voice, tiga mode, dua lapis limit, konfigurasi dinamis.**
> `VoiceInput` menjadi **router cerdas** yang memilih *transport transkripsi* berdasarkan
> entitlement real-time, dengan **fallback berlapis** dan **VU meter wajib**.

> **PERUBAHAN PENTING dari revisi 1:** keputusan Q-01/Q-03/Q-06 membuat **backend ikut berubah**.
> Sebelumnya desain ini 100% frontend; sekarang menambah:
> (a) **setting global dinamis** (`voice_max_duration_seconds`, `voice_language_mode`),
> (b) **limit harian per paket** (`voice_requests_per_day`) dengan usage resolver baru,
> (c) **penyimpanan riwayat transkripsi** di server,
> (d) **endpoint konfigurasi voice** untuk dibaca klien.

> **Sumber keputusan = entitlement + config (bukan hardcode).** Klien membaca
> `features.voice_input`, `usage.voice_requests_per_day`, dan config global dari endpoint
> yang sudah ada/ditambah — semuanya real-time.

> **Backend tetap otoritatif.** Klien memilih jalur; backend **selalu** memvalidasi ulang
> (feature gate + limit gate + ledger reserve). Bila klien salah tebak (race), backend
> menjawab `402/403/429` dan klien **otomatis fallback** — bukan error mentah.

> **Tidak ada regresi.** Mode browser selalu tersedia (dibatasi per hari), jadi tidak ada
> dead-end. Bila limit harian browser juga habis → pesan jelas + hint upgrade (tidak dipaksa).

| Aspek | Keputusan |
|---|---|
| Transport | Web Speech API (browser) **atau** `POST /ai/voice` (server) |
| Pemilih mode | Klien (entitlement + config) + verifikasi backend |
| Fallback | Wajib, 3 lapis: kuota → fitur → browser; **+ limit harian** |
| Kuota | Server: ledger 2-fase + `usage_changed`; Browser: limit harian per paket |
| Real-time | SSE `usage_changed` → bar kuota update tanpa reload |
| Durasi | Default 60s, **dinamis** (`voice_max_duration_seconds`) |
| Bahasa | Default `id`, **mode multi dinamis**; pilih **per-rekaman** (N-02) |
| Riwayat | Server (source of truth) + lokal (cache cepat); retensi **dinamis 90 hari** (N-01) |
| Zona reset harian | **Dinamis** (`voice_daily_reset_timezone`, usulan `Asia/Jakarta`) (N-03) |
| Edit sebelum kirim | **Dinamis** (`voice_allow_edit_before_send`, default aktif) (N-04) |
| VU meter | **Wajib** (Web Audio `AnalyserNode`), toggle dinamis (Q-05) |
| Halaman voice | **Digabung ke chat** (standalone dihapus) |
| Export riwayat | Fase lanjut (N-05) |
| Konfigurasi | **Semua dinamis** via Super Admin (lihat §4.1) |

---

## 3. Daftar Dokumen

| # | Dokumen | Isi |
|---|---|---|
| **00** | [`README.md`](README.md) (dokumen ini) | Ringkasan eksekutif, keputusan produk, peta keputusan, indeks |
| **01** | [`01-audit-kondisi-saat-ini.md`](01-audit-kondisi-saat-ini.md) | Audit kondisi terkini (bukti `file:line`), akar masalah, inventaris dead code |
| **02** | [`02-mekanisme-voice.md`](02-mekanisme-voice.md) | Mekanisme kedua jalur, VU meter, konfigurasi bahasa, perbandingan |
| **03** | [`03-spec-smart-routing.md`](03-spec-smart-routing.md) | Matriks keputusan, state machine, kontrak API, payload, aturan limit harian |
| **04** | [`04-desain-teknis.md`](04-desain-teknis.md) | Desain teknis frontend **& backend**, tipe, hook, komponen, urutan event |
| **05** | [`05-implementation-plan.md`](05-implementation-plan.md) | Rencana implementasi berfase (backend + frontend + admin UI) |
| **06** | [`06-testing-plan.md`](06-testing-plan.md) | Rencana pengujian lengkap dengan checklist `[ ]` |
| **07** | [`07-risiko-dan-mitigasi.md`](07-risiko-dan-mitigasi.md) | Risiko, edge case, mitigasi, keputusan terkunci final |

---

## 4. Peta Keputusan Terkunci

| Kode | Keputusan | Alasan |
|---|---|---|
| **K-VOI-01** | Backend otoritatif; klien hanya **pemilih jalur**, bukan gerbang | Konsisten dengan K-ENT-01 (entitlement) |
| **K-VOI-02** | Fail-safe: setiap kegagalan transport server → **fallback ke browser**, tidak pernah dead-end | User tidak boleh kehilangan kemampuan bicara |
| **K-VOI-03** | Fallback **tidak** mengubah hasil: transcript selalu dikirim sebagai **teks** | Perilaku seragam di semua mode |
| **K-VOI-04** | Kuota habis = **bukan error keras** untuk voice; cukup downgrade ke browser | Voice = input UX |
| **K-VOI-05** | Audio hanya dikirim ke server bila `voice_input` ✅ **dan** kuota ✅ **dan** recorder ✅ | Privasi + biaya |
| **K-VOI-06** | Setiap transkripsi server **wajib** settle ledger + emit `usage_changed` | Akuntansi kuota benar & real-time |
| **K-VOI-07** | Mode yang dipakai **selalu terlihat** ke user (badge/hint) | Transparansi |
| **K-VOI-08** | **Durasi rekam maksimum** = setting global dinamis (`voice_max_duration_seconds`, default 60s) | Q-01 |
| **K-VOI-09** | **Limit harian voice** per paket (`voice_requests_per_day`) dinamis via `plan_limits`; **berlaku untuk KEDUA mode** | Q-03 |
| **K-VOI-10** | **Riwayat transkripsi** disimpan di server (source of truth) + cache lokal (optimistic) | Q-04 |
| **K-VOI-11** | **VU meter wajib** saat merekam (Web Audio `AnalyserNode`) | Q-05 |
| **K-VOI-12** | Bahasa default `id`; **mode multi-bahasa** dinamis (`voice_language_mode`: `id_only` \| `multi`) | Q-06 |
| **K-VOI-13** | Halaman `/user/voice` **digabung ke chat**; rute standalone dihapus | Q-07 |
| **K-VOI-14** | Fallback **wajib**; hint upgrade bersifat **opsional/dismissible** (tidak memaksa) | Q-02, Q-08 |
| **K-VOI-15** | **Retensi riwayat** dinamis (`voice_history_retention_days`, default 90); cleaner harian | N-01 |
| **K-VOI-16** | Pemilihan bahasa **per-rekaman** saat `multi`; bahasa diteruskan ke STT/server | N-02 |
| **K-VOI-17** | Reset limit harian mengikuti **zona waktu** dinamis (`voice_daily_reset_timezone`) | N-03 |
| **K-VOI-18** | Transcript **dapat diedit** sebelum dikirim (dinamis `voice_allow_edit_before_send`) | N-04 |
| **K-VOI-19** | **Semua** knob voice dinamis; tidak ada hardcode | N-01..N-05 |
| **K-VOI-20** | Export riwayat voice ditunda ke fase lanjut | N-05 |

### 4.1 Katalog Konfigurasi Dinamis (Super Admin)

> **Semua** nilai berikut diatur Super Admin (tanpa redeploy) via `PUT /admin/settings`
> (global) atau plan editor (per paket). Di-resolve lewat `runtimeconfig.Resolver`
> (DB-over-env, TTL 5s).

**Global (tabel `settings`, per-platform):**

| Key | Tipe | Default | Fungsi |
|---|---|---|---|
| `voice_max_duration_seconds` | int (1..600) | `60` | Durasi rekam maksimum |
| `voice_language_mode` | enum `id_only`\|`multi` | `id_only` | Mode bahasa |
| `voice_default_language` | enum (mis. `id`,`en`) | `id` | Bahasa default |
| `voice_languages` | string CSV | `id` | Daftar bahasa saat `multi` (mis. `id,en`) |
| `voice_history_retention_days` | int (1..3650) | `90` | Retensi riwayat (N-01) |
| `voice_daily_reset_timezone` | enum `UTC`\|`Asia/Jakarta` | `Asia/Jakarta` | Zona reset harian (N-03) |
| `voice_allow_edit_before_send` | bool | `true` | Izinkan edit transcript (N-04) |
| `voice_server_enabled` | bool | `true` | Kill switch mode server |
| `voice_vu_meter_enabled` | bool | `true` | Tampilkan VU meter (Q-05 wajib → default true) |
| `voice_fallback_to_browser` | bool | `true` | Izinkan fallback (Q-08 wajib → default true) |

**Per paket (tabel `plan_limits`, via plan editor):**

| Key | Tipe | Contoh | Fungsi |
|---|---|---|---|
| `voice_requests_per_day` | int (`-1` = unlimited) | Free=1, Pro=50, Business=-1 | Limit harian (Q-03) |

**Feature (tabel `plan_entitlements`):**

| Key | Tipe | Fungsi |
|---|---|---|
| `voice_input` | bool | Akses mode server (transkripsi AI) |

> **Catatan default N-03:** usulan `Asia/Jakarta` (WIB) karena basis pengguna Indonesia —
> lebih intuitif ("reset tengah malam WIB"). Tetap **dinamis**: bisa diubah ke `UTC`
> atau zona lain. Semua perhitungan harian memakai zona ini secara konsisten.

---

## 5. Peta Temuan (Ringkas)

| ID | Temuan | Severity | Status |
|---|---|---|---|
| **VOI-01** | `/ai/voice` tidak dipanggil frontend mana pun → dead code | 🟠 Tinggi | Akan diperbaiki (V4) |
| **VOI-02** | `VoiceInput.tsx` memakai Web Speech API → **tidak menyentuh kuota** | 🟠 Tinggi | Akan diperbaiki (V4) |
| **VOI-03** | `/user/voice` juga Web Speech API, bukan `/ai/voice` | 🟡 Sedang | Digabung ke chat (V5) |
| **VOI-04** | `VoiceInput` di-import tapi tidak dirender di halaman transaksi | 🟢 Rendah | Dibersihkan (V6) |
| **VOI-05** | Belum ada perekam audio nyata (`MediaRecorder`) | 🟠 Tinggi | Akan dibuat (V2) |
| **VOI-06** | Tidak ada UI yang menjelaskan mode/limit voice | 🟡 Sedang | Badge + hint (V4) |
| **VOI-07** | Tidak ada konfigurasi dinamis voice (durasi, bahasa) | 🟠 Tinggi | Setting baru (V1-BE) |
| **VOI-08** | Tidak ada limit harian voice (browser tak terbatas) | 🟠 Tinggi | Limit baru (V1-BE) |
| **VOI-09** | Tidak ada riwayat transkripsi server | 🟡 Sedang | Tabel baru (V3-BE) |
| **VOI-10** | Tidak ada VU meter | 🟡 Sedang | Web Audio (V2-FE) |

---

## 6. Catatan Metodologi

- Semua klaim kode disertai `file:line` dan diverifikasi pada 2026-10-15.
- Istilah:
  - **Mode Server** = transkripsi via `POST /ai/voice` (audio ke server, memakai kuota token).
  - **Mode Browser** = transkripsi via Web Speech API (audio tidak ke server; **dibatasi limit harian**).
  - **Fallback** = perpindahan otomatis Mode Server → Mode Browser.
- **Limit harian berlaku untuk kedua mode** (keputusan Q-03): baik server maupun browser
  dihitung sebagai "pemakaian voice hari itu". Detail di dok 03 §2.2.
- Dokumen 05 & 06 memakai checkbox `[ ]` untuk pelacakan eksekusi.

---

## 7. Ringkasan Perubahan Arsitektur (vs revisi 1)

| Aspek | Revisi 1 | Revisi 2 (kini) |
|---|---|---|
| Perubahan backend | **Tidak ada** | **Ada** (setting dinamis, limit harian, riwayat, endpoint config) |
| Limit voice | Hanya kuota token/request | **+ limit harian** per paket (kedua mode) |
| Durasi rekam | Konstanta 60s | **Setting dinamis** |
| Bahasa | Hardcode `id-ID` | **Dinamis** (`id_only` \| `multi`) |
| Riwayat | Lokal saja | **Server + lokal** |
| VU meter | Opsional | **Wajib** |
| Halaman voice | Tetap standalone | **Digabung ke chat** |
| Upgrade hint | Ya | Ya, **tidak memaksa** |

---

## 8. Status Implementasi & Bukti (2026-10-15)

> **Status: ✅ SELESAI.** Seluruh fase (BE-1..BE-2, FE-1..FE-4, Admin UI) terimplementasi
> dan terverifikasi.

### 8.1 Berkas baru/berubah

**Backend**
- `database/migrations/0025_voice_transcriptions.sql` (tabel riwayat)
- `database/migrations/0026_voice_daily_usage.sql` (counter harian append-only, R-14)
- `internal/voice/config.go` (resolusi config dinamis + `DayWindow`)
- `internal/voice/retention_worker.go` (prune retensi)
- `internal/domain/voice.go` (tipe + interface repo)
- `internal/repository/postgres/voice_repo.go` (+ test integrasi)
- `internal/delivery/http/handler/voice_handler.go` (config + history)
- `internal/admin/settings.go` (10 knob voice dinamis + snapshot)
- `internal/entitlement/catalog.go` (`voice_requests_per_day`)
- `internal/delivery/http/middleware/usage_resolvers.go` (resolver harian)
- `internal/delivery/http/router.go` (rute + gate), `cmd/server/main.go` (wiring)
- `internal/delivery/http/handler/ai_handler.go` (`/ai/voice` + language + riwayat)

**Frontend**
- `lib/voice-mode.ts`, `lib/voice-errors.ts`, `lib/voice-audio.ts` (+ test)
- `lib/voice-recorder.ts`, `lib/voice-level.ts` (+ test)
- `components/ai/VoiceLevelMeter.tsx`, `components/ai/VoiceHistoryPanel.tsx`
- `hooks/useVoiceConfig.ts`, `hooks/useVoiceMode.ts`, `hooks/useVoiceLanguage.ts`,
  `hooks/useVoiceHistory.ts`
- `components/ai/VoiceInput.tsx` (router cerdas + VU + fallback + badge)
- `components/super-admin/settings/VoiceSettingsTab.tsx` (panel admin)
- `app/(dashboard)/user/voice/` **dihapus** (digabung ke chat, Q-07)
- `components/super-admin/plans/PlanFormDialog.tsx` (dropdown katalog + fix sentinel -1)

### 8.2 Bukti test

| Suite | Hasil |
|---|---|
| Backend unit (`go test ./internal/... ./pkg/...`) | ✅ hijau |
| Backend integrasi (`voice_repo_test.go`, DB nyata) | ✅ 3 test lulus |
| Frontend unit + DOM (`vitest run`) | ✅ **338/338 lulus** |
| Build backend (`go build ./...`) | ✅ |
| Build frontend (`next build`) | ✅ |

### 8.3 Bukti live (API :8096, DB PostgreSQL, Redis)

| # | Aksi | Hasil |
|---|---|---|
| 1 | `GET /me/voice-config` (Pro) | durasi 60, `id_only`, limit harian 50, reset WIB, retensi 90 |
| 2 | `POST /me/voice-history` (browser) | 201; riwayat tersimpan; counter `used` 0→1 |
| 3 | `GET /me/entitlements` | `voice_requests_per_day {used:1,limit:50}` |
| 4 | Free (limit=1) rekam ke-2 | **402** `limit_exceeded`, `limit_key: voice_requests_per_day` |
| 5 | Isolasi: user B hapus riwayat user A | **404** (tidak bocor) |
| 6 | Admin ubah 5 knob via `PUT /admin/settings` | berlaku tanpa restart (durasi 60→90, `multi`, retensi 30, tz UTC, edit off) |
| 7 | Free → `POST /ai/voice` | **403** `feature_not_in_plan` (feature gate) |
| 8 | Kill switch `voice_server_enabled=false` | `server_enabled:false` di config |
| 9 | **R-14:** hapus semua riwayat lalu cek counter | `total=0` tetapi `used` **tetap 1** ✅ |

### 8.4 Temuan & perbaikan saat verifikasi (jujur, berbasis bukti)

| ID | Temuan | Tindakan |
|---|---|---|
| **BUG-1** | Limit harian dihitung dari **jumlah baris riwayat** → menghapus riwayat mengembalikan kuota hari itu (celah R-14). Terbukti live: `used` 1 → hapus → **0**. | **Diperbaiki**: tabel `voice_daily_usage` (append-only), counter di-*increment* terpisah dari riwayat. Terverifikasi: `used` tetap 1 setelah hapus. |
| **BUG-2** | Plan editor membulatkan limit dengan `Math.max(0, v)` → sentinel `-1` (unlimited) berubah jadi `0`. | **Diperbaiki**: pertahankan `-1`; input `min=-1`. |
| **BUG-3** | `recordHistory.mutateAsync(...).catch()` bisa melempar bila mutation mengembalikan non-promise. | **Diperbaiki**: dibungkus `Promise.resolve(...).catch()` + `try/catch`. |

> **Catatan metodologi:** bukti live dijalankan pada instance backend + PostgreSQL + Redis
> lokal (Docker Desktop tidak dapat dijalankan di environment ini, sehingga dipakai
> cluster PostgreSQL native + DB temp dengan skema dimuat dari migrasi kanonik).
> Migrasi 0025/0026 diverifikasi idempoten (`IF NOT EXISTS`) dan dijalankan dua kali.

