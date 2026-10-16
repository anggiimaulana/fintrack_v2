# 04 — Desain Teknis

| Field | Isi |
|---|---|
| **Dokumen** | Desain teknis: frontend **& backend**, tipe, hook, komponen, urutan event |
| **Tanggal** | 2026-10-15 (revisi 2) |
| **Status** | Spec selesai — siap eksekusi |
| **Prinsip** | Aditif, reusable, testable, backend otoritatif, konfigurasi dinamis |

> **PERUBAHAN dari revisi 1:** keputusan Q-01/Q-03/Q-04/Q-06 menambah **perubahan backend**
> (setting dinamis, limit harian, riwayat, endpoint config). Desain kini mencakup
> **frontend + backend**.

---

## 1. Ringkasan Arsitektur

```
┌──────────────────────────────────────────────────────────────────────────┐
│ FRONTEND                                                                  │
│                                                                           │
│  hooks/use-entitlements.ts ──(features, usage.voice_requests_per_day)──┐  │
│  hooks/use-voice-config.ts ──(max_duration, language_mode)─────────────┤  │
│                                                                        ▼  │
│  hooks/useVoiceMode.ts  ◄── pilih mode ──┐                                │
│  lib/voice-recorder.ts  ◄────────────────┤  (MediaRecorder + VU stream)   │
│  lib/voice-level.ts     ◄────────────────┤  (Web Audio AnalyserNode)      │
│  lib/voice-errors.ts    ◄────────────────┤  (klasifikasi 402/403/429)     │
│  lib/voice-history.ts   ◄────────────────┤  (server + cache lokal)        │
│                                          │                                │
│  components/ai/VoiceInput.tsx  ◄─────────┘  (router cerdas)               │
│  components/ai/VoiceLevelMeter.tsx          (VU meter — wajib)            │
│         │                                                                 │
│         │ onTranscript(text)  ← SAMA untuk semua mode                     │
│         ▼                                                                 │
│  ChatInput / InlineAiAssistant / (voice page digabung ke sini)            │
└──────────────────────────────────────────────────────────────────────────┘
                                   │
   (mode server) POST /ai/voice    │  (kedua mode) POST /me/voice-history
                                   ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ BACKEND                                                                   │
│                                                                           │
│  GET  /me/voice-config      (baru) → durasi, bahasa, limit harian         │
│  GET  /me/voice-history     (baru) → riwayat (source of truth)            │
│  POST /me/voice-history     (baru) → catat mode browser (hitung harian)   │
│  POST /ai/voice             (ada)  → featureGate → limitGate → ledger      │
│                                       → commit → emit usage_changed       │
│                                       → (auto) catat riwayat              │
│                                                                           │
│  Settings dinamis: voice_max_duration_seconds, voice_language_mode        │
│  Plan limit baru : voice_requests_per_day (per paket)                     │
│  Usage resolver  : voice_requests_per_day (hitung dari riwayat hari ini)  │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Backend — Perubahan

### 2.1 Setting global dinamis (Q-01, Q-05, Q-06, N-01..N-04)

**Target:** `internal/admin/settings.go` (`writableSettings`)

```go
// ── Durasi (Q-01) ──
// 0 → pakai default klien. Batas wajar 1..600 detik.
"voice_max_duration_seconds": {kind: "int", min: 1, max: 600},

// ── Bahasa (Q-06, N-02) ──
"voice_language_mode":    {kind: "enum", enum: []string{"id_only", "multi"}},
"voice_default_language": {kind: "enum", enum: []string{"id", "en"}},
"voice_languages":        {kind: "string", maxLen: 64},   // CSV: "id,en"

// ── Retensi riwayat (N-01) ──
"voice_history_retention_days": {kind: "int", min: 1, max: 3650},

// ── Zona reset harian (N-03) ──
"voice_daily_reset_timezone": {kind: "enum", enum: []string{"UTC", "Asia/Jakarta"}},

// ── Perilaku (N-04, Q-05, Q-08) ──
"voice_allow_edit_before_send": {kind: "bool"},
"voice_vu_meter_enabled":       {kind: "bool"},
"voice_fallback_to_browser":    {kind: "bool"},

// ── Kill switch (opsional) ──
"voice_server_enabled": {kind: "bool"},
```

Tambahkan juga ke `settingsSnapshot` (agar tampil di UI admin) dan daftar reload
(bila perlu invalidasi runtime config).

**Resolver:** dibaca via `runtimeconfig.Resolver` (DB-over-env, TTL 5s) — **sudah ada**.
Tidak perlu infrastruktur baru.

> **Prinsip (K-VOI-19):** **semua** knob voice ada di sini; tidak ada yang hardcode.
> Nilai default hanya *fallback* bila admin belum mengatur.

### 2.2 Limit harian per paket (Q-03)

**Target:** `database/migrations/00XX_voice_daily_limit.sql` (seed/opsi), `plan_limits`

Limit baru: `voice_requests_per_day` (dinamis, via admin plan editor yang sudah ada).

```sql
-- Contoh seed (dinamis lewat admin UI; ini hanya default awal)
INSERT INTO plan_limits (plan_id, limit_key, limit_value) VALUES
  ('...Free',     'voice_requests_per_day', 1),
  ('...Pro',      'voice_requests_per_day', 50),
  ('...Business', 'voice_requests_per_day', -1)   -- unlimited
ON CONFLICT (plan_id, limit_key) DO UPDATE SET limit_value = EXCLUDED.limit_value;
```

Tambahkan ke `entitlement/catalog.go` `LimitCatalog`:

```go
{Key: "voice_requests_per_day", Label: "Permintaan suara / hari", Unit: "permintaan", Enforced: true},
```

### 2.3 Usage resolver baru

**Target:** `internal/delivery/http/middleware/usage_resolvers.go`, `cmd/server/main.go`

```go
// UsageProviders — tambah field
CountVoiceToday func(ctx context.Context, userID uuid.UUID) (int64, error)

// UsageFor — tambah case
case "voice_requests_per_day":
    return p.countVoiceToday
```

**Sumber hitungan:** jumlah baris `voice_transcriptions` untuk user pada **hari ini**,
di mana batas hari mengikuti **zona dinamis** (`voice_daily_reset_timezone`, N-03).

> **Anti-cheat (append-only):** hitung dari `created_at` (baris yang ada hari ini),
> **bukan** dari "jumlah baris tersisa". Menghapus riwayat tidak mengurangi counter —
> lihat R-14 di dok 07.

### 2.3.1 Worker Retensi Riwayat (N-01)

**Target:** `internal/billing/voice_retention_worker.go` (baru, mengikuti pola
`billing.NewCleanerWorker`)

**Langkah:**
1. Worker harian: hapus `voice_transcriptions` dengan `created_at < now() - retention_days`.
2. `retention_days` dibaca dari runtime config (`voice_history_retention_days`, default 90).
3. Start di `main.go` (`bgCtx`), `defer Stop()`.

**DoD:**
- [ ] Riwayat lebih tua dari N hari terhapus
- [ ] Nilai N dinamis (ubah admin → berlaku pada sweep berikutnya)
- [ ] Tidak menyentuh hari berjalan (aman untuk limit harian)

### 2.4 Tabel riwayat transkripsi (Q-04)

**Target:** `database/migrations/00XX_voice_transcriptions.sql`

```sql
CREATE TABLE IF NOT EXISTS voice_transcriptions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    transcript    text NOT NULL,
    language      varchar(16) NOT NULL DEFAULT 'id',
    mode          varchar(16) NOT NULL CHECK (mode IN ('server', 'browser')),
    duration_ms   integer NOT NULL DEFAULT 0,
    tokens_used   bigint  NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_voice_tx_user_day
    ON voice_transcriptions (user_id, created_at DESC);
```

**Catatan:**
- Ter-scope `user_id` (isolasi ketat, mengikuti pola repo lain).
- `mode` membedakan server/browser (untuk audit & limit).
- `created_at` dipakai untuk hitung harian.

**Tabel pendamping — `voice_daily_usage` (counter append-only, R-14):**

```sql
CREATE TABLE IF NOT EXISTS voice_daily_usage (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    usage_day  date NOT NULL,
    used       integer NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, usage_day)
);
```

> **Mengapa tabel terpisah (bukan COUNT dari riwayat):** limit harian **tidak boleh**
> bisa di-reset dengan menghapus riwayat. Bila counter dihitung dari jumlah baris
> `voice_transcriptions`, user bisa menghapus riwayat untuk mendapat kuota hari itu
> kembali (celah R-14). Tabel `voice_daily_usage` di-*increment* (bukan dihitung ulang)
> dan **tidak pernah** dihapus oleh user, sehingga otoritatif untuk hari tersebut.
> Ini ditemukan & diperbaiki saat verifikasi live (bukti: hapus riwayat → counter tetap).

**Repository:** `internal/repository/postgres/voice_repo.go` (baru) — `Create`, `ListByUser`,
`CountSince`, `DeleteOne`, `ClearByUser`, `DeleteOlderThan`, **`IncrementDailyUsage`**,
**`DailyUsage`**. Semua query riwayat `WHERE user_id = $1`.

### 2.5 Endpoint baru

**Target:** `internal/delivery/http/handler/voice_handler.go` (baru), `router.go`

| Method | Path | Handler | Gate |
|---|---|---|---|
| `GET` | `/me/voice-config` | `VoiceConfig` | auth + RequireUser |
| `GET` | `/me/voice-history` | `ListHistory` | auth + RequireUser |
| `POST` | `/me/voice-history` | `RecordBrowserHistory` | auth + RequireUser + **rate limit** + **limitGate(voice_requests_per_day)** |
| `DELETE` | `/me/voice-history/:id` | `DeleteHistory` | auth + RequireUser |
| `DELETE` | `/me/voice-history` | `ClearHistory` | auth + RequireUser |

**`GET /me/voice-config`** membaca:
- `voice_max_duration_seconds` (default 60) via runtime config.
- `voice_language_mode` (default `id_only`) + `voice_default_language` + `voice_languages`.
- `voice_history_retention_days`, `voice_daily_reset_timezone`.
- `voice_allow_edit_before_send`, `voice_vu_meter_enabled`, `voice_fallback_to_browser`.
- `voice_server_enabled` (kill switch).
- `usage.voice_requests_per_day` dari plan limit + counter, plus `daily_resets_at`.

**`POST /me/voice-history`** (mode browser):
1. Validasi payload (transcript non-kosong, panjang wajar, `mode == "browser"`, `language` valid).
2. Cek limit harian (gate, zona dinamis) — bila habis → 402.
3. Insert baris `voice_transcriptions`.
4. Emit `usage_changed` (agar limit harian real-time).
5. Return entri yang dibuat.

**`POST /ai/voice`** (mode server) — **tambahan**:
- Terima `language` opsional (N-02) → diteruskan ke gateway.
- Setelah sukses, **insert riwayat** (`mode='server'`, `duration_ms`, `tokens_used`, `language`).
- Ini menjadikan riwayat server otoritatif untuk kedua mode.

> **N-04 (edit-before-send):** transcript **asli** dicatat ke riwayat **saat transkripsi
> selesai** (sebelum edit). Riwayat voice menyimpan teks asli; versi editan user adalah
> pesan chat (tercatat di riwayat chat). Detail di dok 03 §6.4.1.

### 2.6 Gate limit harian pada `/ai/voice`

**Target:** `router.go`

```go
ai.POST("/voice", gated(aiH.ProcessVoice,
    featureGate(config, "voice_input"),
    limitGate(config, "voice_requests_per_day"),   // ← BARU
    limitGate(config, "ai_requests_per_cycle"),
    limitGate(config, "ai_tokens_per_cycle"))...)
```

> **Catatan:** `voice_requests_per_day` **juga** membatasi mode server, sehingga total
> pemakaian harian (server + browser) terkendali.

### 2.7 Event realtime

- `POST /ai/voice` sukses → **sudah** emit `usage_changed`; tambahkan
  `voice_requests_per_day` ke daftar `LimitKeys` payload.
- `POST /me/voice-history` → **emit baru** `usage_changed` dengan
  `LimitKeys: ["voice_requests_per_day"]`.

---

## 3. Frontend — Tipe & Modul Baru

### 3.1 `lib/voice-mode.ts` — pemilih mode (pure, testable)

```ts
export type VoiceMode = "server" | "browser" | "daily_limit" | "unavailable";

export interface VoiceCapabilities {
  browserSTT: boolean;   // SpeechRecognition tersedia
  recorder: boolean;     // MediaRecorder + getUserMedia tersedia
}

export interface VoiceEntitlement {
  hasVoiceFeature: boolean;   // features.voice_input
  hasQuota: boolean;          // token & request sisa
  hasDailyQuota: boolean;     // voice_requests_per_day sisa
}

export function decideVoiceMode(
  ent: VoiceEntitlement,
  cap: VoiceCapabilities,
): VoiceMode {
  // Gate harian lebih dulu: habis → blokir SEMUA mode.
  if (!ent.hasDailyQuota) return "daily_limit";
  if (ent.hasVoiceFeature && ent.hasQuota && cap.recorder) return "server";
  if (cap.browserSTT) return "browser";
  return "unavailable";
}
```

### 3.2 `lib/voice-errors.ts` — klasifikasi error (pure)

```ts
export type VoiceErrorKind = "fallback" | "daily_limit" | "fatal" | "permission";

export function classifyVoiceError(err: unknown): VoiceErrorKind {
  const status = (err as { response?: { status?: number } })?.response?.status;
  const code = (err as { response?: { data?: { error?: string } } })
    ?.response?.data?.error;

  if (code === "daily_limit_exceeded" || code === "voice_daily_limit") return "daily_limit";
  if (status === 400) return "fatal";
  if (code === "insufficient_quota" || code === "limit_exceeded") return "fallback";
  if (code === "feature_not_in_plan") return "fallback";
  if (code === "rate_limited") return "fallback";
  if (status === 402 || status === 403 || status === 429) return "fallback";
  if (typeof status === "number" && status >= 500) return "fallback";
  if (status === undefined) return "fallback";  // network/timeout
  return "fatal";
}
```

> **Catatan:** kode `daily_limit` dibedakan agar klien **memblokir** (bukan fallback).

### 3.3 `lib/voice-recorder.ts` — pembungkus MediaRecorder

```ts
export interface Recording {
  blob: Blob;
  durationMs: number;
  /** Stream tetap hidup agar VU meter bisa membaca level. */
  stream: MediaStream;
}

export interface RecorderHandle {
  /** Stream aktif (untuk VU meter). */
  stream: MediaStream;
  stop: () => Promise<Recording>;
  cancel: () => void;
}

export async function startRecording(maxDurationMs: number): Promise<RecorderHandle>;
```

**Persyaratan:**
- `getUserMedia({ audio: true })`; MIME via `MediaRecorder.isTypeSupported`.
- **Stream diekspos** agar `VoiceLevelMeter` bisa membaca level (Q-05).
- Auto-stop pada `maxDurationMs` (**dinamis** dari config).
- **Selalu** stop semua track pada `stop()`/`cancel()`/error/unmount.
- Error `NotAllowedError`/`NotFoundError`/`NotSupportedError` diteruskan apa adanya.

### 3.4 `lib/voice-level.ts` — analisis level (pure-ish)

```ts
export interface LevelMeter {
  /** Mulai menganalisis stream; memanggil onLevel(0..1) tiap frame. */
  start: (stream: MediaStream, onLevel: (level: number) => void) => void;
  stop: () => void;
}

export function createLevelMeter(): LevelMeter;
```

**Persyaratan:**
- `AudioContext` + `AnalyserNode`; `requestAnimationFrame` (bukan `setInterval`).
- `stop()` **wajib** menutup `AudioContext` + `cancelAnimationFrame` (anti-kebocoran).

### 3.5 `lib/voice-audio.ts` — encode

```ts
export async function blobToBase64(blob: Blob): Promise<string>;
export const MAX_AUDIO_BASE64 = 5 * 1024 * 1024;  // sesuai batas backend
```

### 3.6 `lib/voice-history.ts` — klien riwayat

```ts
export interface VoiceHistoryEntry {
  id: string;
  transcript: string;
  language: string;
  mode: "server" | "browser";
  duration_ms: number;
  created_at: string;
}

export async function recordBrowserHistory(input: {
  transcript: string; language: string; durationMs: number;
}): Promise<VoiceHistoryEntry>;
```

### 3.7 `hooks/useVoiceConfig.ts` — config global (baru)

```ts
export interface VoiceConfig {
  // Durasi (Q-01)
  max_duration_seconds: number;
  // Bahasa (Q-06, N-02)
  language_mode: "id_only" | "multi";
  default_language: string;
  languages: string[];
  language_per_recording: boolean;
  // Limit harian (Q-03)
  daily_limit: UsageEntry;
  // Zona reset (N-03)
  daily_reset_timezone: string;
  daily_resets_at: string;
  // Retensi (N-01)
  history_retention_days: number;
  // Perilaku (N-04, Q-05, Q-08)
  allow_edit_before_send: boolean;
  vu_meter_enabled: boolean;
  fallback_to_browser: boolean;
  // Ketersediaan
  server_enabled: boolean;
  server_available: boolean;
}

export function useVoiceConfig(): { config: VoiceConfig | null; isLoading: boolean };
```

### 3.8 `hooks/useVoiceMode.ts` — mode reaktif

```ts
export interface UseVoiceModeResult {
  mode: VoiceMode;
  canRecord: boolean;      // mode !== "unavailable" && mode !== "daily_limit"
  isPremium: boolean;      // mode === "server"
  dailyRemaining: number;  // untuk hint "sisa n/hari"
  capabilities: VoiceCapabilities;
  config: VoiceConfig | null;
}

export function useVoiceMode(): UseVoiceModeResult;
```

**Implementasi:**
- Baca `useEntitlements()` + `useVoiceConfig()`.
- Deteksi kapabilitas browser di `useEffect` (SSR-safe, default `false`).
- `mode` recompute otomatis saat entitlement/config berubah.
- Bila `config.server_enabled === false` → paksa `browser`/`unavailable` (kill switch).

### 3.9 `hooks/useVoiceLanguage.ts` — bahasa per-rekaman (N-02)

```ts
export interface UseVoiceLanguageResult {
  /** Bahasa efektif untuk rekaman berikutnya. */
  language: string;
  /** true bila user boleh memilih (language_mode === "multi"). */
  canChoose: boolean;
  /** Daftar pilihan bahasa (kosong saat id_only). */
  options: string[];
  /** Ganti bahasa (disimpan ke localStorage sebagai default berikutnya). */
  setLanguage: (lang: string) => void;
}

export function useVoiceLanguage(config: VoiceConfig | null): UseVoiceLanguageResult;
```

**Implementasi:**
- Saat `id_only`: `language = config.default_language`, `canChoose = false`.
- Saat `multi`: `options = config.languages`, pilihan per-rekaman, diingat di localStorage.

### 3.10 `components/ai/VoiceLevelMeter.tsx` (baru — Q-05)

```tsx
export interface VoiceLevelMeterProps {
  stream: MediaStream | null;
  active: boolean;
}
```

Menampilkan bar level (0..1). Wajib tampil saat `recording` (bila `vu_meter_enabled`).

---

## 3.11 Alur Edit-Sebelum-Kirim (N-04)

```
Transkripsi selesai (server atau browser)
   │
   ├─► [WAJIB] catat transcript ASLI ke riwayat (voice_transcriptions)
   │
   └─► config.allow_edit_before_send ?
         ├─ true  → isi input chat dengan transcript → user boleh edit → kirim via /ai/chat
         └─ false → langsung kirim ke chat (auto-send) via /ai/chat
```

**Aturan:**
- Riwayat voice **selalu** menyimpan transcript **asli** (untuk audit + limit harian).
- Versi editan (bila ada) tercatat di **riwayat chat**, bukan riwayat voice.
- `onTranscript(text)` tetap menerima teks (asli atau editan, sesuai mode) untuk
  kompatibilitas pemanggil yang ada.

---

## 4. Perubahan `VoiceInput.tsx`

### 4.1 Sebelum vs Sesudah

| Aspek | Sekarang | Sesudah |
|---|---|---|
| Mode | Browser saja | `server` \| `browser` \| `daily_limit` \| `unavailable` |
| Sadar entitlement+config | ❌ | ✅ `useVoiceMode` |
| Rekam audio nyata | ❌ | ✅ `lib/voice-recorder` (mode server) |
| Fallback | ❌ | ✅ otomatis (bila `fallback_to_browser`) |
| Indikator mode | ❌ | ✅ badge "AI"/"Dasar" |
| **VU meter** | ❌ | ✅ **wajib** (Q-05), toggle dinamis |
| Durasi maks | konstanta | **dinamis** (Q-01) |
| Bahasa | hardcode `id-ID` | **dinamis**, **per-rekaman** (Q-06, N-02) |
| Riwayat | lokal | server + lokal (Q-04), retensi dinamis (N-01) |
| Edit sebelum kirim | — | **dinamis** (N-04) |
| Kill switch | — | `voice_server_enabled` (dinamis) |
| `onTranscript` | teks | **tetap** teks (kontrak tak berubah) |

### 4.2 Logika fallback di `startServer`

```ts
async function startServer() {
  setState("recording");
  const handle = await startRecording(config.max_duration_seconds * 1000);
  setStream(handle.stream);                 // ← untuk VU meter
  const recording = await handle.stop();    // saat user stop
  setStream(null);
  setState("transcribing");
  try {
    const base64 = await blobToBase64(recording.blob);
    if (base64.length > MAX_AUDIO_BASE64) { toast.error("Rekaman terlalu panjang."); setState("idle"); return; }
    const res = await api.post("/ai/voice", { audio: base64 }, { headers: { "X-Request-Id": uuid() } });
    const text = res.data?.data?.transcript ?? "";
    if (text) onTranscript(text);           // riwayat server dicatat backend
    setState("idle");
  } catch (err) {
    const kind = classifyVoiceError(err);
    if (kind === "daily_limit") { toast.error("Batas suara harian tercapai."); showUpgradeHint(); setState("idle"); return; }
    if (kind === "fallback" && !fallbackUsedRef.current) {
      fallbackUsedRef.current = true;
      toast.info("Beralih ke mode dasar.");
      startBrowser();
      return;
    }
    if (kind === "permission") toast.error("Izin mikrofon ditolak.");
    else toast.error("Gagal memproses suara.");
    setState("idle");
  }
}
```

### 4.3 Mode browser + catat riwayat

```ts
// Setelah onresult (transcript final)
async function onBrowserTranscript(text: string, durationMs: number) {
  onTranscript(text);
  // Catat ke server agar limit harian menghitung mode browser juga (Q-03).
  try {
    await recordBrowserHistory({ transcript: text, language: activeLang, durationMs });
  } catch {
    // Best-effort: kegagalan mencatat tidak boleh menggagalkan UX.
    // (Backend tetap otoritatif untuk rekaman berikutnya.)
  }
}
```

---

## 5. Penggabungan Halaman `/user/voice` (Q-07)

| Sebelum | Sesudah |
|---|---|
| Rute `/user/voice` standalone | **Dihapus** |
| `app/(dashboard)/user/voice/page.tsx` | Dihapus (logika pindah ke chat) |
| Menu nav "Input Suara" | Dihapus |
| SpeechRecognition inline | Pakai `<VoiceInput>` yang sama |

**Riwayat** dipindahkan ke panel di dalam chat (mis. drawer "Riwayat Suara") yang
memakai `useVoiceHistory()` (query ke `/me/voice-history`).

> **Alasan:** satu tempat untuk semua interaksi AI (chat + suara + OCR) → tidak
> membingungkan (Q-07).

---

## 6. Urutan Event (Mode Server, Sukses)

```
1. User tekan mic
2. useVoiceMode → "server" (voice_input ✅, kuota ✅, harian ✅, recorder ✅)
3. getUserMedia → stream
4. VoiceLevelMeter aktif (VU meter)
5. MediaRecorder.start() (auto-stop pada config.max_duration_seconds)
6. User stop → recorder.stop() → Blob; stream track di-stop
7. blobToBase64(blob) → base64
8. POST /ai/voice { audio }  (X-Request-Id)
      ├─ featureGate ✅
      ├─ limitGate(voice_requests_per_day) ✅   ← BARU
      ├─ limitGate(requests) ✅
      ├─ limitGate(tokens) ✅
      ├─ ledger.Reserve → TranscribeAudio → Commit
      ├─ INSERT voice_transcriptions (mode=server)   ← BARU
      └─ emit usage_changed (ai_tokens, ai_requests, voice_requests_per_day)
9. 200 { transcript }
10. onTranscript(transcript)
11. Bar kuota + sisa harian berkurang (real-time)
```

## 7. Urutan Event (Mode Browser, Sukses)

```
1. User tekan mic
2. useVoiceMode → "browser" (tanpa voice_input ATAU kuota habis)
3. SpeechRecognition.start() (lang dari config)
4. VoiceLevelMeter aktif (VU meter) — stream dari getUserMedia terpisah
5. onresult → transcript final
6. onTranscript(transcript)
7. POST /me/voice-history { transcript, mode: "browser", duration_ms }   ← BARU
      ├─ limitGate(voice_requests_per_day) ✅
      ├─ INSERT voice_transcriptions (mode=browser)
      └─ emit usage_changed (voice_requests_per_day)
8. Sisa harian berkurang (real-time)
```

> **Catatan teknis:** mode browser memakai `SpeechRecognition` (yang mengelola mic-nya
> sendiri). Untuk VU meter, klien membuka `getUserMedia` **terpisah** hanya untuk
> analisis level — **tidak** merekam audio.

---

## 8. Penanganan Race Condition (Klien vs Backend)

| Race | Skenario | Mitigasi |
|---|---|---|
| Kuota habis antara cek & kirim | `useVoiceMode` lihat sisa 5, reserve gagal | Backend 402/429 → fallback |
| **Limit harian habis** antara cek & kirim | Cek bilang sisa 1, backend bilang habis | Backend 402 → blokir + hint |
| Plan berubah saat merekam | Admin downgrade | Backend 403 → fallback |
| Dua tab merekam bersamaan | Tab A & B kirim | Ledger reserve atomik; limit harian dicek backend |
| `usage_changed` belum sampai | Bar belum update | Backend otoritatif tetap bekerja |
| Timeout jaringan | POST tidak balas | Fallback |
| Riwayat gagal dicatat (mode browser) | POST history error | Best-effort; tidak menggagalkan UX |

---

## 9. Kompatibilitas & SSR

| Isu | Penanganan |
|---|---|
| `window` tidak ada saat SSR | Deteksi kapabilitas di `useEffect`, default `false` |
| `AudioContext` tidak ada | Guard `typeof`; VU meter non-aktif tanpa crash |
| `MediaRecorder.isTypeSupported` tidak ada | Guard + try/catch |
| Firefox (tanpa `SpeechRecognition`) | Hanya mode server; bila kuota/harian habis → pesan jelas |
| HTTPS wajib untuk `getUserMedia` | Dokumentasi; `localhost` dikecualikan browser |
| Audio > 5MB | Cek panjang base64 pra-kirim |

---

## 10. Dampak & Non-Dampak

| Berubah | Tidak berubah |
|---|---|
| **Backend:** settings, limit harian, riwayat, endpoint baru | Kontrak `onTranscript(text)` |
| `VoiceInput.tsx` (router cerdas + VU meter) | Gate/ledger `/ai/voice` yang ada |
| Halaman `/user/voice` **dihapus** (digabung) | Event `usage_changed` (hanya ditambah key) |
| Modul baru: `lib/voice-*.ts`, `hooks/useVoice*.ts`, `VoiceLevelMeter` | Isolasi data per-user |
| `ChatInput`/`InlineAiAssistant` **tidak** berubah | Halaman lain |

---

## 11. Alternatif yang Dipertimbangkan

| Alternatif | Alasan ditolak |
|---|---|
| Selalu pakai server | User Free/kuota habis kehilangan suara; Firefox mati |
| Selalu pakai browser | Fitur berbayar tidak nyata |
| Limit harian hanya di klien | Tidak otoritatif; mudah dilewati |
| Durasi/bahasa hardcode | Melanggar prinsip "dinamis via Super Admin" |
| Riwayat lokal saja | Hilang saat ganti perangkat (Q-04) |
| VU meter opsional | Q-05 mewajibkan |
| Pertahankan halaman voice terpisah | Q-07: membingungkan |
