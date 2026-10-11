# Security Review & Postur Kepatuhan PCI-DSS SAQ A - FinTrack v2

Dokumen ini memaparkan audit keamanan siber menyeluruh sistem FinTrack v2, mencakup mekanisme enkripsi data sensitif at-rest, rotasi rahasia zero-downtime, verifikasi kriptografis webhook pembayaran, mitigasi OWASP API Security Top 10, serta pemenuhan postur PCI-DSS SAQ A.

---

## 1. Arsitektur Enkripsi At-Rest (AES-256-GCM)

### 1.1 Standar & Spesifikasi Kriptografi
Seluruh secret sensitif (kunci endpoint AI upstream, token rahasia webhook gateway, token integrasi pihak ketiga) yang disimpan dalam basis data PostgreSQL dienkripsi secara simetris menggunakan algoritma **AES-256-GCM** (Galois/Counter Mode).
- **Cipher**: AES dengan panjang kunci 256-bit (32 byte).
- **Inisialisasi Vektor (IV / Nonce)**: 12-byte acak kriptografis (`crypto/rand`) unik per setiap operasi enkripsi.
- **Authentication Tag**: 16-byte tag integritas bawaan GCM untuk menangkal manipulasi ciphertext (*tamper resistance*).
- **Format Penyimpanan DB**: `base64( nonce[12] || ciphertext[N] || tag[16] )`.

### 1.2 Implementasi Kode Go (`internal/pkg/crypto/encrypt.go`)
```go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrDecryptionFailed = errors.New("crypto: autentikasi ciphertext gagal atau korup")
	ErrInvalidKeySize   = errors.New("crypto: panjang kunci master wajib 32 byte")
)

// EncryptSecret mengenkripsi plaintext menggunakan AES-256-GCM dengan nonce acak
func EncryptSecret(plaintext []byte, masterKey []byte) (string, error) {
	if len(masterKey) != 32 {
		return "", ErrInvalidKeySize
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Menempelkan ciphertext + tag di belakang nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret mendekripsi string base64 dan memverifikasi integritas data
func DecryptSecret(encodedCiphertext string, masterKey []byte) ([]byte, error) {
	if len(masterKey) != 32 {
		return nil, ErrInvalidKeySize
	}

	raw, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, ErrDecryptionFailed
	}

	nonce, ciphertextWithTag := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextWithTag, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}
```

---

## 2. Prosedur Rotasi Secret Zero-Downtime

### 2.1 Skema Dual-Key Decryption
Aplikasi mendukung dua variabel lingkungan kunci master secara bersamaan:
- `APP_ENCRYPTION_MASTER_KEY_CURRENT`: Kunci utama untuk operasi enkripsi data baru.
- `APP_ENCRYPTION_MASTER_KEY_PREVIOUS`: Kunci sekunder untuk membaca data lama yang belum dire-enkripsi.

Alur pembacaan data:
1. Dekripsi dicoba menggunakan `CURRENT`.
2. Jika gagal karena tag mismatch, sistem otomatis mencoba dekripsi dengan `PREVIOUS`.
3. Jika berhasil dengan `PREVIOUS`, worker latar belakang menandai baris tersebut untuk dire-enkripsi ulang ke `CURRENT`.

### 2.2 CLI Migrasi Re-Enkripsi Data
Tersedia perintah admin CLI internal `fintrack-admin secrets re-encrypt` yang mengeksekusi re-enkripsi seluruh kolom terenkripsi pada tabel `ai_endpoints` dan `payment_configs` secara bertahap (batch 100 baris per transaksi) tanpa mengunci tabel.

---

## 3. Verifikasi Webhook Pembayaran Kriptografis

### 3.1 Webhook Midtrans (SHA-512)
- Rumus kalkulasi tanda tangan: `SHA512(order_id + status_code + gross_amount + ServerKey)`.
- Verifikasi dilakukan dengan mencocokkan hasil kalkulasi hash terhadap nilai header/payload `signature_key` menggunakan pembanding waktu-konstan (`subtle.ConstantTimeCompare`).

### 3.2 Webhook Xendit (Callback Token Validation)
- Setiap request webhook Xendit menyertakan header `x-callback-token`.
- Backend membandingkan token ini dengan rahasia yang terdaftar di konfigurasi sistem:
```go
if subtle.ConstantTimeCompare([]byte(incomingToken), []byte(configuredToken)) != 1 {
    auditLogSecurityBreach("XENDIT_WEBHOOK_UNAUTHORIZED_TOKEN", c.ClientIP())
    c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized webhook signature"})
    return
}
```

---

## 4. Postur Kepatuhan PCI-DSS SAQ A (Self-Assessment Questionnaire A)

### 4.1 Definisi Cakupan & Alur Transaksi
FinTrack v2 mematuhi kualifikasi **PCI-DSS v4.0 SAQ A** untuk *Card-not-present merchant*:
1. Semua pemrosesan pembayaran kartu kredit sepenuhnya dialihkan (*outsourced*) ke Payment Gateway berlisensi PCI-DSS Level 1 (Midtrans & Xendit).
2. Tidak ada komponen sistem FinTrack v2 (frontend web Next.js maupun backend Go) yang memproses, menyimpan, atau mentransmisikan data kartu kredit (*Cardholder Data* atau *Sensitive Authentication Data*).
3. Pemasukan data kartu kredit pengguna terjadi secara eksklusif pada halaman checkout redirect resmi Midtrans/Xendit atau modal Snap yang terisolasi dalam iframe aman dari domain gateway.

### 4.2 Matriks Kontrol PCI-DSS SAQ A FinTrack v2
| No. Kontrol | Persyaratan PCI-DSS SAQ A | Implementasi FinTrack v2 | Status |
|---|---|---|:---:|
| Req 2.2 | Sistem dikonfigurasi aman tanpa default password | Seluruh kredensial default dihapus; password DB menggunakan string 32-karakter acak. | COMPLIANT |
| Req 6.4.3 | Seluruh skrip halaman pembayaran terotorisasi | Header CSP (`Content-Security-Policy`) membatasi `script-src` hanya ke domain resmi payment gateway (`app.midtrans.com`, `checkout.xendit.co`). | COMPLIANT |
| Req 8.3 | Autentikasi multi-faktor untuk akses administratif | Panel Super Admin FinTrack dilindungi sesi terotentikasi ketat dan audit log IP. | COMPLIANT |
| Req 10.2 | Pencatatan log akses dan event keamanan | Seluruh upaya webhook unauthorized dan transaksi gagal dicatat append-only di `audit_logs`. | COMPLIANT |
| Req 11.3 | Uji penetrasi dan vulnerability scanning | Pemindaian berkala dependensi Go (`govulncheck`) dan frontend (`pnpm audit`). | COMPLIANT |

---

## 5. Prinsip Least Privilege & Pengerasan Lingkungan (Hardening)

### 5.1 Hak Akses Role Database PostgreSQL
- User aplikasi `fintrack_app` dibatasi hanya memiliki hak:
  ```sql
  GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO fintrack_app;
  REVOKE DROP, TRUNCATE, ALTER ON ALL TABLES IN SCHEMA public FROM fintrack_app;
  ```
- Eksekusi DDL migrasi skema dipisahkan murni melalui user deployment khusus `fintrack_migrator`.

### 5.2 Pengerasan Kontainer Docker
- Image backend Go menggunakan multi-stage build minimal berbasis Alpine/Distroless.
- Runtime user disetel non-root: `USER 10001:10001`.
- Kemampuan Linux dinonaktifkan: `cap_drop: [ALL]`.
- Pencegahan eskalasi hak akses: `security_opt: [no-new-privileges:true]`.

---

## 6. Mitigasi OWASP API Security Top 10

1. **Broken Object Level Authorization (BOLA)**: Setiap query pengambilan atau modifikasi data keuangan selalu mencantumkan klausul `WHERE user_id = :auth_user_id` dari konteks sesi JWT yang terverifikasi.
2. **Broken Authentication**: Token JWT menggunakan algoritma Ed25519/HMAC-SHA256 dengan waktu kedaluwarsa pendek (15 menit) dan refresh token berputar (*rotating refresh token*).
3. **Broken Object Property Level Authorization**: Model request binding di-whitelist secara eksplisit menggunakan Go struct tag `binding:"required"`, mencegah eksploitasi mass assignment.
4. **Unrestricted Resource Consumption**: Rate limiting bertingkat dipasang di level Nginx/Traefik reverse proxy dan middleware Redis sliding-window per IP dan per User ID.
5. **Server-Side Request Forgery (SSRF)**: Panggilan AI Gateway upstream dibatasi murni pada daftar IP/host yang telah di-whitelist di tabel `ai_endpoints`, memblokir request ke metadata IP internal cloud (`169.254.169.254` atau subnet privat `10.0.0.0/8`).
