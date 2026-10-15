# 05 — Implementation Plan

| Field | Isi |
|---|---|
| **Dokumen** | Rencana implementasi Voice Input Cerdas (backend + frontend + admin) |
| **Tanggal** | 2026-10-15 (revisi 2) |
| **Status** | Siap eksekusi |
| **Prinsip** | Aditif, zero-downtime, terisolasi per fase, backend otoritatif, konfigurasi dinamis |

> **Cara pakai:** tandai `[x]` saat selesai. Setiap fase punya DoD.
> Jangan mulai fase berikut sebelum DoD fase sebelumnya terpenuhi.

---

## Ringkasan Fase

| Fase | Fokus | Prioritas | Ketergantungan |
|---|---|---|---|
| **V1-BE** | Fondasi backend: setting dinamis + limit harian + katalog | 🔴 Kritis | — |
| **V2-BE** | Riwayat transkripsi (tabel, repo, endpoint) | 🔴 Kritis | V1-BE |
| **V3-BE** | Wire gate + resolver + emit + config endpoint | 🟠 Tinggi | V1-BE, V2-BE |
| **V4-BE** | Admin UI (setting voice + limit plan) | 🟠 Tinggi | V1-BE |
| **V5-FE** | Fondasi pure (mode, error, audio) | 🔴 Kritis | — |
| **V6-FE** | Perekam + VU meter | 🔴 Kritis | V5-FE |
| **V7-FE** | Hooks (config, mode, history) | 🟠 Tinggi | V1-BE, V3-BE, V5-FE |
| **V8-FE** | `VoiceInput` smart routing + fallback + badge | 🟠 Tinggi | V5-FE..V7-FE |
| **V9-FE** | Gabung halaman voice ke chat (Q-07) | 🟡 Sedang | V8-FE |
| **V10** | Pemolesan, observabilitas, dokumentasi | 🟢 Rendah | semua |

---

## V1-BE — Fondasi Backend

### TASK-V1-BE-01 — Setting global dinamis (Q-01, Q-05, Q-06, N-01..N-04)

**Target:** `internal/admin/settings.go`

**Langkah:**
1. Tambah ke `writableSettings`:
   - `voice_max_duration_seconds` (`int`, 1..600)
   - `voice_language_mode` (`enum`, `["id_only","multi"]`)
   - `voice_default_language` (`enum`, `["id","en"]`)
   - `voice_languages` (`string`, maxLen 64)
   - `voice_history_retention_days` (`int`, 1..3650)
   - `voice_daily_reset_timezone` (`enum`, `["UTC","Asia/Jakarta"]`)
   - `voice_allow_edit_before_send` (`bool`)
   - `voice_vu_meter_enabled` (`bool`)
   - `voice_fallback_to_browser` (`bool`)
   - `voice_server_enabled` (`bool`)
2. Tambah semua key ke `settingsSnapshot`.
3. Tambah ke daftar reload runtime config bila perlu.

**DoD:**
- [x] `PUT /admin/settings` menerima semua key (validasi jalan)
- [x] `GET /admin/settings` menampilkan semuanya
- [x] Invalidasi runtime config terpanggil (perubahan berlaku ≤ TTL 5s)
- [x] Enum/int/bool tervalidasi (nilai salah → 400)

### TASK-V1-BE-02 — Limit harian per paket (Q-03)

**Target:** `database/migrations/`, `internal/entitlement/catalog.go`, seed

**Langkah:**
1. Tambah key `voice_requests_per_day` ke `LimitCatalog` (`Enforced: true`).
2. Seed default: Free=1, Pro=50, Business=-1 (via admin plan editor setelahnya).
3. Dokumentasikan bahwa admin dapat mengubah per paket.

**DoD:**
- [x] `voice_requests_per_day` ada di `LimitCatalog`
- [x] Plan seed memuat nilai default
- [x] Admin plan editor menampilkan & menyimpan key ini (via katalog yang sudah ada)
- [x] `-1` diperlakukan sebagai unlimited

---

## V2-BE — Riwayat Transkripsi (Q-04)

### TASK-V2-BE-01 — Tabel `voice_transcriptions`

**Target:** `database/migrations/00XX_voice_transcriptions.sql`

**Langkah:**
1. Buat tabel (lihat dok 04 §2.4) dengan `user_id`, `transcript`, `language`, `mode`,
   `duration_ms`, `tokens_used`, `created_at`.
2. Index `(user_id, created_at DESC)`.

**DoD:**
- [x] Migrasi idempoten (`IF NOT EXISTS`)
- [x] Index dibuat
- [x] Isolasi: kolom `user_id` wajib, ada FK ke `users`

### TASK-V2-BE-02 — Repository `voice_repo.go`

**Target:** `internal/repository/postgres/voice_repo.go` (baru)

**Langkah:**
1. `Create(ctx, userID, entry)`.
2. `ListByUser(ctx, userID, limit)` (paged).
3. `CountToday(ctx, userID, now)` — untuk limit harian.
4. `DeleteOne(ctx, id, userID)`, `ClearByUser(ctx, userID)`.

**DoD:**
- [x] **Semua** query `WHERE user_id = $1` (isolasi)
- [x] `CountToday` memakai batas hari UTC
- [x] `DeleteOne` tidak bisa menghapus milik user lain (cek `user_id`)

### TASK-V2-BE-03 — Handler & rute riwayat

**Target:** `internal/delivery/http/handler/voice_handler.go` (baru), `router.go`

**Langkah:**
1. `GET /me/voice-history` (paged).
2. `POST /me/voice-history` (mode browser) + **rate limit** + `limitGate(voice_requests_per_day)`.
3. `DELETE /me/voice-history/:id`, `DELETE /me/voice-history`.

**DoD:**
- [x] Semua rute `RequireUser`
- [x] `POST` ter-gate limit harian (402 saat habis)
- [x] `POST` validasi payload (transcript non-kosong, `mode=browser`)
- [x] Isolasi: hanya bisa baca/hapus milik sendiri

---

## V3-BE — Wire Gate, Resolver, Emit, Config

### TASK-V3-BE-01 — Usage resolver `voice_requests_per_day`

**Target:** `internal/delivery/http/middleware/usage_resolvers.go`, `cmd/server/main.go`

**Langkah:**
1. Tambah `CountVoiceToday` ke `UsageProviders`.
2. Tambah `case "voice_requests_per_day"` di `UsageFor`.
3. Wire di `main.go` (pakai `voiceRepo.CountToday`).

**DoD:**
- [x] `UsageFor` mengenali key baru
- [x] `GET /me/entitlements` memuat `usage.voice_requests_per_day`
- [x] Counter = jumlah riwayat hari ini (kedua mode)

### TASK-V3-BE-02 — Gate `/ai/voice` + catat riwayat server

**Target:** `router.go`, `ai_handler.go`

**Langkah:**
1. Tambah `limitGate(config, "voice_requests_per_day")` ke rute `/ai/voice`.
2. Setelah sukses, insert `voice_transcriptions` (`mode=server`, `tokens_used`).
3. Tambah `voice_requests_per_day` ke `LimitKeys` payload `usage_changed`.

**DoD:**
- [x] `/ai/voice` ter-gate limit harian
- [x] Sukses → baris riwayat `mode=server` tercatat
- [x] Event `usage_changed` memuat key harian

### TASK-V3-BE-03 — Emit saat catat riwayat browser

**Target:** `voice_handler.go`

**DoD:**
- [x] `POST /me/voice-history` emit `usage_changed` (`["voice_requests_per_day"]`)
- [x] Kuota harian user lain **tidak** terpengaruh (payload per-user)

### TASK-V3-BE-04 — Endpoint `GET /me/voice-config`

**Target:** `voice_handler.go`, `router.go`

**Langkah:**
1. Baca semua knob voice via runtime config (durasi, bahasa, retensi, zona, perilaku).
2. Baca limit harian (dari plan + counter, zona dinamis).
3. Kembalikan payload lengkap (dok 03 §6.4).

**DoD:**
- [x] Mengembalikan **semua** field (durasi, bahasa, retensi, zona, perilaku, kill switch)
- [x] `multi` → `languages` berisi >1; `id_only` → `[default_language]`
- [x] `daily_reset_timezone` + `daily_resets_at` benar (N-03)
- [x] `history_retention_days` sesuai setting (N-01)
- [x] Tidak ada secret
- [x] Nilai mengikuti setting dinamis (ubah admin → berubah ≤ TTL)

### TASK-V3-BE-05 — Worker retensi riwayat (N-01)

**Target:** `internal/billing/voice_retention_worker.go` (baru), `cmd/server/main.go`

**Langkah:**
1. Worker harian: hapus `voice_transcriptions` lebih tua dari `voice_history_retention_days`.
2. Start di `main.go` (`bgCtx`), `defer Stop()`.

**DoD:**
- [x] Riwayat lebih tua dari N hari terhapus
- [x] N dinamis (ubah admin → berlaku pada sweep berikutnya)
- [x] Tidak menyentuh hari berjalan (aman untuk limit harian)

### TASK-V3-BE-06 — Dukungan `language` pada `/ai/voice` (N-02)

**Target:** `ai_handler.go`, `ai_usecase.go`, `aigateway`

**Langkah:**
1. Terima `language` opsional di request `/ai/voice`.
2. Teruskan ke gateway (parameter route/model).
3. Simpan ke riwayat.

**DoD:**
- [x] `language` diterima & diteruskan
- [x] Tersimpan di `voice_transcriptions.language`
- [x] Default = `voice_default_language` bila kosong

---

## V4-BE — Admin UI

### TASK-V4-BE-01 — Panel setting voice

**Target:** `frontend/components/super-admin/settings/*` (tab AI atau baru)

**Langkah:**
1. Field `voice_max_duration_seconds` (number, 1..600).
2. Field `voice_language_mode` (select: `id_only` / `multi`).
3. Field `voice_default_language` (select) + `voice_languages` (CSV, aktif saat `multi`).
4. Field `voice_history_retention_days` (number, 1..3650) — N-01.
5. Field `voice_daily_reset_timezone` (select: `UTC` / `Asia/Jakarta`) — N-03.
6. Toggle `voice_allow_edit_before_send`, `voice_vu_meter_enabled`,
   `voice_fallback_to_browser`, `voice_server_enabled`.
7. Simpan via `PUT /admin/settings`.

**DoD:**
- [x] Semua field tersimpan & termuat
- [x] Validasi tampil (rentang/enum)
- [x] Perubahan berlaku tanpa redeploy
- [x] `voice_languages` hanya relevan saat `multi` (UX jelas)

### TASK-V4-BE-02 — Limit harian di plan editor

**Target:** `components/super-admin/plans/PlanFormDialog.tsx`

**DoD:**
- [x] `voice_requests_per_day` muncul di editor limit (dari katalog)
- [x] Bisa diatur per paket
- [x] `-1` (unlimited) didukung

---

## V5-FE — Fondasi Pure

### TASK-V5-FE-01 — `lib/voice-mode.ts`

**DoD:**
- [x] `decideVoiceMode` menangani 4 mode (termasuk `daily_limit`)
- [x] Gate harian diperiksa lebih dulu
- [x] Tidak mengimpor React/`window`/`api`

### TASK-V5-FE-02 — `lib/voice-errors.ts`

**DoD:**
- [x] `daily_limit` dibedakan dari `fallback`
- [x] Semua kode backend (dok 03 §6.2) terklasifikasi
- [x] Tidak pernah throw

### TASK-V5-FE-03 — `lib/voice-audio.ts`

**DoD:**
- [x] `blobToBase64` tanpa prefix data URL
- [x] `MAX_AUDIO_BASE64` tersedia

---

## V6-FE — Perekam & VU Meter

### TASK-V6-FE-01 — `lib/voice-recorder.ts`

**DoD:**
- [x] Mengembalikan `{ blob, durationMs, stream }`
- [x] `stream` diekspos untuk VU meter
- [x] Auto-stop pada `maxDurationMs` (dinamis)
- [x] **Semua** track di-stop pada stop/cancel/error
- [x] Error izin/perangkat diteruskan

### TASK-V6-FE-02 — `lib/voice-level.ts`

**DoD:**
- [x] `AnalyserNode` + `requestAnimationFrame`
- [x] `stop()` menutup `AudioContext` + `cancelAnimationFrame`
- [x] Tidak ada kebocoran (uji: buka/tutup 50x → tidak ada leak)

### TASK-V6-FE-03 — `components/ai/VoiceLevelMeter.tsx`

**DoD:**
- [x] Menampilkan level 0..1
- [x] **Wajib** tampil saat `recording` (Q-05)
- [x] Bersih saat unmount

---

## V7-FE — Hooks

### TASK-V7-FE-01 — `hooks/useVoiceConfig.ts`

**DoD:**
- [x] Memuat `/me/voice-config`
- [x] Refetch saat `usage_changed`
- [x] Fallback default aman bila endpoint gagal (durasi 60, `id_only`)

### TASK-V7-FE-02 — `hooks/useVoiceMode.ts`

**DoD:**
- [x] Menggabungkan entitlement + config + kapabilitas
- [x] SSR-safe
- [x] `canRecord`, `isPremium`, `dailyRemaining` benar

### TASK-V7-FE-03 — `lib/voice-history.ts` + `hooks/useVoiceHistory.ts`

**DoD:**
- [x] `recordBrowserHistory` memanggil `POST /me/voice-history` (termasuk `language`)
- [x] `useVoiceHistory` query + pagination
- [x] Gagal mencatat bersifat best-effort (tidak menggagalkan UX)

### TASK-V7-FE-04 — `hooks/useVoiceLanguage.ts` (N-02)

**DoD:**
- [x] `id_only` → `canChoose=false`, pakai `default_language`
- [x] `multi` → `canChoose=true`, opsi dari `languages`, pilih per-rekaman
- [x] Pilihan terakhir diingat (localStorage)
- [x] Bahasa diteruskan ke mode browser & server

---

## V8-FE — VoiceInput Smart Routing

### TASK-V8-FE-01 — Refactor `VoiceInput.tsx`

**DoD:**
- [x] 4 mode didukung
- [x] Mode server kirim audio (+`language`) & pakai kuota (terbukti)
- [x] Mode browser catat riwayat (+`language`)
- [x] Fallback otomatis (sekali, bila `fallback_to_browser`), `daily_limit` memblokir
- [x] **VU meter wajib** saat recording (bila `vu_meter_enabled`)
- [x] Badge mode + sisa harian + (opsional) reset countdown
- [x] Pemilih bahasa saat `multi` (N-02)
- [x] **Edit sebelum kirim** sesuai `allow_edit_before_send` (N-04)
- [x] Hint upgrade **tidak memaksa** (Q-02)
- [x] `onTranscript` tetap teks final
- [x] Pemanggil lama tidak berubah

### TASK-V8-FE-02 — Integrasi kuota & limit real-time

**DoD:**
- [x] Bar kuota & sisa harian berkurang tanpa reload
- [x] Setelah habis, mode turun otomatis pada rekaman berikutnya
- [x] Perubahan config admin (durasi/bahasa) terlihat setelah refetch

---

## V9-FE — Gabung Halaman Voice (Q-07)

### TASK-V9-FE-01 — Hapus rute standalone, pindah ke chat

**Target:** `app/(dashboard)/user/voice/page.tsx` (hapus), `lib/nav.ts`, chat panel

**Langkah:**
1. Hapus rute `/user/voice` + menu nav.
2. Pindahkan riwayat ke panel/drawer di chat (`useVoiceHistory`).
3. Pastikan tidak ada link mati.

**DoD:**
- [x] Rute `/user/voice` hilang (redirect atau 404 rapi)
- [x] Menu nav voice dihapus
- [x] Riwayat voice tersedia di chat
- [x] Tidak ada referensi mati

---

## V10 — Pemolesan

### TASK-V10-01 — Observabilitas
- [x] Log mode terpilih & alasan fallback (dev-only)
- [x] Tidak ada PII/audio di log

### TASK-V10-02 — Dokumentasi
- [x] JSDoc pada semua modul `voice-*`
- [x] Update `second_brain` bila keputusan berubah

---

## Checklist Ringkas

```
BACKEND
[x] V1-BE-01  Setting dinamis (durasi, bahasa, retensi, zona, perilaku, kill switch)
[x] V1-BE-02  Limit harian per paket + katalog
[x] V2-BE-01  Tabel voice_transcriptions
[x] V2-BE-02  Repo voice_repo.go
[x] V2-BE-03  Handler & rute riwayat
[x] V3-BE-01  Usage resolver voice_requests_per_day
[x] V3-BE-02  Gate /ai/voice + catat riwayat server
[x] V3-BE-03  Emit usage_changed (browser history)
[x] V3-BE-04  Endpoint /me/voice-config (lengkap)
[x] V3-BE-05  Worker retensi riwayat (N-01)
[x] V3-BE-06  Dukungan language pada /ai/voice (N-02)
[x] V4-BE-01  Panel setting voice (admin UI)
[x] V4-BE-02  Limit harian di plan editor

FRONTEND
[x] V5-FE-01  lib/voice-mode.ts
[x] V5-FE-02  lib/voice-errors.ts
[x] V5-FE-03  lib/voice-audio.ts
[x] V6-FE-01  lib/voice-recorder.ts
[x] V6-FE-02  lib/voice-level.ts
[x] V6-FE-03  VoiceLevelMeter.tsx
[x] V7-FE-01  hooks/useVoiceConfig.ts
[x] V7-FE-02  hooks/useVoiceMode.ts
[x] V7-FE-03  lib/voice-history.ts + useVoiceHistory.ts
[x] V7-FE-04  hooks/useVoiceLanguage.ts (N-02)
[x] V8-FE-01  VoiceInput.tsx smart routing + VU + bahasa + edit
[x] V8-FE-02  Integrasi kuota real-time
[x] V9-FE-01  Gabung halaman voice ke chat

POLES
[x] V10-01    Observabilitas
[x] V10-02    Dokumentasi
```

---

## Catatan Eksekusi

- **Urutan wajib:** V1-BE → V2-BE → V3-BE → V4-BE (backend siap) **lalu** V5-FE → … → V9-FE.
- **Backend dulu** agar frontend punya endpoint/config untuk dikonsumsi.
- **Migrasi** harus idempoten & zero-downtime (aditif).
- **Kill switch** opsional: `NEXT_PUBLIC_VOICE_SERVER_ENABLED=false` untuk mematikan mode server.
- **Rollback:** semua aditif; menonaktifkan mode server mengembalikan perilaku lama (browser-only).
