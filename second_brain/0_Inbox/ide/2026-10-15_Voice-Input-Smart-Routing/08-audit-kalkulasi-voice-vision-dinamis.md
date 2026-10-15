# Audit Kalkulasi & Logika Voice & Vision — Desain Konfigurasi Dinamis Super Admin

**Tanggal:** 2026-10-15  
**Status:** Selesai Dianalisis & Siap Implementasi  
**Target:** Voice & Vision Pipeline, Metering Heuristics, Dynamic Quota Estimation, Super Admin Control  

---

## 1. Ringkasan Eksekutif

Audit mendalam terhadap implementasi Voice (`ai_handler.go`, `ai_gateway_vision_voice.go`, `voice_handler.go`, `voice_repo.go`) dan Vision (`AnalyzeReceipt`, `snapCategory`, `receiptPrompt`) menemukan bahwa:
1. **Logika inti pengkategorian cerdas Vision sudah sangat baik**: normalisasi nama kategori dari DB live, format tanggal, penanganan refusals, dan fuzzy snapping kategori berjalan akurat.
2. **Logika kuota harian Voice (`voice_requests_per_day`) sudah tangguh**: pencegahan TOCTOU race condition via atomic conditional increment `ConsumeDailyUsage` pada tabel terpisah `voice_daily_usage` (history terhapus tidak merefund limit).
3. **Kelemahan kalkulasi saat ini**: Estimasi reserve kuota token untuk Voice dan Vision masih **flat/statis** (Voice flat 1500-2000 token, Vision flat 2500-5000 token), tidak adaptif terhadap panjang audio maupun ukuran gambar.
4. **Kalkulasi dinamis seperti Chat sepenuhnya dapat diterapkan**: Audio dapat dikalkulasi berdasarkan panjang byte/durasi, dan Vision berdasarkan prompt kategori + image token budget + output headroom.
5. **Dapat dikonfigurasi 100% dari Super Admin**: Seluruh parameter estimasi dan metering dapat didaftarkan ke `setting_definitions` dan diintegrasikan ke kalkulator kuota admin (`/super-admin/kuota`).

---

## 2. Temuan Audit Detail: Voice Input

### 2.1. Alur & Logika (Status: Baik dengan 1 Catatan)
- **Dua Mode Transkripsi**:
  - `server`: Audio dikirim via `POST /ai/voice` (AI Gateway) → memotong `ai_tokens_per_cycle`, `ai_requests_per_cycle`, dan `voice_requests_per_day`.
  - `browser`: Transkripsi di-device (Web Speech API) disimpan via `POST /me/voice-history` → hanya memotong `voice_requests_per_day`.
- **Atomic Daily Claim**: `voiceRepo.ConsumeDailyUsage(userID, day, limit)` mengunci slot harian di tingkat DB (`WHERE used < limit`). Hapus riwayat tidak mengembalikan kuota harian.
- **Catatan Alur Server Mode**: Pada `AIHandler.ProcessVoice`, pemanggilan `h.voiceDaily` dilakukan di `recordVoiceHistory` setelah transkripsi selesai. Hasil boolean `granted` diabaikan karena gateway sudah terlanjur dijalankan (gate awal `limitGate` sudah memfilter di awal). Ini dapat diterima untuk UX agar hasil tidak dibuang, namun jika terjadi race simultan, request kedua tetap terlayani.

### 2.2. Kalkulasi Token Voice (Status: Butuh Dinamisasi)
Saat ini di `ai_handler.go`:
```go
res, rerr := h.ledger.Reserve(c.Request.Context(), userID,
    h.estimateFor(c.Request.Context(), billing.ModalityVoice),
    billing.ModalityVoice, "voice-reserve:"+reqID)
```
- **Masalah**: `h.estimateFor(ModalityVoice)` mengembalikan nilai konstan (default 2000 token).
  - Audio 1 detik (misal ~4 KB WebM) tetap me-reserve 2000 token. User dengan sisa kuota 1000 token akan **ditolak (402 Quota Exceeded)** padahal aslinya hanya butuh ~50 token!
  - Audio 60 detik (~240 KB WebM) juga me-reserve 2000 token.
- **Heuristik Fallback Audio**:
  Di `backend/internal/usecase/ai_gateway_vision_voice.go`:
  ```go
  func estimateAudioSeconds(audio []byte) float64 {
      const bytesPerSecond = 4000.0 // Hardcoded!
      return float64(len(audio)) / bytesPerSecond
  }
  ```
  - Nilai `4000.0` (asumsi WebM Opus ~32 kbps) di-hardcode. Jika audio diunggah dalam format WAV atau MP3 (128 kbps = ~16.000 byte/detik), kalkulasi durasi salah 4x lipat lebih lama.

---

## 3. Temuan Audit Detail: Vision (Receipt OCR)

### 3.1. Logika Kategori & Normalisasi (Status: Sangat Bagus)
- Fungsi `loadReceiptCategories` membaca kategori aktif dari PostgreSQL.
- `receiptPrompt` menyuntikkan daftar kategori riil dan tanggal sekarang ke sistem prompt AI.
- `snapCategory` mencocokkan hasil model:
  1. Exact match (case-insensitive).
  2. Substring match dua arah (misal "Makanan" cocok dengan "Makanan & Minuman").
  3. Fallback cerdas ke kategori berawalan "Lain" atau kategori pertama yang tersedia.
- Hasil: Menghilangkan error "Kategori tidak ditemukan" di sisi client.

### 3.2. Kalkulasi Token Vision (Status: Butuh Dinamisasi)
Saat ini di `ai_handler.go`:
```go
res, rerr := h.ledger.Reserve(c.Request.Context(), userID,
    h.estimateFor(c.Request.Context(), billing.ModalityVision),
    billing.ModalityVision, "vision-reserve:"+reqID)
```
- **Masalah**: `h.estimateFor(ModalityVision)` mengembalikan nilai flat (default 5000 token).
- **Heuristik Fallback Vision**:
  Di `backend/internal/aigateway/metering/estimator.go`:
  ```go
  func (e *Estimator) EstimateVision(promptText, responseText string) domain.TokenUsage {
      u := e.Estimate(promptText, responseText, domain.UseCaseVision)
      cfg := e.Config()
      u.PromptTokens += cfg.ImageTokens // Default: 85 token!
      u.TotalTokens += cfg.ImageTokens
      return u
  }
  ```
  - Nilai default `ImageTokens = 85` terlalu rendah untuk provider modern (OpenAI GPT-4o vision memakan 258 - 1105 token per tile/struk; Gemini 1.5 memakan 258 token per image).
  - Nilai `metering_image_tokens` di database hanya dibaca saat startup di `main.go`, tidak reload saat diubah oleh Super Admin.

---

## 4. Desain Kalkulasi Dinamis (Seperti Chat)

Kalkulasi `ForChat` saat ini menghitung:
$$\text{Tokens}_{\text{chat}} = \text{Overhead} + \left\lceil \frac{\text{Panjang Karakter}}{\text{CharsPerToken}} \right\rceil + \text{Headroom}$$

Kita terapkan formula setara untuk Voice dan Vision:

### 4.1. Formula Dinamis Voice (`ForVoice`)
Sebelum melakukan reservasi kuota ke ledger:
1. Hitung perkiraan durasi audio:
   $$\text{Durasi Detik} = \frac{\text{Panjang Byte Audio}}{\text{BytesPerSecond (Admin)}}$$
2. Hitung estimasi token:
   $$\text{Estimasi Token} = \max\left(\text{VoiceFloor}, \left(\text{Durasi Detik} \times \text{TokensPerSec}\right) + \text{VoiceOverhead} + \text{VoiceOutputHeadroom}\right)$$

*Contoh:*
- Audio 2 detik (8 KB): $(2 \times 25) + 50 + 100 = 200$ token (jauh lebih adil daripada flat 2000 token).
- Audio 30 detik (120 KB): $(30 \times 25) + 50 + 100 = 900$ token.

### 4.2. Formula Dinamis Vision (`ForVision`)
Sebelum melakukan reservasi kuota ke ledger:
1. Hitung token prompt teks (termasuk daftar kategori yang dimuat):
   $$\text{PromptTokens} = \left\lceil \frac{\text{Panjang Karakter Prompt}}{\text{CharsPerToken}} \right\rceil$$
2. Hitung token gambar berdasarkan ukuran file atau resolusi tile:
   $$\text{ImageTokens} = \text{BaseImageTokens} + \begin{cases} 
   \text{TileExtraTokens} & \text{jika FileSize} > 500\text{ KB} \\
   0 & \text{lainnya}
   \end{cases}$$
3. Hitung estimasi token:
   $$\text{Estimasi Token} = \max\left(\text{VisionFloor}, \text{PromptTokens} + \text{ImageTokens} + \text{VisionOutputHeadroom}\right)$$

*Contoh:*
- Struk biasa 200 KB: $250\text{ (prompt)} + 800\text{ (image)} + 400\text{ (JSON output)} = 1450$ token (tidak memboroskan kuota hingga 5000).

---

## 5. Rencana Parameter Konfigurasi Super Admin

Semua parameter berikut didaftarkan ke `setting_definitions` dan tabel `settings`:

| Key Pengaturan | Tipe | Default | Min / Max | Keterangan |
|---|---|---|---|---|
| `ai_estimate_voice_bytes_per_sec` | `int` | `4000` | 1.000 – 64.000 | Perkiraan byte audio per detik (WebM Opus ~4KB/s) |
| `ai_estimate_voice_tokens_per_sec` | `int` | `25` | 1 – 200 | Rasio token audio per detik |
| `ai_estimate_voice_overhead` | `int` | `50` | 0 – 5.000 | Overhead pemrosesan voice |
| `ai_estimate_voice_headroom` | `int` | `100` | 0 – 5.000 | Headroom token output hasil transkripsi |
| `ai_estimate_vision_image_tokens` | `int` | `800` | 50 – 10.000 | Alokasi token untuk 1 gambar struk |
| `ai_estimate_vision_output_headroom`| `int` | `400` | 50 – 5.000 | Headroom token output JSON struk belanja |
| `metering_image_tokens` | `int` | `800` | 50 – 10.000 | Heuristik token gambar saat provider tidak lapor |
| `metering_audio_tokens_per_sec` | `int` | `25` | 1 – 1.000 | Heuristik token audio saat provider tidak lapor |

---

## 6. Integrasi Kalkulator Super Admin (`/super-admin/kuota`)

Pada halaman kalkulator Super Admin, tambahkan panel simulasi modalitas:
1. **Chat**: Input panjang karakter teks → perkiraan token & potongan kuota unit per paket.
2. **Voice**: Input slider durasi audio (1s – 60s) → perkiraan token raw, durasi detik, dan unit kuota terpotong berdasarkan rasio paket.
3. **Vision**: Input jumlah struk dan perkiraan kategori → perkiraan token raw dan unit kuota terpotong berdasarkan rasio paket.

---

## 7. Langkah Implementasi Selanjutnya
1. Tambahkan `ForVoice` dan `ForVision` di `backend/internal/billing/estimate.go`.
2. Pasang pemanggilan dinamis di `ai_handler.go`:
   - `AnalyzeReceipt`: panggil `h.estimates.ForVision(ctx, header.Size, int64(len(prompt)))`.
   - `ProcessVoice`: panggil `h.estimates.ForVoice(ctx, int64(len(audioBytes)))`.
3. Daftarkan kunci pengaturan baru ke `0029_config_catalog.sql` dan allowlist `writableSettings` di `admin/settings.go`.
4. Tambahkan hot-reload metering pada event `config_changed` di `main.go`.
5. Perbarui halaman UI `/super-admin/kuota` agar menampilkan kalkulator dinamis Voice & Vision.
