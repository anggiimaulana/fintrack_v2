# Implementation Rules — P03 AI Gateway Terpusat

Aturan di bawah ini bersifat **mengikat (binding)** untuk seluruh kode yang ditulis dalam lingkup Fase P03. Setiap Pull Request atau kode baru wajib mematuhi aturan ini tanpa pengecualian.

---

## 1. Arsitektur & Struktur Kode
1. **Dilarang Rewrite Masif pada `ai_usecase.go`**:
   - `internal/usecase/ai_usecase.go` memiliki lebih dari 2.000 baris kode yang mencakup domain logic finansial (agregasi transaksi, kategori, budget, sanitasi prompt RAG).
   - Pekerjaan di `ai_usecase.go` **hanya berupa adapter tipis**: mengganti blok pemanggilan client eksternal dengan pemanggilan interface `aigateway.Service`.
2. **Paket AI Gateway Mandiri (`internal/aigateway/`)**:
   - Seluruh logika koneksi HTTP, parsing stream SSE, failover, balancing, circuit breaker, dan health check wajib berada di dalam `internal/aigateway/`.
   - Modul `internal/aigateway/` tidak boleh mengimpor package `usecase` (mencegah cyclic dependency).
3. **Pemisahan Interface Bersih**:
   - Definisikan interface di `internal/domain/aigateway.go`.
   - Implementasikan abstraksi konkret pada sub-package masing-masing (`client`, `breaker`, `router`, `health`, `metering`).

---

## 2. Ketersediaan & Resiliensi (Circuit Breaker & Failover)
1. **Standar State Machine Circuit Breaker**:
   - Threshold: **5 kegagalan berturut-turut** memicu transisi ke `StateOpen`.
   - Cooldown: **30 detik** saat `StateOpen`. Selama durasi ini, dilarang melakukan HTTP request keluar ke endpoint terkait.
   - Probe: Tepat **1 request** saat `StateHalfOpen`. Jika berhasil, status kembali ke `StateClosed`. Jika gagal, kembali ke `StateOpen` selama 30 detik berikutnya.
2. **Kategori Error Upstream**:
   - Dihitung sebagai kegagalan: HTTP 500, 502, 503, 504, Connection Refused, Network Timeout, HTTP 429 (Rate Limit).
   - Tidak dihitung sebagai kegagalan upstream: HTTP 400 (Bad Request), 401 (Auth Key Invalid saat konfigurasi, ditangani alert terpisah), 404 (Model Not Found).
3. **Failover Berurutan**:
   - Failover wajib mengikuti urutan kolom `priority` (angka terkecil dieksekusi lebih dulu).
   - Seluruh perpindahan failover wajib menghasilkan log dengan level `WARN` berisi endpoint asal yang gagal dan endpoint tujuan failover.

---

## 3. Kontrak Model & Penguncian Dimensi Embedding
1. **Penguncian Dimensi Embedding 768d**:
   - Seluruh konfigurasi route untuk `use_case = 'embedding'` **wajib** menghasilkan vektor dengan panjang tepat 768 elemen (`len(embedding) == 768`).
   - Sistem wajib melakukan validasi runtime:
     ```go
     if len(res.Embedding) != 768 {
         return nil, fmt.Errorf("invalid embedding dimension: expected 768, got %d", len(res.Embedding))
     }
     ```
   - Dilarang mengganti model embedding ke model dengan dimensi berbeda (misal 1536d atau 384d) karena melanggar integritas tabel PostgreSQL `transaction_embeddings vector(768)`.

---

## 4. Keamanan & Kredensial
1. **Enkripsi Kunci Provider (AES-GCM 256)**:
   - Dilarang menyimpan API key provider dalam bentuk plaintext di database `ai_endpoints`.
   - Kolom `api_key_encrypted` wajib dienkripsi dan didekripsi menggunakan AES-GCM 256 melalui fungsi kripto terpusat aplikasi.
2. **Pengarsipan Data Kunci Lama**:
   - Tabel `gemini_api_keys` tidak boleh di-DROP atau dihapus datanya secara fisik.
   - Gunakan soft-archive dengan flag `is_archived = true` dan timestamp `archived_at`.
3. **Penghapusan Total BYOK pada Akses Publik**:
   - Endpoint HTTP untuk penambahan atau modifikasi API key oleh user wajib mengembalikan respons status `410 Gone`.

---

## 5. Token Metering & Pelaporan
1. **Ekstraksi Token Otoritatif**:
   - Gateway wajib mem-parsing field `usage` dari payload respons OpenAI-compatible.
2. **Fallback Estimator Deterministic**:
   - Jika field `usage` tidak tersedia dari provider, gunakan heuristik:
     `estimated_tokens = int(math.Ceil(float64(len(text)) / 4.0 * 1.1))`
   - Angka token yang dikembalikan tidak boleh bernilai negatif atau 0 untuk request yang menghasilkan teks.

---

## 6. Zero Downtime & Rollout
1. **Feature Flag Guard**:
   - Implementasi gateway dikontrol melalui environment flag atau database flag: `AI_GATEWAY_CENTRALIZED_ENABLED`.
   - Saat rollout pertama kali, sistem mampu beralih kembali ke mode lama tanpa migrasi ulang skema.
2. **Kesesuaian Port Lokal**:
   - Endpoint primer default disetel ke 9router lokal: `http://localhost:20128/v1`.
