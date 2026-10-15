# 07 — Risiko, Edge Case & Mitigasi

| Field | Isi |
|---|---|
| **Dokumen** | Analisis risiko, edge case, kompatibilitas, keputusan final |
| **Tanggal** | 2026-10-15 (revisi 2) |
| **Status** | Selesai |

---

## 1. Risiko Teknis

| ID | Risiko | Dampak | Kemungkinan | Mitigasi |
|---|---|---|---|---|
| **R-01** | `getUserMedia` butuh HTTPS → dev/prod salah konfigurasi | Mode server gagal | Sedang | Dokumentasi; fallback browser; pesan jelas |
| **R-02** | Izin mikrofon ditolak | Tidak bisa rekam | Tinggi | Deteksi `NotAllowedError`; pesan panduan; **jangan** fallback |
| **R-03** | `MediaRecorder` MIME tidak didukung (Safari) | Blob kosong | Rendah | `isTypeSupported` + fallback default; validasi blob kosong |
| **R-04** | Kebocoran track (mic nyala terus) | Privasi + UX buruk | Sedang | **Wajib** stop semua track pada stop/cancel/error/unmount |
| **R-05** | Audio > 5MB | 400 dari backend | Rendah | Cek panjang base64 pra-kirim; auto-stop dinamis |
| **R-06** | Fallback loop | UI hang | Rendah | Flag `fallbackUsedRef` sekali per sesi |
| **R-07** | Race kuota habis antara cek & kirim | 402 setelah bicara | Sedang | Backend otoritatif → fallback |
| **R-08** | `usage_changed` lag (SSE) | Bar telat | Rendah | Backend tetap menolak; fallback bekerja |
| **R-09** | Firefox tanpa STT + kuota/harian habis | Tidak ada jalur | Sedang | Mode `unavailable`/`daily_limit` + pesan jelas |
| **R-10** | Dua tab merekam bersamaan | Salah satu 429 | Rendah | Ledger reserve atomik; limit harian dicek backend |
| **R-11** | **Kebocoran `AudioContext`** (VU meter) | Memory leak | Sedang | **Wajib** `close()` + `cancelAnimationFrame` saat stop/unmount |
| **R-12** | **Riwayat gagal dicatat** (mode browser) | Limit harian tidak terhitung | Sedang | Best-effort + backend tetap otoritatif; retry halus |
| **R-13** | **Admin salah set durasi/bahasa** | UX rusak | Rendah | Validasi rentang (1..600); enum bahasa; audit log |
| **R-14** | **Limit harian direset oleh user** (hapus riwayat) | Melewati limit | Sedang | Hitung dari `created_at` (append-only); hapus riwayat **tidak** mengurangi counter |
| **R-15** | **Migrasi tabel baru** gagal di produksi | Fitur mati | Rendah | Migrasi idempoten `IF NOT EXISTS`; zero-downtime |
| **R-16** | **Zona waktu salah** → reset harian tidak akurat | Bingung/keluhan | Sedang | Enum tervalidasi; tampilkan `daily_resets_at`; default `Asia/Jakarta` (N-03) |
| **R-17** | **Retensi menghapus riwayat terlalu agresif** | Kehilangan data user | Sedang | Default 90 hari; rentang 1..3650; hanya housekeeping (N-01) |
| **R-18** | **Bahasa tidak didukung provider** | Transkripsi gagal/salah | Rendah | Batasi `voice_languages` ke yang didukung; fallback ke default |
| **R-19** | **Edit transcript** membingungkan audit | Riwayat ≠ pesan chat | Rendah | Riwayat simpan teks **asli**; dokumentasikan (N-04) |
| **R-20** | **Terlalu banyak knob dinamis** disalah-set | UX rusak | Rendah | Validasi rentang/enum; audit log; preview di admin UI |

---

## 2. Risiko Privasi

| ID | Risiko | Mitigasi |
|---|---|---|
| **P-01** | Audio dikirim ke server tanpa user sadar | Badge mode ("AI") **wajib**; hint jelas |
| **P-02** | Audio/transcript di log | **Jangan** log audio/base64; transcript hanya bila perlu (hindari PII) |
| **P-03** | Mode browser mengirim audio ke vendor | Hint "Mode dasar" menjelaskan audio tidak ke server kita |
| **P-04** | **Riwayat transkripsi tersimpan di server** (Q-04) | Ter-scope user; user bisa hapus; dokumentasikan retensi |

---

## 3. Risiko Biaya

| ID | Risiko | Mitigasi |
|---|---|---|
| **C-01** | User Free menyalahgunakan mode server | Gate `voice_input` + limit request/token + **limit harian** |
| **C-02** | Estimasi token voice tidak akurat | `EstimateAudio`; commit token aktual bila provider melaporkan |
| **C-03** | Overrun token | Ledger clamp ke sisa kuota (sudah ada) |
| **C-04** | **Mode browser tak terbatas** (tanpa limit harian) | **Ditutup** oleh `voice_requests_per_day` (Q-03) |

---

## 4. Edge Case Detail

### 4.1 Auto-stop & Durasi (dinamis)

| Kasus | Perilaku |
|---|---|
| User lupa berhenti | Auto-stop pada `voice_max_duration_seconds` (dinamis) |
| Rekaman sangat pendek (<100ms) | Kirim apa adanya; transcript kosong → tidak `onTranscript` |
| Admin ubah durasi saat merekam | Berlaku pada rekaman **berikutnya** (bukan yang sedang jalan) |

### 4.2 Limit Harian

| Kasus | Perilaku |
|---|---|
| Habis sebelum rekam | Mode `daily_limit`; mic disabled + hint |
| Habis saat rekam (server) | 402 → blokir + hint |
| Habis saat rekam (browser) | `POST /me/voice-history` → 402 → berhenti |
| Reset pada tengah malam **zona dinamis** (N-03) | Counter kembali 0; mode kembali normal |
| User hapus riwayat | **Counter harian tidak berubah** (dihitung dari `created_at`, bukan jumlah baris saat ini) |

> **Penting (R-14):** hitung harian harus **append-only** (dari `created_at`), bukan
> "jumlah baris sekarang", agar penghapusan riwayat tidak melewati limit.

### 4.3 Siklus Hidup

| Kasus | Perilaku |
|---|---|
| Unmount saat merekam | Cleanup: stop recorder + track + `AudioContext` |
| Logout saat merekam | Hentikan; jangan kirim ke endpoint terproteksi |
| Tab background | Rekaman tetap jalan; auto-stop berlaku |

### 4.4 Konten & Bahasa

| Kasus | Perilaku |
|---|---|
| `id_only` | Paksa `id-ID`; tidak ada pemilih |
| `multi` | Pemilih bahasa tampil; diteruskan ke STT/server |
| Bahasa campur | Bergantung provider/browser |

---

## 5. Kompatibilitas Browser

| Fitur | Chrome | Edge | Safari | Firefox |
|---|---|---|---|---|
| `SpeechRecognition` | ✅ | ✅ | ✅ (webkit) | ❌ |
| `MediaRecorder` | ✅ | ✅ | ✅ | ✅ |
| `getUserMedia` | ✅ | ✅ | ✅ | ✅ |
| `AudioContext`/`AnalyserNode` | ✅ | ✅ | ✅ | ✅ |
| `audio/webm;codecs=opus` | ✅ | ✅ | ⚠️ (mp4/aac) | ✅ |

**Implikasi:**
- Safari: MIME default berbeda → `isTypeSupported` memilih yang cocok.
- Firefox: hanya mode server; bila habis → `daily_limit`/`unavailable` + pesan.

---

## 6. Keputusan Terkunci (FINAL — hasil jawaban open question)

| Kode | Keputusan | Sumber |
|---|---|---|
| **K-VOI-01** | Backend otoritatif; klien pemilih jalur | — |
| **K-VOI-02** | Fail-safe: kegagalan server → fallback browser | — |
| **K-VOI-03** | Fallback tidak mengubah hasil: transcript tetap teks | — |
| **K-VOI-04** | Kuota habis → downgrade ke browser (bukan error keras) | — |
| **K-VOI-05** | Audio ke server hanya bila fitur ✅ **dan** kuota ✅ **dan** recorder ✅ | — |
| **K-VOI-06** | Setiap transkripsi server settle ledger + emit `usage_changed` | — |
| **K-VOI-07** | Mode selalu terlihat ke user (badge) | — |
| **K-VOI-08** | Durasi maks default **60s**, **dinamis** via Super Admin | Q-01 |
| **K-VOI-09** | **Limit harian** `voice_requests_per_day` per paket, dinamis, **berlaku kedua mode** | Q-03 |
| **K-VOI-10** | **Riwayat** di server (source of truth) + cache lokal | Q-04 |
| **K-VOI-11** | **VU meter wajib** saat merekam | Q-05 |
| **K-VOI-12** | Bahasa default `id`; **multi-bahasa dinamis** (`id_only` \| `multi`) | Q-06 |
| **K-VOI-13** | Halaman `/user/voice` **digabung ke chat**; rute standalone dihapus | Q-07 |
| **K-VOI-14** | Fallback **wajib**; hint upgrade **tidak memaksa** (dismissible) | Q-02, Q-08 |
| **K-VOI-15** | **Retensi riwayat** dinamis (`voice_history_retention_days`, default 90) | N-01 |
| **K-VOI-16** | Pemilihan bahasa **per-rekaman** saat `multi` | N-02 |
| **K-VOI-17** | Reset limit harian mengikuti **zona dinamis** (`voice_daily_reset_timezone`) | N-03 |
| **K-VOI-18** | Transcript **dapat diedit** sebelum kirim (dinamis `voice_allow_edit_before_send`) | N-04 |
| **K-VOI-19** | **Semua** knob voice dinamis; tidak ada hardcode | N-01..N-05 |
| **K-VOI-20** | Export riwayat voice ditunda ke fase lanjut | N-05 |

---

## 7. Rencana Rollback

| Skenario | Rollback |
|---|---|
| Smart routing regresi | Kembalikan `VoiceInput` ke browser-only (perubahan aditif) |
| Kuota voice membengkak | Matikan mode server via env `NEXT_PUBLIC_VOICE_SERVER_ENABLED=false` |
| Limit harian terlalu ketat | Naikkan `voice_requests_per_day` via admin (tanpa redeploy) |
| Provider voice bermasalah | Kill switch gateway (`UseCaseVoice`) → fallback otomatis |
| Setting salah | Revert via admin UI (dinamis) |
| Migrasi bermasalah | Tabel aditif; `DROP` aman (tidak ada dependensi kritis) |

---

## 8. Keputusan Lanjutan (FINAL — N-01 … N-05)

| ID | Keputusan | Dinamis? | Key |
|---|---|---|---|
| **N-01** | Retensi riwayat **90 hari** | ✅ | `voice_history_retention_days` |
| **N-02** | Bahasa **per-rekaman** saat `multi` | ✅ | `voice_language_mode` + `voice_languages` |
| **N-03** | Reset harian zona **server** (usulan `Asia/Jakarta`) | ✅ | `voice_daily_reset_timezone` |
| **N-04** | Transcript **dapat diedit** sebelum kirim | ✅ | `voice_allow_edit_before_send` |
| **N-05** | **Export riwayat** → fase lanjut | — | (belum ada) |

> **Semua** (kecuali N-05) dinamis & dapat dikonfigurasi Super Admin tanpa redeploy.
> Katalog lengkap di README §4.1.

### 8.1 Catatan N-04 (klarifikasi "bukan langsung gas ke `/ai/voice`")

Transkripsi (`/ai/voice` atau Web Speech) **hanya mengubah audio → teks**. Pengiriman ke
chat **selalu** lewat `POST /ai/chat`. Jadi:
- Tidak ada jalur "audio → langsung jawaban chat".
- Edit-sebelum-kirim diterapkan pada **kedua** mode.
- Riwayat voice menyimpan **teks asli**; versi editan adalah pesan chat.
- Detail lengkap di dok 03 §6.4.1.

---

## 9. Ringkasan Risiko Tertinggi

> **Tujuh hal yang wajib benar sebelum rilis:**
> 1. **Kebocoran track mikrofon** (R-04/P-01) — uji manual wajib.
> 2. **Kebocoran `AudioContext`** (R-11) — uji buka/tutup berulang.
> 3. **Fallback tidak loop** (R-06) — uji unit + DOM.
> 4. **Limit harian append-only** (R-14) — jangan hitung dari baris saat ini.
> 5. **Firefox tidak crash** (R-09) — pesan jelas, bukan dead-end.
> 6. **Zona reset benar** (R-16) — konsisten di gate & tampilan.
> 7. **Semua knob dinamis tervalidasi** (R-20) — rentang/enum + audit.
