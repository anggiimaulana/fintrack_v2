# Implementation Rules — P04: Subscription & Token Core

Dokumen ini berisi aturan implementasi yang mengikat (*binding rules*) bagi seluruh pengembang yang berkontribusi pada modul `internal/billing/` dan `internal/entitlement/`. Pelanggaran terhadap aturan ini akan menggagalkan *code review* dan otomatis membatalkan *pull request*.

---

## 1. Aturan Moneter dan Representasi Token (K-P04-01, K-27)
1. **Dilarang Keras Penggunaan Float:**
   - Dilarang mendeklarasikan tipe data `float32` atau `float64` untuk atribut moneter (Rupiah), jumlah token, kuota, atau sisa saldo di layer manapun (database DDL, Go struct, DTO API, dan TypeScript frontend).
   - Seluruh nominal moneter dinyatakan dalam satuan Rupiah penuh sebagai integer 64-bit (`int64` di Go, `BIGINT` di PostgreSQL, `number` di TypeScript dengan batas aman integer).
2. **Pembulatan Kalkulasi:**
   - Perhitungan biaya token dan konversi persentase yang menghasilkan bilangan pecahan WAJIB dibulatkan ke atas (*ceiling*) ke integer IDR terdekat:
     ```go
     // Contoh kalkulasi IDR yang sah
     func CeilDiv(a, b int64) int64 {
         return (a + b - 1) / b
     }
     ```
   - Perhitungan kredit prorata saat upgrade WAJIB dibulatkan ke bawah (*floor*) ke integer IDR terdekat demi asas kehati-hatian finansial (*conservatism principle*).

---

## 2. Aturan Transaksi Ledger Token (K-P04-02, K-P04-03, K-28)
1. **Append-Only Immutability:**
   - Tabel `token_ledger` dan `subscription_events` adalah tabel append-only.
   - Dilarang membuat query SQL `UPDATE` atau `DELETE` pada tabel `token_ledger` dan `subscription_events`. Izin DML di database produksi dibatasi hanya `INSERT` dan `SELECT`.
   - Koreksi kuota atau penyesuaian administratif wajib dilakukan melalui baris baru dengan tipe mutasi `adjust` atau `grant`.
2. **Larangan Long-Running Database Lock:**
   - Dilarang keras menahan transaksi basis data aktif (`sql.Tx`) atau baris lock (`FOR UPDATE`) selama pemanggilan eksternal jaringan (AI Gateway, OpenAI, Midtrans, dll).
   - Alur wajib:
     1. Transaksi DB 1: Buka tx -> Validasi kuota -> Lock baris `token_periods` -> Tambah `reserved_tokens` -> Insert ledger `reserve` -> Commit tx.
     2. Eksekusi Eksternal: Panggil AI Gateway streaming di luar transaksi database.
     3. Transaksi DB 2: Buka tx baru -> Sesuaikan `reserved_tokens` & `used_tokens` -> Insert ledger `commit` / `release` -> Commit tx.
3. **Idempotency Key Mandatori:**
   - Setiap mutasi ledger wajib menyertakan `idempotency_key` unik non-kosong.
   - Kegagalan akibat benturan `idempotency_key` (PostgreSQL error `23505`) wajib ditangani secara idempoten: baca entitas mutasi yang sudah ada dan kembalikan sukses tanpa melakukan mutasi ganda.

---

## 3. Aturan Penegakan Hak Akses & Middleware (K-P04-07, K-P04-08, K-13, K-14)
1. **Backend Merupakan Otoritas Tunggal:**
   - Frontend tidak boleh diasumsikan aman. Seluruh endpoint privat yang melibatkan fitur berbayar atau penambahan entitas wajib diproteksi oleh middleware `RequireFeature` atau `WithinLimit`.
2. **Zero Login/Logout pada Mutasi Paket:**
   - Dilarang menyimpan daftar permissions atau hak akses subscription di dalam payload JWT token pengguna.
   - Hak akses wajib divalidasi per-request melalui resolver cache berversi (`entitlement:u:{userID}:v:{version}`).
3. **Bypass Role Super Admin:**
   - Middleware hak akses wajib mengecek peran pengguna terlebih dahulu: jika `user.role == "super_admin"`, request otomatis dilewatkan (*bypass*) tanpa mengurangi kuota atau memeriksa saklar fitur.
4. **Format Error Standar:**
   - Penolakan fitur mengembalikan HTTP `403 Forbidden` dengan kode error terstruktur:
     ```json
     {
       "success": false,
       "error": "feature_not_in_plan",
       "message": "Fitur ini tidak tersedia pada paket Anda. Silakan tingkatkan paket.",
       "feature_key": "ocr_receipt",
       "upgrade_hint": true,
       "upgrade_url": "/user/settings/subscription"
     }
     ```
   - Penolakan limit kuota mengembalikan HTTP `402 Payment Required`:
     ```json
     {
       "success": false,
       "error": "limit_exceeded",
       "message": "Anda telah mencapai batas paket Anda. Silakan tingkatkan paket.",
       "limit_key": "max_wallets",
       "limit_value": 5,
       "current_usage": 5,
       "upgrade_hint": true,
       "upgrade_url": "/user/settings/subscription"
     }
     ```
   > Catatan: kuota dijawab **402** (bukan 429) agar tidak ambigu dengan rate limiter;
   > `upgrade_url` mengarah ke halaman langganan swalayan `/user/settings/subscription`.

---

## 4. Aturan Kebijakan Downgrade & Soft-Limit (K-P04-09, K-09)
1. **Tidak Ada Penghapusan Data Pengguna:**
   - Dilarang keras menulis kode atau job latar belakang yang menghapus transaksi, anggaran, dompet, atau file tanda terima milik pengguna saat langganan berakhir atau diturunkan ke paket Free.
2. **Penegakan Soft-Limit:**
   - Pengguna yang memiliki entitas melampaui batas kuota baru hanya diblokir pada operasi pembuatan data baru (`POST` / create actions).
   - Seluruh operasi pembacaan data (`GET`), pembaruan transaksi eksisting (`PUT/PATCH`), dan ekspor data tetap diizinkan.

---

## 5. Aturan Caching dan Invalidasi (K-P04-07, K-21)
1. **Pemisahan Cache Level:**
   - L1 Cache: In-memory (TTL 60 detik). Bersifat lokal di instance Go.
   - L2 Cache: Redis (TTL 24 jam). Bersifat terdistribusi.
2. **Kewajiban Sinkronisasi Pub/Sub:**
   - Setiap mutasi langganan (upgrade, downgrade, perpanjangan, pembatalan, perubahan status oleh admin) WAJIB memicu:
     1. `INCR entitlement:ver:{userID}` di Redis.
     2. `PUBLISH entitlement:invalidations {userID}` ke Redis.
   - Dilarang mengandalkan masa habis TTL saja untuk pembaruan hak akses.
3. **Fallback Resilient Saat Redis Down:**
   - Kode cache manager wajib memiliki blok pelindung: jika query ke Redis menghasilkan error koneksi atau timeout > 50ms, sistem harus mengambil data langsung dari PostgreSQL tanpa membuat request HTTP gagal (*graceful degradation*).

---

## 6. Aturan Struktur Kode dan Batasan Arsitektur
1. **Clean Architecture Separation:**
   - Modul `internal/billing/` bertanggung jawab atas status langganan, kuota token, transaksi ledger, dan penetapan harga.
   - Modul `internal/entitlement/` bertanggung jawab atas pembacaan hak akses teragregasi, caching berversi, dan middleware HTTP.
   - Komunikasi antar-modul dilakukan melalui Go Interface, bukan ketergantungan langsung ke struct internal repository.
2. **Pencatatan Audit Sensitif:**
   - Seluruh aksi administratif dan transisi state langganan wajib menghasilkan rekaman log di tabel `subscription_events` dan `audit_logs`.
