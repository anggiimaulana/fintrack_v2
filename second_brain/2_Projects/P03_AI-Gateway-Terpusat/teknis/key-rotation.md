# Rotasi Kunci, Enkripsi, dan Deprecate BYOK

Dokumen ini mendefinisikan standar keamanan enkripsi kredensial API key provider terpusat, pengarsipan kunci user lama (BYOK), dan prosedur rotasi kunci tanpa downtime.

---

## 1. Penghentian Total Fitur BYOK (Bring Your Own Key)

Sesuai keputusan **K-16**, fitur di mana pengguna akhir (user) memasukkan API key pribadi dihapus sepenuhnya dari FinTrack v2.

### 1.1. Langkah Pengarsipan Database
Data pada tabel `gemini_api_keys` tidak dihapus secara destruktif (`DROP TABLE` atau `DELETE FROM`) guna memenuhi kepatuhan audit dan riwayat sistem (UU PDP No. 27/2022).
Tabel diubah statusnya menjadi arsip melalui migrasi `0006_ai_gateway.sql`:

```sql
-- Tambah kolom audit arsip
ALTER TABLE gemini_api_keys 
ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ DEFAULT NOW();

-- Tandai seluruh kunci lama sebagai terarsip
UPDATE gemini_api_keys 
SET is_archived = TRUE, archived_at = NOW() 
WHERE is_archived IS FALSE;
```

### 1.2. Deprecasi Endpoint HTTP Pengguna
Rute HTTP lama di backend yang mengelola API key per-user (`/api/v1/gemini-keys`) dinonaktifkan:
- Request `POST`, `PUT`, `DELETE` mengembalikan status `HTTP 410 Gone` dengan payload JSON:
  ```json
  {
    "status": 410,
    "error": "BYOK_DEPRECATED",
    "message": "Fitur API Key mandiri telah dihentikan. FinTrack v2 kini menggunakan AI Gateway terpusat."
  }
  ```
- Request `GET` mengembalikan array kosong `[]` agar klien frontend lama tidak mengalami error parsing.

### 1.3. Pembersihan Frontend
- Navigasi link "API Keys" pada sidebar menu pengaturan dihapus.
- Halaman route frontend `/settings/api-keys` dialihkan (redirect 307) menuju `/settings/general`.
- Semua komponen modal input kunci di-remove dari codebase frontend.

---

## 2. Standar Enkripsi Kredensial Terpusat (AES-256-GCM)

API Key upstream provider (OpenAI, Anthropic, Gemini, Groq, dsb) disimpan pada kolom `api_key_encrypted BYTEA` di tabel `ai_endpoints`.

### 2.1. Spesifikasi Kriptografi
- **Algoritma**: AES-256-GCM (Galois/Counter Mode).
- **Master Key**: Berasal dari environment variable `ENCRYPTION_KEY` berukuran 32 bytes (256 bits).
- **Nonce / IV**: Berukuran 12 bytes, digenerasi menggunakan `crypto/rand` unik pada setiap operasi enkripsi.
- **Format Penyimpanan**: Nonce digabungkan di depan ciphertext sebelum disimpan ke database:
  `[ 12 Bytes Nonce ] + [ N Bytes Ciphertext + Tag Auth 16 Bytes ]`

### 2.2. Implementasi Kripto Helper Go

```go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// EncryptKey mengenkripsi string plaintext API key menggunakan AES-GCM
func EncryptKey(plaintext string, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// DecryptKey mendekripsi ciphertext kembali menjadi plaintext
func DecryptKey(ciphertext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
```

---

## 3. Prosedur Rotasi Kunci Tanpa Downtime

Rotasi kunci upstream provider (misal kunci OpenAI utama kedaluwarsa atau harus diganti berkala) dilakukan dengan alur zero downtime:

```
[ Step 1: Input Kunci Baru ]
Super admin mendaftarkan endpoint/kunci baru dengan nama unik (misal openai-v2) 
dengan status='active', priority sama atau lebih rendah.
         │
         ▼
[ Step 2: Health Verification ]
Health checker memvalidasi kunci baru menghasilkan HTTP 200 via test ping probe.
         │
         ▼
[ Step 3: Route Switch ]
Super admin mengalihkan foreign key endpoint_id di ai_routes ke ID endpoint baru.
In-memory cache reload seketika via signal pub/sub.
         │
         ▼
[ Step 4: Deaktivasi Kunci Lama ]
Endpoint lama diubah statusnya menjadi status='inactive'.
Trafik in-flight selesai diproses tanpa terputus.
```

Keuntungan pendekatan ini:
1. Tidak ada hard restart pada backend FinTrack v2.
2. Jika kunci baru salah atau belum diisi saldo, rute tidak akan dialihkan karena verifikasi health check gagal.
3. Kunci lama dapat diaktifkan kembali dalam hitungan detik jika terjadi anomali pada kunci baru.
