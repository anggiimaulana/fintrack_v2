# Kepatuhan Regulasi UU PDP No. 27/2022 & Rekonsiliasi Retensi Pajak - FinTrack v2

Dokumen ini menetapkan kerangka kerja kepatuhan hukum FinTrack v2 terhadap Undang-Undang Perlindungan Data Pribadi (UU No. 27 Tahun 2022) Republik Indonesia, hak-hak subjek data, serta resolusi arsitektural atas ketegangan hukum antara kewajiban retensi data perpajakan 10 tahun dan hak penghapusan data pribadi (*Right to Erasure*).

---

## 1. Landasan Hukum & Prinsip Pemrosesan Data Pribadi

### 1.1 Kepatuhan UU PDP No. 27/2022
Sebagai Pengendali Data Pribadi (*Data Controller*), FinTrack v2 mematuhi seluruh asas pemrosesan data pribadi (Pasal 16 UU PDP):
1. **Tujuan Spesifik & Sah**: Data keuangan dan profil dikumpulkan murni untuk penyediaan layanan manajemen keuangan pribadi dan pencatatan transaksi pengguna.
2. **Keterbatasan Pengumpulan (Data Minimization)**: Hanya data yang relevan secara fungsional yang disimpan. FinTrack v2 tidak meminta izin akses perangkat yang tidak diperlukan (seperti kontak buku telepon atau lokasi latar belakang).
3. **Akurasi & Integritas**: Pengguna memiliki akses mandiri untuk memperbarui profil dan catatan keuangan secara real-time.
4. **Keamanan Data**: Penggunaan enkripsi kuat pada data at-rest (AES-256-GCM) dan in-transit (TLS 1.3).
5. **Pembatasan Retensi**: Data yang telah kedaluwarsa atau tidak lagi memiliki dasar hukum pemrosesan wajib dimusnahkan.

---

## 2. Hak Subjek Data & Implementasi Fungsional

UU PDP Bab IV menjamin hak-hak mendasar subjek data pribadi:

| Hak Subjek Data | Pasal UU PDP | Implementasi Teknis FinTrack v2 |
|---|---|---|
| **Hak Informasi** | Pasal 20 | Dokumen Kebijakan Privasi transparan saat registrasi; log audit aktivitas dapat dilihat di dashboard. |
| **Hak Akses & Salinan** | Pasal 21 | Endpoint `GET /api/v1/user/export-data` mengunduh seluruh data profil, transaksi, dan histori AI dalam format JSON/CSV terstruktur (*Data Portability*). |
| **Hak Pembaruan / Ralat** | Pasal 22 | Endpoint `PUT /api/v1/user/profile` untuk ralat nama, email, nomor kontak langsung. |
| **Hak Penghentian / Hapus** | Pasal 23 & 43 | Endpoint `POST /api/v1/user/delete-account` yang mengeksekusi alur pemusnahan data terverifikasi. |

---

## 3. Resolusi Ketegangan Hukum: Hak Hapus (UU PDP) vs Retensi Pajak 10 Tahun (UU KUP)

### 3.1 Konflik Regulasi
- **UU PDP Pasal 43 & Pasal 44**: Subjek data berhak mengakhiri pemrosesan dan menghapus data pribadinya (*Right to be Forgotten / Erasure*). Pelanggaran terhadap hak ini dapat dikenakan sanksi administratif dan denda hingga 2% dari pendapatan tahunan (Pasal 57).
- **UU No. 28 Tahun 2007 tentang KUP (Ketentuan Umum dan Tata Cara Perpajakan) Pasal 28**: Wajib Pajak diwajibkan menyimpan buku, catatan, dan dokumen yang menjadi dasar pembukuan atau pencatatan, termasuk faktur pajak dan bukti penerimaan pembayaran, selama **10 (sepuluh) tahun** di Indonesia.
- **UU No. 8 Tahun 1997 tentang Dokumen Perusahaan**: Dokumen keuangan wajib disimpan sekurang-kurangnya 10 tahun.

Jika FinTrack v2 menghapus baris tabel `invoices` dan `payments` saat pengguna meminta penghapusan akun, perusahaan melanggar UU Perpajakan dan gagal audit fiskal DJP. Namun, jika FinTrack v2 menolak menghapus data identitas pengguna, perusahaan melanggar UU PDP.

### 3.2 Solusi Arsitektur: Pseudonimisasi & Anonimisasi Permanen (Cryptographic Shredding)
FinTrack v2 menyelesaikan ketegangan ini melalui pendekatan kepatuhan ganda yang diakui secara internasional dan selaras dengan penjelasan Pasal 43 UU PDP:
**Data PII pengguna dihapus dan ditimpa secara permanen, namun record transaksi keuangan dianonimkan untuk mempertahankan integritas neraca pembukuan.**

```
                             [ Permintaan Hapus Akun ]
                                         │
                                         ▼
            ┌────────────────────────────────────────────────────────┐
            │       Alur Eksekusi Anonimisasi & Pemusnahan           │
            └────────────────────────────────────────────────────────┘
                       │                                    │
                       ▼                                    ▼
       [ DATA NON-FINANSIAL & PII ]             [ DATA KEUANGAN / PAJAK ]
       - Chat History AI -> HARD DELETE        - Tabel 'invoices' -> ANONYMIZE
       - pgvector Embedding -> HARD DELETE     - Tabel 'payments' -> ANONYMIZE
       - Password hash, IP -> HARD DELETE      - Nomor Faktur -> TETAP UTUH
       - Kolom Nama/Email -> REPLACED DGN:     - Rincian PPN -> TETAP UTUH
         "deleted-user-<UUID>@anonymized"      - Nilai Rupiah -> TETAP UTUH
       - Master Crypto Key -> SHREDDED         - Retensi Wajib: 10 TAHUN
```

---

## 4. Spesifikasi Teknis Pemusnahan Data (`internal/user/anonymize.go`)

Ketika pengguna mengonfirmasi penghapusan akun:

### 4.1 Tahap 1: Pembersihan Data Non-Finansial (Hard Delete)
```sql
-- 1. Hapus riwayat chat sesi AI dan memory context
DELETE FROM chat_messages WHERE user_id = $1;
DELETE FROM chat_sessions WHERE user_id = $1;

-- 2. Hapus memori vector embedding RAG (pgvector 768d)
DELETE FROM transaction_embeddings WHERE user_id = $1;

-- 3. Hapus sesi aktif, cache token, dan refresh token
DELETE FROM user_sessions WHERE user_id = $1;
```

### 4.2 Tahap 2: Pseudonimisasi Tabel Profil Pengguna (`users`)
```sql
UPDATE users 
SET 
    email = 'anonymized-' || id || '@deleted.fintrack.internal',
    full_name = 'Eks Pengguna (Teranonimkan)',
    phone_number = NULL,
    password_hash = 'DELETED_CRYPTOGRAPHIC_SHREDDING',
    avatar_url = NULL,
    is_active = FALSE,
    deleted_at = NOW()
WHERE id = $1;
```

### 4.3 Tahap 3: Anonimisasi Dokumen Faktur Pajak & Pembayaran
```sql
-- Update invoice penagihan untuk audit perpajakan (UU KUP)
UPDATE invoices
SET 
    customer_name = 'Subjek Pajak Teranonimkan (UU PDP)',
    customer_email = 'anonymized@deleted.fintrack.internal',
    customer_address = 'Telah Dimusnahkan Berdasarkan Permintaan Subjek Data',
    is_anonymized = TRUE
WHERE user_id = $1;

-- Update status payment record
UPDATE payments
SET 
    payer_details_masked = TRUE
WHERE user_id = $1;
```

---

## 5. Audit Trail & Log Kepatuhan Hukum

Setiap proses penghapusan dan anonimisasi data dicatat ke dalam tabel `audit_logs` dengan format kepatuhan:
- `action`: `USER_DATA_SUBJECT_ERASURE_COMPLETED`
- `entity`: `user`
- `entity_id`: UUID subjek data
- `actor`: `system_pdp_compliance_worker`
- `metadata`: Memuat rincian volume baris data yang dimusnahkan (misal: "Hapus 142 chat_messages, 45 embeddings, anonimisasi 3 invoices").
- Log audit ini disimpan selama 2 tahun dan dikecualikan dari penghapusan subjek data sebagai bukti pemenuhan kewajiban kepatuhan hukum (*legal defense exemption* Pasal 43 ayat 3 UU PDP).
