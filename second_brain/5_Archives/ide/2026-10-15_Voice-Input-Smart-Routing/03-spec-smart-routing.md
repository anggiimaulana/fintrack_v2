# 03 — Spesifikasi Smart Routing

| Field | Isi |
|---|---|
| **Dokumen** | Spesifikasi perilaku: matriks keputusan, state machine, kontrak, payload |
| **Tanggal** | 2026-10-15 |
| **Status** | Spec selesai — belum dieksekusi |
| **Prinsip** | Fail-safe, transparan, backend otoritatif |

---

## 1. Definisi Mode

| Mode | Kode | Transkripsi | Audio ke server | Kuota |
|---|---|---|---|---|
| **Mode Server** | `server` | AI Gateway (`/ai/voice`) | Ya | Token + request + **harian** |
| **Mode Browser** | `browser` | Web Speech API | Tidak | **Harian saja** |
| **Mode Limit Harian** | `daily_limit` | — | — | Limit harian habis → blokir |
| **Mode Tidak Tersedia** | `unavailable` | — | — | — |

---

## 2. Matriks Keputusan

Klien memilih mode saat **mic ditekan** (bukan saat mount), memakai data entitlement terbaru
dari `useEntitlements()` **dan** config global dari `GET /me/voice-config`.

**Prasyarat (gate harian):** bila `voice_requests_per_day` sudah habis → **semua mode
transkripsi diblokir** (baik server maupun browser), tampilkan pesan + hint upgrade.

| # | `voice_input` | Kuota token/request | Limit harian tersisa | `MediaRecorder` | `SpeechRecognition` | Mode Terpilih |
|---|---|---|---|---|---|---|
| 1 | ✅ | ✅ cukup | ✅ | ✅ | — | **`server`** |
| 2 | ✅ | ✅ cukup | ✅ | ❌ | ✅ | **`browser`** (+ hint) |
| 3 | ✅ | ❌ habis | ✅ | — | ✅ | **`browser`** (+ hint kuota) |
| 4 | ❌ (plan) | — | ✅ | — | ✅ | **`browser`** |
| 5 | ✅ | ✅ cukup | ✅ | ✅ | ✅ | **`server`** (prioritas) |
| 6 | ❌ | — | ✅ | — | ❌ | **`unavailable`** (pesan jelas) |
| 7 | ✅ | ✅ cukup | ✅ | ❌ | ❌ | **`unavailable`** |
| 8 | ✅ | ❌ habis | ✅ | — | ❌ | **`unavailable`** |
| 9 | apa pun | apa pun | ❌ **habis** | — | — | **`daily_limit`** (blokir + hint upgrade) |

> **Aturan prioritas:** `server` > `browser` > `unavailable`, **kecuali** limit harian habis
> → `daily_limit` (semua jalur diblokir sampai reset harian).

### 2.1 Definisi "kuota cukup"

```ts
function hasVoiceQuota(ent: Entitlements): boolean {
  const tokens = ent.usage?.ai_tokens_per_cycle;
  const reqs   = ent.usage?.ai_requests_per_cycle;
  const tokensOk = !tokens || tokens.unlimited || tokens.remaining >= 1;
  const reqsOk   = !reqs   || reqs.unlimited   || reqs.remaining   >= 1;
  return tokensOk && reqsOk;
}
```

### 2.2 Definisi "limit harian tersisa" (Q-03)

```ts
function hasDailyVoiceQuota(ent: Entitlements): boolean {
  const daily = ent.usage?.voice_requests_per_day;
  if (!daily) return true;                 // tidak dilacak → izinkan (fail-open utk UX)
  if (daily.unlimited) return true;
  return daily.remaining >= 1;
}
```

> **Catatan:** `voice_requests_per_day` **berlaku untuk kedua mode** (server & browser).
> Backend otoritatif; klien hanya estimasi optimistis. Bila backend menolak (race),
> klien fallback/berhenti dengan pesan jelas.
>
> **Definisi hari (N-03):** hari kalender mengikuti **zona waktu dinamis**
> (`voice_daily_reset_timezone`, usulan default `Asia/Jakarta`). Reset pada tengah malam
> zona tersebut. Nilai zona dikirim ke klien via `/me/voice-config`
> (`daily_reset_timezone` + `daily_resets_at`) agar UI bisa menampilkan "reset dalam X jam".
>
> **Anti-cheat (append-only):** hitungan harian dihitung dari `created_at` baris riwayat,
> **bukan** "jumlah baris saat ini" — sehingga menghapus riwayat **tidak** mengurangi counter
> (lihat dok 07 R-14).

---

## 3. Alur Keputusan (Happy Path & Fallback)

```
                 ┌──────────────────────────────┐
                 │ User menekan tombol mic      │
                 └──────────────┬───────────────┘
                                ▼
                 ┌──────────────────────────────┐
                 │ Baca entitlement (cache/SSE) │
                 └──────────────┬───────────────┘
                                ▼
             ┌──────────────────────────────────────┐
             │ voice_input && kuota cukup &&        │
             │ MediaRecorder tersedia ?             │
             └───────┬──────────────────────┬───────┘
                  Ya │                      │ Tidak
                     ▼                      ▼
        ┌────────────────────┐   ┌────────────────────────┐
        │ MODE SERVER        │   │ SpeechRecognition ada? │
        │ (MediaRecorder)    │   └───────┬────────┬───────┘
        └─────────┬──────────┘        Ya │        │ Tidak
                  │                      ▼        ▼
                  │          ┌──────────────┐  ┌───────────────┐
                  │          │ MODE BROWSER │  │ UNAVAILABLE   │
                  │          └──────┬───────┘  │ (pesan jelas) │
                  │                 │          └───────────────┘
                  ▼                 ▼
        ┌──────────────────┐  ┌──────────────┐
        │ Rekam audio      │  │ Rekam suara  │
        │ → POST /ai/voice │  │ (vendor)     │
        └────────┬─────────┘  └──────┬───────┘
                 │                   │
       ┌─────────┴──────────┐        │
       │ Respons?           │        │
       ├─ 200 transcript ───┼────────┤
       ├─ 402 kuota habis ──┼──► FALLBACK ke MODE BROWSER (otomatis)
       ├─ 403 fitur ────────┼──► FALLBACK ke MODE BROWSER (otomatis)
       ├─ 429 rate limit ───┼──► FALLBACK ke MODE BROWSER (otomatis)
       └─ 5xx / network ────┼──► FALLBACK ke MODE BROWSER (otomatis)
                            │
                            ▼
                 ┌──────────────────────┐
                 │ onTranscript(text)   │  ← SAMA untuk semua mode
                 └──────────────────────┘
```

---

## 4. Aturan Fallback (Wajib)

| Pemicu | Aksi | Pesan ke user |
|---|---|---|
| `402 limit_exceeded` | Fallback → `browser`, mulai ulang rekam | "Kuota suara habis — beralih ke mode dasar." (toast) |
| `429 insufficient_quota` | Fallback → `browser` | "Kuota suara habis — beralih ke mode dasar." |
| `403 feature_not_in_plan` | Fallback → `browser` | "Fitur suara premium tidak tersedia di paket Anda — memakai mode dasar." |
| `429 rate_limited` | Fallback → `browser` | "Terlalu banyak permintaan — memakai mode dasar." |
| `5xx` / network error | Fallback → `browser` | "Layanan suara sedang bermasalah — memakai mode dasar." |
| Limit harian habis (pra-cek) | **Blokir** semua mode | "Batas suara harian tercapai. Reset besok." + hint upgrade (**dismissible**) |
| Limit harian habis (backend) | **Blokir** setelah respons | "Batas suara harian tercapai." + hint upgrade |
| `browser` juga tidak tersedia | Tampilkan `unavailable` | "Browser Anda tidak mendukung input suara. Gunakan Chrome/Edge." |
| Mikrofon ditolak (`NotAllowedError`) | Hentikan, **jangan** fallback | "Izin mikrofon ditolak. Aktifkan di pengaturan browser." |

**Aturan penting:**
- Fallback **hanya sekali** per sesi rekam (mencegah loop).
- Setelah fallback, set flag sesi `mode = browser` agar tidak mencoba server lagi
  pada rekaman berikutnya **sampai entitlement diperbarui**.
- **Fallback tidak boleh kehilangan data**: teks dari mode browser tetap dikirim ke
  tujuan yang sama (`onTranscript`).
- **Hint upgrade TIDAK memaksa** (Q-02): tampilkan sebagai toast/panel yang bisa
  ditutup; jangan blokir interaksi lain. Tidak ada modal blocking.

### 4.1 Perbedaan: Fallback vs Blokir

| Situasi | Perilaku | Kenapa |
|---|---|---|
| Kuota token habis | **Fallback** ke browser | Voice tetap bisa dipakai (UX) |
| Fitur tidak di plan | **Fallback** ke browser | Voice tetap bisa dipakai (UX) |
| **Limit harian habis** | **Blokir** | Batas keras harian (Q-03) |
| Mikrofon ditolak | **Berhenti** | Tidak ada jalur yang bisa dipakai |

---

## 5. State Machine (per Sesi Rekam)

```
      ┌─────────┐
      │  idle   │◄────────────────────────────┐
      └────┬────┘                             │
           │ tekan mic                       │ selesai / batal / error
           ▼                                 │
   ┌────────────────┐                        │
   │  selecting     │  (baca entitlement+cfg)│
   └───────┬────────┘                        │
           │ pilih mode                      │
           ├──────────────► recording_server ─┤
           ├──────────────► recording_browser─┤
           ├──────────────► daily_limit ──────┘ (toast + hint, kembali idle)
           └──────────────► unavailable ──────┘ (toast, kembali idle)

   recording_server ──kirim──► transcribing ──┬─► success → idle
                                              └─► fallback ─► recording_browser

   recording_browser ──onresult──► success → idle
                     ──onerror───► error   → idle
```

**State yang diekspos ke UI:**

```ts
type VoiceState =
  | "idle"
  | "recording"        // sedang merekam (mode apa pun) — VU meter WAJIB tampil
  | "transcribing"     // khusus mode server: menunggu respons
  | "error";           // error yang tidak bisa di-fallback
```

> **Catatan:** `daily_limit` dan `unavailable` **bukan** state sesi (tidak ada rekaman);
> keduanya ditangani sebagai keputusan mode + toast, lalu kembali ke `idle`.

---

## 5.1 VU Meter — Bagian dari State `recording` (Q-05)

Selama `state === "recording"`, **wajib** menampilkan level mikrofon (dok 02 §6).
Komponen `VoiceLevelMeter` menerima `stream` dari perekam aktif dan `active={state==="recording"}`.

---

## 6. Kontrak API

### 6.1 Request — `POST /api/v1/ai/voice`

**Sudah ada; tidak berubah.**

```jsonc
// Request body
{
  "audio": "<base64 audio/webm; opus>"   // wajib, ≤ 5MB setelah decode
}
```

**Headers:**
- `X-Request-Id` (opsional) — untuk idempotensi reserve ledger.
- Cookie `ft_at` (auth) — dikirim otomatis (`withCredentials`).

### 6.2 Response

| Status | Body | Arti |
|---|---|---|
| `200` | `{ success: true, data: { success: true, transcript: "..." } }` | Sukses |
| `400` | `{ error: "file_too_large" }` | Audio > 5MB |
| `400` | `{ error: "validation_error" }` | Body tidak valid |
| `402` | `{ error: "limit_exceeded", limit_key, limit_value, current_usage }` | Limit request/token habis (dari `WithinLimit`) |
| `403` | `{ error: "feature_not_in_plan", feature_key }` | Plan tanpa `voice_input` (dari `RequireFeature`) |
| `429` | `{ error: "insufficient_quota", upgrade_hint: true }` | Kuota ledger tidak cukup saat reserve |
| `429` | `{ error: "rate_limited" }` | Rate limit |
| `500` | `{ error: "..." }` | Gangguan server |

> **Catatan kontrak:** klien **hanya** butuh membedakan "sukses" vs "fallback-able"
> vs "fatal". Semua `402`/`403`/`429`/`5xx` = fallback-able. `400` = fatal (bug klien).
>
> **Kode error yang harus dikenali klien (fallback):**
> `insufficient_quota`, `limit_exceeded`, `feature_not_in_plan`, `rate_limited`.

### 6.3 Idempotensi

- Klien **wajib** mengirim `X-Request-Id` unik per rekaman (mis. `crypto.randomUUID()`).
- Bila retry karena timeout, `X-Request-Id` **sama** → ledger tidak dobel-charge
  (reserve idempoten lewat `voice-reserve:<reqID>`).

### 6.4 Endpoint Konfigurasi Voice — `GET /api/v1/me/voice-config`

**Baru.** Mengembalikan konfigurasi voice efektif untuk user (semua **dinamis**).
Tidak mengandung secret. Dipakai klien untuk memutuskan mode & mengatur UI.

```jsonc
// 200
{
  "success": true,
  "data": {
    // ── Durasi (Q-01) ──
    "max_duration_seconds": 60,          // voice_max_duration_seconds
    // ── Bahasa (Q-06, N-02) ──
    "language_mode": "multi",            // "id_only" | "multi"
    "default_language": "id",            // voice_default_language
    "languages": ["id", "en"],           // voice_languages (saat multi)
    "language_per_recording": true,      // N-02: pilih per-rekaman
    // ── Limit harian (Q-03) ──
    "daily_limit": {
      "limit": 50,
      "used": 3,
      "remaining": 47,
      "unlimited": false
    },
    // ── Zona reset (N-03) ──
    "daily_reset_timezone": "Asia/Jakarta",  // voice_daily_reset_timezone
    "daily_resets_at": "2026-10-16T00:00:00+07:00", // waktu reset berikutnya
    // ── Retensi (N-01) ──
    "history_retention_days": 90,        // voice_history_retention_days
    // ── Perilaku (N-04, Q-05, Q-08) ──
    "allow_edit_before_send": true,      // voice_allow_edit_before_send
    "vu_meter_enabled": true,            // voice_vu_meter_enabled
    "fallback_to_browser": true,         // voice_fallback_to_browser
    // ── Ketersediaan ──
    "server_enabled": true,              // voice_server_enabled (kill switch)
    "server_available": true             // features.voice_input (info cepat)
  }
}
```

> **Catatan:** `daily_limit` **duplikat** info dari `/me/entitlements`
> (`usage.voice_requests_per_day`) agar klien cukup satu panggilan untuk setup voice.
> Bila `voice_requests_per_day` tidak ada di plan → `unlimited: true`.
>
> **Semua field di atas dinamis.** Super Admin ubah → berlaku ≤ TTL resolver (5 detik).

### 6.4.1 Klarifikasi N-04 — Edit Sebelum Kirim (dan "kenapa bukan langsung ke `/ai/voice`")

Poin penting: **transkripsi** dan **pengiriman ke chat** adalah **dua langkah terpisah**.

```
Rekam ──► [Transkripsi] ──► transcript ──► [opsional: EDIT] ──► kirim ke chat
              │                                            │
        server: /ai/voice                          selalu POST /ai/chat
        browser: Web Speech API                    (bukan /ai/voice)
```

- **`/ai/voice` hanya melakukan transkripsi** (audio → teks). Ia **tidak** mengirim pesan
  ke chat dan **tidak** memanggil LLM.
- Setelah transcript diperoleh (dari mode apa pun), teks dikirim ke chat lewat
  `POST /ai/chat` seperti pesan teks biasa.
- Jadi memang **tidak ada** "langsung gas ke `/ai/voice` sampai jadi jawaban chat" —
  itu **bukan** desainnya, dan **memang seharusnya begitu** (pemisahan tanggung jawab).

**Implikasi N-04 (`voice_allow_edit_before_send`, default `true`):**
- Transcript (server **maupun** browser) dapat diedit user sebelum dikirim ke chat.
- **Transcript asli** tetap dicatat ke riwayat (`voice_transcriptions`) saat transkripsi
  selesai — **sebelum** edit — sehingga limit harian & audit memakai teks asli.
- **Riwayat server menyimpan transcript asli**, bukan versi editan. Versi editan adalah
  pesan chat, tercatat di riwayat chat (bukan riwayat voice).
- Bila `false`: transcript langsung dikirim tanpa kesempatan edit (mode "auto-send").

> **Kesimpulan:** tidak ada yang "bohong" — arsitektur ini sengaja memisahkan transkripsi
> (voice) dari percakapan (chat). Edit-before-send adalah perilaku chat yang sudah ada,
> kini diterapkan konsisten untuk kedua mode transkripsi.

### 6.5 Endpoint Riwayat Transkripsi — `GET/POST/DELETE /api/v1/me/voice-history`

**Baru.** Riwayat transkripsi (Q-04: server = source of truth, lokal = cache cepat).

| Method | Path | Fungsi |
|---|---|---|
| `GET` | `/me/voice-history?page&per_page` | Daftar riwayat (paged, ter-scope user) |
| `POST` | `/me/voice-history` | Catat transkripsi **mode browser** (mode server otomatis tercatat) |
| `DELETE` | `/me/voice-history/:id` | Hapus satu entri |
| `DELETE` | `/me/voice-history` | Hapus semua riwayat user |

```jsonc
// POST body (mode browser — mode server dicatat otomatis oleh /ai/voice)
{
  "transcript": "beli kopi 25 ribu",
  "language": "id",
  "mode": "browser",
  "duration_ms": 3200
}
```

> **Penting:** `POST /me/voice-history` untuk mode browser **wajib** agar limit harian
> (Q-03) menghitung pemakaian browser juga. Endpoint ini juga menjadi sumber
> `usage.voice_requests_per_day`.
>
> **Anti-abuse:** endpoint ini **dibatasi** (rate limit + validasi panjang transcript),
> dan tetap ter-scope user (tidak bisa menulis riwayat user lain).

---

## 7. Kontrak Komponen (Frontend)

```ts
export type VoiceMode = "server" | "browser" | "daily_limit" | "unavailable";

export interface VoiceInputProps {
  /** Dipanggil dengan teks final, dari mode apa pun. */
  onTranscript: (text: string) => void;
  /** Dipanggil saat mode berubah (untuk indikator UI). */
  onModeChange?: (mode: VoiceMode) => void;
  /** Dipanggil saat rekaman mulai/berhenti (untuk state tombol). */
  onStateChange?: (state: VoiceState) => void;
  disabled?: boolean;
  /** Override durasi maks (ms). Default: dari config (dinamis). */
  maxDurationMs?: number;
  /** Bahasa untuk rekaman ini (N-02, per-rekaman). Default: config default. */
  language?: string;
}
```

**Jaminan kontrak:**
1. `onTranscript` menerima **teks final** saja (tidak ada interim), dari mode apa pun.
2. `onModeChange` dipanggil **setiap** kali mode ditentukan (termasuk `daily_limit`/`unavailable`).
3. Tidak pernah throw ke parent; semua error ditangani internal (toast + state).
4. `disabled` mematikan tombol tanpa mengubah mode terakhir.
5. `language` bersifat **per-rekaman** (N-02) — diabaikan saat `language_mode === "id_only"`.
6. Durasi maks **selalu** dari config dinamis bila `maxDurationMs` tidak diberikan.

---

## 7.1 Aturan Bahasa Per-Rekaman (N-02)

- Saat `language_mode === "id_only"`: tidak ada pemilih bahasa; bahasa dipaksa
  `voice_default_language` (mis. `id`).
- Saat `language_mode === "multi"`: user **memilih bahasa per-rekaman** (dropdown kecil di
  dekat tombol mic); pilihan diteruskan sebagai prop `language`.
- Bahasa diteruskan ke:
  - **Mode browser:** `recognition.lang` (mis. `id-ID`, `en-US`).
  - **Mode server:** parameter bahasa pada request `/ai/voice` (diteruskan ke route/model).
- Bahasa yang dipakai **disimpan** di riwayat (`voice_transcriptions.language`).
- Pilihan terakhir **diingat di klien** (localStorage) sebagai default berikutnya —
  namun tetap dapat diganti tiap rekaman.

---

## 8. Aturan Tampilan (UX)

| Mode | Ikon tombol | Hint | Warna |
|---|---|---|---|
| `server` (idle) | `Mic` + badge "AI" | "Suara premium (AI) · sisa {n}/hari" | normal |
| `browser` (idle) | `Mic` + badge "Dasar" | "Mode dasar (tanpa kuota) · sisa {n}/hari" | normal |
| `unavailable` | `MicOff` (disabled) | "Browser tidak mendukung suara" | muted |
| `daily_limit` | `MicOff` (disabled) | "Batas harian tercapai · reset besok" | amber |
| recording | `Square` + **VU meter** | "Klik untuk berhenti" | merah |
| transcribing | `Loader2` (spin) | "Mentranskripsi…" | muted |

- Badge mode **wajib** terlihat (K-VOI-07) — user tahu kapan audio dikirim ke server.
- **VU meter wajib** saat `recording` (K-VOI-11 / Q-05).
- **Sisa limit harian** ditampilkan di hint (mis. "sisa 3/10 hari ini").
- **Pemilih bahasa** hanya muncul saat `language_mode === "multi"` (N-02); pilihan per-rekaman.
- **Zona reset** ditampilkan halus (mis. "reset 6j lagi") bila relevan (N-03).
- Saat fallback terjadi, tampilkan **satu** toast informatif (bukan berulang).
- **Hint upgrade tidak memaksa** (Q-02): dismissible, tanpa modal blocking.
- **Edit sebelum kirim** (N-04): bila aktif, transcript masuk ke input chat untuk
  ditinjau/diedit; bila nonaktif, langsung terkirim.

---

## 9. Interaksi dengan Kuota & Limit Real-Time

```
Mode server sukses
   └─ backend emit `usage_changed`
        └─ SSE → dispatcher invalidate `entitlements.me` (+ `voice-config` bila perlu)
             └─ useEntitlements() ter-update
                  └─ bar QuotaUsageCard berkurang (tanpa reload)
                  └─ sisa limit harian berkurang (tanpa reload)
                  └─ `hasVoiceQuota()` / `hasDailyVoiceQuota()` berikutnya baca nilai baru

Mode browser selesai
   └─ POST /me/voice-history (catat)
        └─ backend emit `usage_changed`
             └─ (idem) sisa limit harian berkurang
```

> Karena `useVoiceMode` membaca `usage` dari query yang sama, **mode akan otomatis
> turun** (`server`→`browser`, atau `browser`→`daily_limit`) pada rekaman berikutnya
> begitu sumber daya habis — tanpa reload.
