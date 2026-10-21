# 09 — Strategi Pengujian, Keamanan & Mitigasi Risiko

**Topik:** Matriks Pengujian QA, Pengujian Konkurensi Saldo, Kepatuhan Keamanan UU PDP, & Rencana Kontinjensi  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Matriks Pengujian Kualitas (QA Testing Matrix)

Pengujian fitur integrasi perbankan dirancang berlapis dari level unit, integrasi database, hingga konkurensi data.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          PIRAMIDA PENGUJIAN FITUR                           │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                               [End-to-End]
                 Upload PDF Mutasi -> Saldo Ter-update ->
                     Sankey Tergambar -> AI RAG Akurat
                                    │
                                    ▼
                             [Integrasi & Concurrency]
                   - Concurrent Batch Commits (Row Locks)
                   - Batch Rollback (Saldo Reversion)
                   - Entitlement Paywall (403 Verification)
                                    │
                                    ▼
                                  [Unit]
                   - 8 Parser Bank & E-Wallet (Fixtures)
                   - SHA-256 Fingerprint Determinism
                   - Stage 1 Regex Normalizer & Noise Stripper
                   - Smart Transfer Pairing Window (<= 36 Jam)
```

### 1.1 Rincian Pengujian Unit (`backend/internal/ingestion/`)
| Komponen yang Diuji | Skenario Uji | Kriteria Sukses |
|---|---|---|
| **BCA CSV & PDF Parser** | Memproses file mutasi KlikBCA & myBCA dengan karakter aneh, koma Indonesia, dan multi-line text. | Ekstraksi 100% nominal, tanggal ISO valid, dan penandaan `DB`/`CR` tepat. |
| **GoPay / DANA Parser** | Memproses e-statement bulanan dengan transaksi GoFood, QRIS, dan biaya top-up. | Memilah transaksi fee secara terpisah dari transaksi utama. |
| **Fingerprint Generator** | Menguji dua string deskripsi yang identik secara makna namun beda spasi/huruf besar-kecil. | Menghasilkan hash SHA-256 yang identik (case & whitespace insensitive). |
| **Transfer Pairing Logic** | Menguji 2 mutasi: Pengeluaran BCA Rp 100.000 pada 10 Okt 08:00 dan Pemasukan GoPay Rp 100.000 pada 10 Okt 08:05. | Sistem otomatis memasangkan keduanya sebagai `transfer` dan menyematkan `transfer_pair_id`. |
| **Pairing Negative Test** | Nominal sama Rp 100.000 tetapi selisih waktu 48 jam, atau arah transaksi sama-sama pengeluaran. | Tidak dipasangkan sebagai transfer internal (tetap transaksi independen). |

### 1.2 Pengujian Konkurensi & Integritas Saldo (`concurrency_test.go`)
- **Skenario:** Menjalankan 5 goroutine simultan yang masing-masing mengunggah berkas mutasi berbeda pada akun bank yang sama dengan saldo awal Rp 1.000.000.
- **Verifikasi:** 
  - Saldo akhir `financial_accounts.current_balance` harus persis sama dengan:
    $$\text{Saldo Akhir} = 1.000.000 + \sum \Delta \text{Balance Seluruh Baris}$$
  - Tidak terjadi *deadlock* pada database PostgreSQL.
  - Kunci `SELECT ... FOR UPDATE` berhasil menahan request concurrent hingga transaksi pertama selesai di-commit.

### 1.3 Pengujian Batch Rollback (Undo Impor)
- **Skenario:** Mengimpor file mutasi berisi 50 transaksi, lalu memicu endpoint rollback.
- **Verifikasi:**
  - 50 transaksi ditandai `is_deleted = true`.
  - Saldo akun kembali persis ke saldo sebelum file diimpor.
  - Seluruh vector embedding terkait terhapus dari `transaction_embeddings`.
  - Percobaan rollback kedua pada batch yang sama mengembalikan error `400 Bad Request` (`ErrBatchAlreadyRolledBack`).

---

## 2. Standar Keamanan & Perlindungan Data Pribadi (UU PDP)

Data finansial perbankan adalah data pribadi yang bersifat spesifik menurut UU No. 27 Tahun 2022. Standar keamanan berikut wajib diterapkan:

### 2.1 Masking Nomor Akun & Rekening
- FinTrack v2 **dilarang keras** menyimpan nomor rekening, nomor kartu debit, atau nomor telepon akun secara lengkap.
- Fungsi sanitasi mask:
  ```go
  func MaskAccountNumber(raw string) string {
      cleaned := strings.ReplaceAll(raw, " ", "")
      if len(cleaned) <= 4 { return "****" }
      return "****" + cleaned[len(cleaned)-4:]
  }
  ```

### 2.2 Penanganan Berkas Sementara (Zero Disk Footprint)
- Berkas mutasi (PDF/CSV/XLSX) yang diunggah pengguna diproses langsung secara *in-memory* (`bytes.Reader` / streaming `multipart.File`).
- Sistem **tidak menyimpan file fisik** mutasi di hard disk server atau public cloud bucket.
- Setelah parsing selesai dan batch disimpan ke database, buffer memori langsung dibebaskan (*garbage collected*).

### 2.3 Sanitasi Validasi Berkas Unggahan
- **Batas Ukuran:** Maksimal 10 MB per berkas.
- **Validasi Magic Bytes:** Memeriksa magic number header berkas (misal `%PDF-` untuk berkas PDF) sebelum parsing dijalankan untuk mencegah file executable berbahaya yang disamarkan (*polyglot attack*).
- **Proteksi Denial of Service (ReDoS):** Seluruh regex pada Stage 1 Normalizer diuji dengan timeout batas eksekusi maksimum 50 milidetik per baris.

---

## 3. Matriks Risiko & Mitigasi Operasional

| Risiko Potensial | Tingkat Dampak | Probabilitas | Rencana Mitigasi Sistem |
|---|---|---|---|
| **Perubahan Format PDF oleh Bank** (Misal myBCA memperbarui layout font/tabel). | Tinggi | Sedang | 1. Sediakan **Generic CSV Parser** sebagai fallback instan.<br>2. Unit test berkala dengan file dummy bulanan.<br>3. Pustaka parser modular: perbaikan layout hanya mengubah satu file adapter tanpa menyentuh modul lain. |
| **Kehabisan Kuota Token AI Pengguna** di tengah-tengah pemrosesan mutasi ratusan baris. | Rendah | Tinggi | Sistem menerapkan **Graceful Fallback**: transaksi tetap berhasil diimpor dengan kategori *"Uncategorized"*. Pengguna tidak pernah kehilangan data. |
| **Upload Ganda Berkas yang Sama** oleh pengguna yang lupa. | Sedang | Tinggi | Sistem menggunakan **SHA-256 Fingerprint Deduplication**. Seluruh baris yang identik otomatis di-skip dengan pesan informatif: *"0 transaksi baru, 45 duplikat diabaikan"*. |
| **Salah Unggah Berkas** (Pengguna mengunggah mutasi rekening A ke rekening B). | Sedang | Sedang | Fitur **One-Click Batch Rollback** memungkinkan pengguna membatalkan impor seketika dan mengembalikan saldo rekening ke kondisi semula. |
| **Percobaan Akses Lintas Tenant (IDOR)** pada akun atau batch impor. | Kritis | Rendah | Seluruh query SQL repository wajib menyertakan filter `WHERE user_id = $1` yang diambil dari konteks JWT auth yang tervalidasi. |
