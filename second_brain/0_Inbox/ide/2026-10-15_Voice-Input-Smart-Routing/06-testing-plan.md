# 06 — Testing Plan

| Field | Isi |
|---|---|
| **Dokumen** | Rencana pengujian Voice Input Cerdas (backend + frontend + admin) |
| **Tanggal** | 2026-10-15 (revisi 2) |
| **Status** | Siap eksekusi |
| **Prinsip** | Uji logika murni dulu, lalu integrasi, lalu end-to-end; setiap mode, fallback, limit & config wajib terbukti |

> **Cara pakai:** tandai `[x]` saat lulus. Konvensi repo: `*.test.ts` = unit murni,
> `*.dom.test.tsx` = React/DOM, Go test untuk backend.

---

## 1. Piramida Pengujian

| Lapis | Alat | Cakupan |
|---|---|---|
| **Unit murni (FE)** | vitest (`*.test.ts`) | `voice-mode`, `voice-errors`, `voice-audio`, `voice-level` |
| **Unit DOM (FE)** | vitest + RTL (`*.dom.test.tsx`) | `useVoiceMode`, `useVoiceConfig`, `VoiceInput`, `VoiceLevelMeter` |
| **Unit (BE)** | Go test | repo `voice_repo`, handler, resolver, validasi setting |
| **Integrasi (BE)** | Go test + live | ledger, gate, limit harian, riwayat, `usage_changed` |
| **E2E / Manual** | Browser + live | izin mikrofon, HTTPS, lintas browser, admin config |

---

## 2. Backend Test

### 2.1 Setting dinamis (V1-BE)

| # | Skenario | Harapan |
|---|---|---|
| 1 | `PUT /admin/settings` `voice_max_duration_seconds=90` | 200; tersimpan |
| 2 | Nilai di luar 1..600 | 400 validasi |
| 3 | `voice_language_mode=multi` | 200; tersimpan |
| 4 | `voice_language_mode=xx` (tidak valid) | 400 |
| 5 | Non-super-admin | 403 |
| 6 | Perubahan berlaku ≤ TTL | `/me/voice-config` mengembalikan nilai baru |
| 7 | **`voice_history_retention_days=30`** | 200; tersimpan (N-01) |
| 8 | **`voice_daily_reset_timezone=UTC`** | 200; tersimpan (N-03) |
| 9 | **`voice_allow_edit_before_send=false`** | 200; tersimpan (N-04) |
| 10 | **`voice_server_enabled=false`** | Klien tidak memakai mode server (kill switch) |

### 2.1.1 Retensi riwayat (N-01)

| # | Skenario | Harapan |
|---|---|---|
| 1 | Riwayat lebih tua dari N hari | Terhapus oleh worker |
| 2 | Riwayat dalam N hari | Tetap ada |
| 3 | Ubah N via admin | Berlaku pada sweep berikutnya |
| 4 | Hari berjalan | Tidak terpengaruh (aman limit harian) |

### 2.1.2 Zona waktu reset (N-03)

| # | Skenario | Harapan |
|---|---|---|
| 1 | `Asia/Jakarta` | Hari berganti pada 00:00 WIB |
| 2 | `UTC` | Hari berganti pada 00:00 UTC |
| 3 | `daily_resets_at` di config | Benar sesuai zona |
| 4 | Konsistensi gate & tampilan | Memakai zona yang sama |

### 2.1.3 Bahasa per-rekaman (N-02)

| # | Skenario | Harapan |
|---|---|---|
| 1 | `id_only` | Tidak ada pilihan; pakai default |
| 2 | `multi` | Pilihan bahasa muncul |
| 3 | Pilih `en` saat rekam | Tersimpan di riwayat (`language=en`) |
| 4 | Bahasa diteruskan ke `/ai/voice` | Tercatat/terpakai |

### 2.2 Limit harian (V1-BE, V3-BE)

| # | Skenario | Harapan |
|---|---|---|
| 1 | Free, limit harian 1, rekam ke-1 | 200 |
| 2 | Free, rekam ke-2 hari itu | 402 `limit_exceeded` |
| 3 | Business (-1) | Tidak terbatas |
| 4 | Reset pada tengah malam **zona dinamis** (N-03) | Counter kembali 0 |
| 5 | `voice_requests_per_day` di `/me/entitlements` | Tampil dengan `used/limit/remaining` |
| 6 | Mode browser + mode server hari yang sama | Dihitung **bersama** |
| 7 | Hapus riwayat lalu rekam lagi | Counter **tidak** berkurang (append-only, R-14) |

> **Bukti live (2026-10-15):** Free limit=1 → rekam ke-2 `402 limit_exceeded`
> (`limit_key: voice_requests_per_day`). Pro limit=50 → `used` naik 0→1 pada
> `/me/voice-config` & `/me/entitlements`. **R-14:** setelah `DELETE /me/voice-history`
> (`total=0`), `used` **tetap 1** — celah bypass tertutup (tabel `voice_daily_usage`).

### 2.3 Riwayat (V2-BE)

| # | Skenario | Harapan |
|---|---|---|
| 1 | `POST /me/voice-history` (browser) | 201; baris tersimpan |
| 2 | `GET /me/voice-history` | Hanya milik user sendiri |
| 3 | User A baca/hapus riwayat user B | 404/403 (tidak bocor) |
| 4 | `DELETE /me/voice-history/:id` milik sendiri | 200 |
| 5 | `/ai/voice` sukses | Baris `mode=server` otomatis tercatat |
| 6 | `POST` transcript kosong | 400 |
| 7 | Rate limit `POST` | 429 setelah N |

### 2.4 Gate `/ai/voice` (V3-BE)

| # | Skenario | Harapan |
|---|---|---|
| 1 | Tanpa auth | 401 |
| 2 | Plan tanpa `voice_input` | 403 `feature_not_in_plan` |
| 3 | Kuota token habis | 402/429 |
| 4 | **Limit harian habis** | 402 `limit_exceeded` |
| 5 | Sukses | Ledger reserve+commit; riwayat; `usage_changed` |
| 6 | Gagal transkripsi | Ledger release |
| 7 | Idempotensi (`X-Request-Id` sama) | Tidak dobel-charge |

### 2.5 Config endpoint (V3-BE)

| # | Skenario | Harapan |
|---|---|---|
| 1 | `id_only` | `languages=[default_language]` |
| 2 | `multi` | `languages` >1 |
| 3 | `max_duration_seconds` | Sesuai setting |
| 4 | `daily_limit` | Sesuai plan + counter |
| 5 | `daily_reset_timezone` + `daily_resets_at` | Benar (N-03) |
| 6 | `history_retention_days` | Sesuai setting (N-01) |
| 7 | `allow_edit_before_send`, `vu_meter_enabled`, `fallback_to_browser` | Sesuai setting |
| 8 | `server_enabled=false` | Kill switch terlihat |
| 9 | Tidak ada secret di respons | ✅ |

**DoD Backend:**
- [ ] `go test ./...` hijau
- [ ] Bukti live: 1 voice server → ledger + riwayat + `usage_changed` + limit harian naik

---

## 3. Unit Test — Logika Murni (FE)

### 3.1 `lib/voice-mode.test.ts`

| # | `voice_input` | kuota | harian | recorder | STT | Harapan |
|---|---|---|---|---|---|---|
| 1 | ✅ | ✅ | ✅ | ✅ | — | `server` |
| 2 | ✅ | ✅ | ✅ | ❌ | ✅ | `browser` |
| 3 | ✅ | ❌ | ✅ | — | ✅ | `browser` |
| 4 | ❌ | — | ✅ | — | ✅ | `browser` |
| 5 | ✅ | ✅ | ✅ | ✅ | ✅ | `server` |
| 6 | ❌ | — | ✅ | — | ❌ | `unavailable` |
| 7 | ✅ | ✅ | ✅ | ❌ | ❌ | `unavailable` |
| 8 | ✅ | ❌ | ✅ | — | ❌ | `unavailable` |
| 9 | apa pun | apa pun | ❌ | — | — | **`daily_limit`** |

**DoD:**
- [ ] 9 kasus matriks tertutup
- [ ] Gate harian diperiksa **lebih dulu** (kasus 9 menang atas semua)

### 3.2 `lib/voice-errors.test.ts`

| # | Input | Harapan |
|---|---|---|
| 1 | 402 `limit_exceeded` | `fallback` |
| 2 | 403 `feature_not_in_plan` | `fallback` |
| 3 | 429 `insufficient_quota` | `fallback` |
| 4 | 429 `rate_limited` | `fallback` |
| 5 | 500 | `fallback` |
| 6 | network (tanpa response) | `fallback` |
| 7 | 400 `file_too_large` | `fatal` |
| 8 | **402 `daily_limit_exceeded`** | **`daily_limit`** |
| 9 | `Error("boom")` | `fallback` |

### 3.3 `lib/voice-audio.test.ts`
- [ ] Blob → base64 tanpa prefix `data:`
- [ ] Deteksi > `MAX_AUDIO_BASE64`

### 3.4 `lib/voice-level.test.ts`
- [ ] RMS 0 → level 0; RMS tinggi → level mendekati 1
- [ ] `stop()` membersihkan resource

---

## 4. Unit DOM Test (FE)

### 4.1 `hooks/useVoiceMode.dom.test.tsx`

| # | Skenario | Harapan |
|---|---|---|
| 1 | Pro + kuota + harian + recorder | `mode="server"`, `isPremium=true` |
| 2 | Free | `mode="browser"` |
| 3 | Kuota habis | `mode="browser"` |
| 4 | **Harian habis** | `mode="daily_limit"`, `canRecord=false` |
| 5 | Entitlement berubah | `mode` ikut berubah |
| 6 | SSR | Tidak error |

### 4.2 `hooks/useVoiceConfig.dom.test.tsx`
- [ ] Memuat config; refetch saat `usage_changed`
- [ ] Fallback default saat endpoint gagal

### 4.3 `VoiceInput.dom.test.tsx`

| # | Skenario | Harapan |
|---|---|---|
| 1 | Mode server sukses | `POST /ai/voice`; `onTranscript` teks |
| 2 | Mode server 402 | Fallback browser; tidak ada audio kedua terkirim |
| 3 | Mode server 403 | Fallback browser |
| 4 | Mode server 500 | Fallback browser |
| 5 | **402 daily_limit** | **Blokir** + hint (bukan fallback) |
| 6 | Fallback hanya 1x | Tidak loop |
| 7 | Mode browser | Tidak `POST /ai/voice`; **`POST /me/voice-history`** dipanggil |
| 8 | `unavailable` | Disabled + pesan |
| 9 | `daily_limit` | Disabled + hint upgrade dismissible |
| 10 | Izin ditolak | Toast izin; tidak fallback |
| 11 | Badge mode | "AI"/"Dasar" sesuai |
| 12 | **VU meter** | Tampil saat recording; hilang saat idle |
| 13 | Hint upgrade tidak memaksa | Bisa ditutup; tidak blokir UI |

### 4.4 `VoiceLevelMeter.dom.test.tsx`
- [ ] Render level; `active=false` → tidak menganalisis
- [ ] Unmount → resource bersih

### 4.5 `hooks/useVoiceLanguage.dom.test.tsx` (N-02)
- [ ] `id_only` → `canChoose=false`, `language=default`
- [ ] `multi` → `canChoose=true`, `options` benar
- [ ] `setLanguage("en")` → tersimpan & terpakai
- [ ] Pilihan diingat (localStorage)

### 4.6 Edit sebelum kirim (N-04) — bagian `VoiceInput`
- [ ] `allow_edit_before_send=true` → transcript masuk input (bisa diedit) sebelum kirim
- [ ] `allow_edit_before_send=false` → transcript langsung terkirim
- [ ] Riwayat menyimpan transcript **asli** (bukan editan)

**DoD:**
- [ ] Semua baris lulus
- [ ] `onTranscript` teks final saja
- [ ] Tidak ada `POST /ai/voice` di jalur browser

---

## 5. Integrasi Frontend

| # | Skenario | Harapan |
|---|---|---|
| 1 | Rekam server → kirim → transcript | Teks di chat; 1 request |
| 2 | Rekam browser → catat riwayat | Teks di chat; 1 request history |
| 3 | Kuota habis setelah `usage_changed` | Rekaman berikutnya → browser |
| 4 | Harian habis | Mode `daily_limit`; mic disabled |
| 5 | Race kuota habis antara cek & kirim | 402 → fallback |
| 6 | Race harian habis | 402 → blokir |
| 7 | Timeout jaringan | Fallback |
| 8 | Riwayat gagal dicatat | UX tetap jalan (best-effort) |

---

## 6. Admin UI Test

| # | Skenario | Harapan |
|---|---|---|
| 1 | Ubah durasi 60→90 | Tersimpan; klien dapat 90 |
| 2 | Ubah bahasa `id_only`→`multi` | Klien dapat pilihan bahasa |
| 3 | Ubah limit harian Pro 50→10 | Gate memakai 10 |
| 4 | Validasi durasi <1 / >600 | Error tampil |

---

## 7. Uji Manual / QA (Browser)

| # | Skenario | Chrome | Edge | Safari | Firefox |
|---|---|---|---|---|---|
| 1 | Mode server (Pro) | [ ] | [ ] | [ ] | [ ] |
| 2 | Mode browser (Free) | [ ] | [ ] | [ ] | ❌ (tak ada STT) |
| 3 | Fallback kuota habis | [ ] | [ ] | [ ] | [ ] |
| 4 | **Limit harian habis** | [ ] | [ ] | [ ] | [ ] |
| 5 | Izin mikrofon ditolak | [ ] | [ ] | [ ] | [ ] |
| 6 | **VU meter bergerak** saat bicara | [ ] | [ ] | [ ] | [ ] |
| 7 | HTTPS (produksi) | [ ] | [ ] | [ ] | [ ] |
| 8 | Bar kuota + sisa harian live | [ ] | [ ] | [ ] | [ ] |
| 9 | Halaman voice hilang, riwayat di chat | [ ] | [ ] | [ ] | [ ] |
| 10 | Bahasa `multi` → pemilih muncul | [ ] | [ ] | [ ] | [ ] |

---

## 8. Uji Edge Case & Negatif

| # | Kasus | Harapan |
|---|---|---|
| 1 | Rekaman > `maxDurationMs` | Auto-stop; tetap diproses |
| 2 | Audio > 5MB | Pesan ramah; tidak POST |
| 3 | Dua tab rekam bersamaan | Tidak crash; limit harian dicek backend |
| 4 | Logout saat merekam | Dihentikan |
| 5 | SSE mati | Fallback tetap; config via refetch |
| 6 | `MediaRecorder` tak didukung | Browser bila STT ada, else `unavailable` |
| 7 | `AudioContext` tak ada | VU meter non-aktif; tidak crash |
| 8 | Plan naik saat halaman terbuka | Mode naik ke server |
| 9 | Admin ubah durasi saat user merekam | Berlaku pada rekaman **berikutnya** |
| 10 | Riwayat dihapus semua | Limit harian **tidak** berubah retroaktif (dokumentasikan) |

---

## 9. Matriks Keterlacakan (Requirement → Test)

| Requirement | Dok | Test |
|---|---|---|
| Voice server berfungsi | 03 §1 | 4.3-1, 2.4-5 |
| Kuota berkurang real-time | 03 §9 | 5-1/2, 2.5-1 |
| **Durasi dinamis (Q-01)** | 03 §6.4 | 2.1-1/6, 6-1 |
| **Hint tidak memaksa (Q-02)** | 03 §4 | 4.3-13 |
| **Limit harian per paket (Q-03)** | 03 §2.2 | 2.2, 3.1-9, 4.3-5 |
| **Riwayat server (Q-04)** | 03 §6.5 | 2.3, 4.3-7 |
| **VU meter wajib (Q-05)** | 02 §6 | 4.3-12, 4.4, 7-6 |
| **Bahasa dinamis (Q-06)** | 03 §6.4 | 2.1-3, 6-2, 7-10 |
| **Gabung halaman (Q-07)** | 04 §5 | 7-9 |
| **Fallback wajib (Q-08)** | 03 §4 | 3.2, 4.3-2/3/4, 5-5/7 |
| **Retensi 90 hari dinamis (N-01)** | 02 §9 | 2.1-7, 2.1.1 |
| **Bahasa per-rekaman (N-02)** | 03 §7.1 | 2.1.3, 4.5 |
| **Zona reset dinamis (N-03)** | 02 §10 | 2.1-8, 2.1.2, 2.2-4 |
| **Edit sebelum kirim (N-04)** | 03 §6.4.1 | 2.1-9, 4.6 |
| **Semua dinamis (N-05)** | README §4.1 | 2.1, 6 (admin) |
| Isolasi data | — | 2.3-3 |

---

## 10. Kriteria Lulus Rilis (Go/No-Go)

- [ ] Semua unit & DOM test FE hijau (`pnpm test`)
- [ ] `pnpm build` sukses
- [ ] `go test ./...` hijau
- [ ] Bukti live: voice server → kuota + riwayat + limit harian naik real-time
- [ ] Bukti live: kuota habis → fallback browser (tanpa error mentah)
- [ ] Bukti live: limit harian habis → blokir + hint (tidak memaksa)
- [ ] Bukti live: admin ubah **durasi/bahasa/retensi/zona/perilaku** → berlaku tanpa redeploy
- [ ] Bukti live: bahasa `multi` → pilih per-rekaman → tersimpan di riwayat
- [ ] Bukti live: retensi worker menghapus riwayat lama sesuai setting
- [ ] Bukti live: kill switch `voice_server_enabled=false` → mode server tidak dipakai
- [ ] Tidak ada kebocoran track mikrofon / `AudioContext`
- [ ] Firefox menampilkan pesan jelas (bukan crash)
- [ ] Tidak ada link mati ke `/user/voice`
