# 02 — Mekanisme Voice: Web Speech API vs `/ai/voice`

| Field | Isi |
|---|---|
| **Dokumen** | Penjelasan mekanisme kedua jalur transkripsi & perbandingannya |
| **Tanggal** | 2026-10-15 |
| **Status** | Selesai |

---

## 1. Mode Browser — Web Speech API (`SpeechRecognition`)

### 1.1 Cara kerja

```
User klik mic
   │
   ▼
new SpeechRecognition()  ──► browser meminta izin mikrofon (getUserMedia implisit)
   │
   ▼
recognition.start()  ──► browser merekam audio
   │
   ▼
browser mengirim audio ke layanan speech VENDOR (Chrome→Google, dst.)
   │
   ▼
onresult(event)  ──► { isFinal: true, transcript: "..." }
   │
   ▼
onTranscript(teks)  ──► parent (chat/inline)
```

### 1.2 Karakteristik

| Aspek | Nilai |
|---|---|
| Di mana transkripsi | Sisi vendor browser (bukan server kita) |
| Audio ke server kita | **Tidak** |
| Menyentuh kuota | **Tidak** (tapi menyentuh **limit harian** — lihat §5) |
| Ter-gate `voice_input` | **Tidak** |
| Dukungan browser | Chrome, Edge, Safari (Firefox: **tidak**) |
| Privasi | Audio dikirim ke vendor browser (Google pada Chrome) |
| Offline | Tidak (butuh koneksi untuk vendor) |
| Biaya kita | Rp 0 |
| Kontrol bahasa | `recognition.lang` (dinamis — lihat §4) |
| Streaming | Ya, `interimResults: true` (hasil sementara) |

### 1.3 Batasan teknis penting

- **Tidak dapat diprediksi kualitasnya** (bergantung vendor).
- **Tidak ada jaminan ketersediaan** (`network` error → toast di kode saat ini).
- **Tidak ada payload audio** yang bisa kita simpan/audit.
- `continuous: true` di halaman `/user/voice` vs `false` di `VoiceInput.tsx`
  → perilaku berbeda antar tempat (inkonsistensi).

---

## 2. Mode Server — `POST /api/v1/ai/voice`

### 2.1 Cara kerja (end-to-end)

```
Klien merekam audio (MediaRecorder) ──► Blob (audio/webm)
   │
   ▼
base64(Blob) ──► POST /api/v1/ai/voice { audio: "<base64>" }
   │
   ▼
[Middleware]
   ├─ AuthMiddleware        → user_id dari JWT cookie
   ├─ RequireUser
   ├─ featureGate("voice_input")        → 403 bila plan tidak punya
   ├─ limitGate("ai_requests_per_cycle") → 402 bila habis
   └─ limitGate("ai_tokens_per_cycle")   → 402 bila habis
   │
   ▼
[Handler ProcessVoice]  (ai_handler.go:473)
   ├─ validasi ukuran ≤ 5MB
   ├─ ledger.Reserve(est=voice_estimate, ModalityVoice, key=voice-reserve:<reqID>)
   │     └─ gagal → 402 "Sisa kuota Anda tidak mencukupi..."
   ├─ aiUC.ProcessVoice(ctx, userID, audioBase64)
   │     └─ gateway.TranscribeAudio(...)  ──► provider AI
   ├─ settleVoice(...)
   │     ├─ sukses → ledger.Commit(actual tokens)
   │     └─ gagal  → ledger.Release(...)   (kuota kembali)
   └─ emitUsageChanged("ai_voice", ai_tokens, ai_requests)
   │
   ▼
200 { success: true, transcript: "..." }
```

### 2.2 Karakteristik

| Aspek | Nilai |
|---|---|
| Di mana transkripsi | AI Gateway kita (provider terpilih, failover) |
| Audio ke server kita | **Ya** (base64, ≤5MB) |
| Menyentuh kuota | **Ya** (token + request) |
| Ter-gate `voice_input` | **Ya** |
| Dukungan browser | Semua yang punya `MediaRecorder`/`getUserMedia` |
| Privasi | Audio diproses server kita → provider AI |
| Offline | Tidak |
| Biaya kita | Token provider (dibebankan ke kuota user) |
| Kontrol bahasa | Ditentukan route/model gateway |
| Streaming | Belum (satu request → satu transcript) |

### 2.3 Titik akuntansi kuota (bukti)

```go
// ai_handler.go — reserve
res, rerr := h.ledger.Reserve(c.Request.Context(), userID,
    h.estimateFor(c.Request.Context(), billing.ModalityVoice),
    billing.ModalityVoice, "voice-reserve:"+reqID)

// ai_handler.go — settle
settleVoice(h.ledger, userID, reservation, result, usage, err)
//  sukses → Commit(actual) ; gagal → Release
```

```go
// settleVoice
if callErr == nil && resultSucceeded(result) {
    actual := int64(usage.TotalTokens)
    if actual <= 0 { actual = reservation.Amount }
    if actual <= 0 { actual = 1 }
    _ = ledger.Commit(ctx, userID, reservation.ID, actual, "voice-commit:"+reservation.ID.String())
    return
}
_ = ledger.Release(ctx, userID, reservation.ID, "voice-release:"+reservation.ID.String())
```

```go
// internal/aigateway/gateway/service.go:452 — estimasi bila provider tak melaporkan usage
usage := metering.EstimateAudio(estimateAudioSeconds(len(req.Audio)), text)
// estimateAudioSeconds: ~4000 byte ≈ 1 detik (WebM opus ~32kbps)
```

---

## 3. Perbandingan Langsung

| Kriteria | Mode Browser (Web Speech) | Mode Server (`/ai/voice`) |
|---|---|---|
| Biaya kuota | 0 | Token + request |
| Gate `voice_input` | Tidak | Ya |
| Privasi audio | Ke vendor browser | Ke server + provider kita |
| Dukungan browser | Chrome/Edge/Safari | Lebih luas (MediaRecorder) |
| Kualitas | Vendor-dependent | Model pilihan kita |
| Auditabilitas | Tidak ada | Ada (ledger) |
| Failover | Tidak | Ya (route breaker gateway) |
| Butuh backend | Tidak | Ya |
| Bisa offline | Tidak | Tidak |
| Cocok untuk | Semua plan (fallback) | Plan Pro/Business (premium) |

---

## 4. Mengapa Keduanya Dibutuhkan (bukan pilih salah satu)

1. **Mode Browser = jaring keselamatan.** Selalu tersedia untuk plan apa pun, tanpa kuota.
   Tanpa ini, user Free (atau user yang kuotanya habis) kehilangan kemampuan bicara.
2. **Mode Server = fitur bernilai tambah.** Kualitas lebih baik, konsisten lintas browser,
   teraudit, dan merupakan alasan nyata untuk upgrade ke Pro/Business.
3. **Kombinasi = "smart routing".** Sistem memilih otomatis jalur terbaik yang tersedia
   untuk user saat itu, dan **degradasi dengan anggun** saat sumber daya habis.

> Ini bukan "dua fitur berbeda", melainkan **satu fitur dengan dua tingkat layanan**
> (premium vs dasar), dipilih otomatis oleh sistem berdasarkan entitlement + ketersediaan.

---

## 5. Catatan Kompatibilitas

| Kemampuan | Chrome | Edge | Safari | Firefox |
|---|---|---|---|---|
| `SpeechRecognition` | ✅ | ✅ | ✅ (webkit) | ❌ |
| `MediaRecorder` | ✅ | ✅ | ✅ | ✅ |
| `getUserMedia` | ✅ (HTTPS/localhost) | ✅ | ✅ | ✅ |

**Implikasi:**
- Firefox: **hanya** bisa Mode Server (karena `SpeechRecognition` tidak ada).
  → Bila Firefox + plan tanpa voice: **tidak ada** transkripsi yang bisa dilakukan
  (harus tampilkan pesan jelas, bukan crash). Lihat `07-risiko-dan-mitigasi.md`.
- `getUserMedia` **wajib HTTPS** (kecuali `localhost`) — dev di `localhost:3001` aman,
  produksi wajib HTTPS.

---

## 6. VU Meter (Indikator Level Mikrofon) — WAJIB (Q-05)

### 4.1 Mengapa wajib

Tanpa indikator level, user **tidak tahu** apakah mikrofon benar-benar menangkap suara.
Ini penyebab paling umum keluhan "suara saya tidak masuk". VU meter memberi umpan balik
visual real-time.

### 4.2 Cara kerja (Web Audio API)

```
getUserMedia({ audio: true })
   │
   ▼
AudioContext ──► MediaStreamAudioSourceNode ──► AnalyserNode
                                                     │
                                          getByteTimeDomainData(buf)
                                                     │
                                                     ▼
                                          hitung RMS → level 0..1
                                                     │
                                                     ▼
                                          requestAnimationFrame → render bar
```

### 4.3 Detail teknis

```ts
const ctx = new AudioContext();
const src = ctx.createMediaStreamSource(stream);
const analyser = ctx.createAnalyser();
analyser.fftSize = 256;
analyser.smoothingTimeConstant = 0.8;
src.connect(analyser);

const buf = new Uint8Array(analyser.fftSize);
function tick() {
  analyser.getByteTimeDomainData(buf);
  let sum = 0;
  for (const v of buf) {
    const x = (v - 128) / 128;   // normalize ke [-1, 1]
    sum += x * x;
  }
  const rms = Math.sqrt(sum / buf.length);
  const level = Math.min(1, rms * 3);  // skala tampilan
  // render level (0..1) ke bar
  raf = requestAnimationFrame(tick);
}
```

### 4.4 Aturan

- **Wajib** tampil selama `state === "recording"` (kedua mode).
- Harus **membersihkan** `AudioContext` + `requestAnimationFrame` saat berhenti/unmount
  (mencegah kebocoran resource).
- Berbasis `requestAnimationFrame` (bukan `setInterval`) agar hemat & sinkron dengan frame.
- Tidak boleh merekam/mengirim data apa pun — hanya menganalisis stream yang sama.

### 4.5 Komponen

```
components/ai/VoiceLevelMeter.tsx
  props: { stream: MediaStream | null, active: boolean }
```

---

## 7. Limit Harian (Q-03) — Berlaku untuk KEDUA Mode

> **Keputusan Q-03:** mode browser dibatasi **per hari** dan **berbeda per paket**
> (mis. Free = 1/hari). Ini **berlaku juga** untuk mode server, sehingga total pemakaian
> voice per hari terkendali di kedua jalur.

| Paket (contoh) | `voice_requests_per_day` |
|---|---|
| Free | 1 |
| Pro | 50 |
| Business | -1 (unlimited) |

**Mekanisme:**
- Disimpan di `plan_limits` dengan key baru `voice_requests_per_day` (dinamis via admin).
- Dihitung dari **riwayat transkripsi** (tabel `voice_transcriptions`) per hari kalender.
- Diperiksa **di backend** (otoritatif) untuk kedua mode:
  - Mode server: gate sebelum reserve.
  - Mode browser: klien mencatat pemakaian lewat endpoint `POST /ai/voice/history`
    (mode browser tidak lewat `/ai/voice`), backend yang menghitung.
- Diekspos di `/me/entitlements` sebagai `usage.voice_requests_per_day`.

> **Rasional:** meski mode browser gratis dari sisi token, tetap perlu dibatasi agar
> (a) konsisten dengan paket, (b) mendorong upgrade, (c) mencegah penyalahgunaan.

---

## 8. Konfigurasi Bahasa (Q-06, N-02) — Dinamis

| Setting | Nilai | Default | Efek |
|---|---|---|---|
| `voice_language_mode` | `id_only` \| `multi` | `id_only` | Menentukan bahasa yang boleh dipilih user |
| `voice_default_language` | enum (mis. `id`,`en`) | `id` | Bahasa default |
| `voice_languages` | CSV (mis. `id,en`) | `id` | Daftar bahasa saat `multi` |

**Perilaku:**
- `id_only`: klien memaksa `recognition.lang` = bahasa default (mode browser) dan tidak
  menawarkan pilihan bahasa.
- `multi`: klien menampilkan pemilih bahasa; bahasa dipilih **per-rekaman** (N-02) dan
  diteruskan ke mode browser (`recognition.lang`) dan mode server (parameter route/model).
- Bahasa yang dipakai **disimpan** di riwayat (`voice_transcriptions.language`).

**Sumber nilai:** `GET /me/voice-config` yang membaca `runtimeconfig` → `settings`.
Dinamis: Super Admin ubah → berlaku tanpa redeploy (≤ TTL resolver 5 detik).

---

## 9. Retensi Riwayat (N-01) — Dinamis

| Setting | Nilai | Default | Efek |
|---|---|---|---|
| `voice_history_retention_days` | int (1..3650) | `90` | Usia maksimum riwayat sebelum dibersihkan |

**Mekanisme:**
- Worker harian (mengikuti pola `billing.NewCleanerWorker`) menghapus riwayat lebih tua
  dari N hari.
- Retensi **per-platform** (global setting), dinamis.
- **Penting:** penghapusan retensi bersifat housekeeping — **tidak** memengaruhi hitungan
  limit harian untuk hari berjalan (yang hanya melihat hari ini).

---

## 10. Zona Waktu Reset Harian (N-03) — Dinamis

| Setting | Nilai | Default | Efek |
|---|---|---|---|
| `voice_daily_reset_timezone` | `UTC` \| `Asia/Jakarta` (IANA) | `Asia/Jakarta` | Zona untuk batas "hari" |

**Perilaku:**
- Limit harian dihitung pada hari kalender zona ini.
- `GET /me/voice-config` mengirim `daily_reset_timezone` + `daily_resets_at` agar UI bisa
  menampilkan "reset dalam X jam".
- **Semua** perhitungan harian (gate, counter, tampilan) memakai zona yang sama.

> **Usulan default `Asia/Jakarta` (WIB):** lebih intuitif untuk basis pengguna Indonesia
> ("reset tengah malam WIB") dibanding UTC. Tetap **dinamis**.

---

## 11. Ringkasan

> Kedua jalur **berbeda secara fundamental**: browser = gratis/privat/lokal tapi
> tidak ter-gate fitur; server = berbayar/teraudit/ter-gate. Keduanya kini **diikat**
> oleh (a) **limit harian** yang sama, (b) **VU meter** yang sama, (c) **konfigurasi
> bahasa per-rekaman** yang sama, (d) **zona reset** yang sama. Semua nilai **dinamis**.
> Solusi "smart routing" memakainya secara komplementer.
