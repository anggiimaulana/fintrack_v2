# Temuan Regulasi: Kesenjangan Hak Subjek Data UU PDP No. 27/2022 & Retensi Pajak

**Kode Dokumen**: AUDIT-REG-01  
**Klasifikasi**: Kepatuhan Regulasi & Perlindungan Data Pribadi (Legal Compliance & Data Privacy)  

---

## 1. Lokasi & Dasar Kesenjangan (Regulatory Gap)
- **Dasar Hukum**:
  - UU No. 27 Tahun 2022 tentang Perlindungan Data Pribadi (UU PDP):
    - **Pasal 21**: Hak Mengakses dan Memperoleh Salinan Data Pribadi (*Right to Access / Data Portability*).
    - **Pasal 43 & 44**: Kewajiban Pengendali Data Menghapus Data Pribadi (*Right to Erasure / Right to be Forgotten*).
    - **Pasal 57**: Sanksi administratif denda maksimal 2% dari total pendapatan tahunan.
  - UU No. 28 Tahun 2007 (UU KUP) Pasal 28: Kewajiban menyimpan pembukuan & bukti transaksi selama 10 tahun.
- **Dokumen Referensi Proyek**: `second_brain/2_Projects/P08_Hardening-GoLive/teknis/pdp-compliance.md` §2 & §4.
- **Lokasi Kode Terdampak**: `backend/internal/delivery/http/router.go`, `backend/internal/user/anonymize.go`, `frontend/app/(dashboard)/user/profile/page.tsx`.

---

## 2. Analisis Kesenjangan Teknis (Technical Discrepancy)

### 2.1 Kondisi Saat Ini (As-Is)
Pada Fase P08, tim telah berhasil mengimplementasikan dan memverifikasi unit `AnonymizeService` di `backend/internal/user/anonymize.go` yang menyelesaikan konflik antara penghapusan PII dan retensi pajak 10 tahun.

**Namun, terjadi kesenjangan integrasi pada lapisan HTTP dan Antarmuka Pengguna**:
1. **Tidak Ada Endpoint Penghapusan Akun Mandiri**:
   - `router.go` sama sekali tidak mengekspos endpoint `POST /api/v1/user/delete-account` atau `DELETE /api/v1/me`.
   - Modul `AnonymizeUserData` hanya dapat dijalankan secara internal melalui kode Go / database langsung.
2. **Tidak Ada Endpoint Ekspor Data Pribadi (*Data Portability*)**:
   - Dokumen `pdp-compliance.md` menyebutkan endpoint `GET /api/v1/user/export-data` untuk mengunduh arsip JSON/CSV profil, transaksi, dan histori percakapan AI pengguna.
   - Endpoint ini **belum dibuat** pada `router.go` maupun handler HTTP backend.
3. **Ketiadaan Menu di Antarmuka Pengguna (Frontend UI)**:
   - Pada halaman profil pengguna (`frontend/app/(dashboard)/user/profile/page.tsx`), tidak tersedia tombol "Unduh Salinan Data Saya" (Pasal 21) maupun opsi zona bahaya "Hapus Akun Saya Secara Permanen" (Pasal 43).

### 2.2 Risiko Hukum & Kepatuhan
Jika pengguna FinTrack v2 mengajukan permohonan penghapusan data atau salinan data pribadi melalui email layanan pelanggan dan sistem tidak menyediakan sarana swalayan yang terotomasi dalam batas waktu 3x24 jam (Pasal 43 ayat 2 UU PDP), perusahaan menghadapi risiko sanksi peringatan tertulis dan denda administratif dari Komisi Pelindungan Data Pribadi.

---

## 3. Solusi & Spesifikasi Implementasi

### 3.1 Endpoint 1: `DELETE /api/v1/user/account` (Right to Erasure)
- **Metode**: `DELETE` (atau `POST /api/v1/user/delete-account`)
- **Autentikasi**: Wajib login JWT + verifikasi konfirmasi password terkini (`confirm_password`) untuk mencegah aksi tidak sah dari sesi yang tertinggal.
- **Alur Kerja**:
  1. Validasi password pengguna via `auth.CheckPassword`.
  2. Eksekusi `anonymizeSvc.AnonymizeUserData(ctx, userID, &userID, req.Reason)`.
  3. Bersihkan sesi cookie (`ft_at`, `ft_rt`).
  4. Hapus token sesi dari Redis.
  5. Kembalikan respons sukses HTTP 200 dan arahkan pengguna ke halaman utama.

```go
type DeleteAccountRequest struct {
    Password string `json:"password" binding:"required"`
    Reason   string `json:"reason"`
}

func (h *UserHandler) DeleteAccount(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    var req DeleteAccountRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "Password konfirmasi wajib diisi")
        return
    }

    // 1. Verifikasi password terkini
    _, hash, err := h.userRepo.GetByIDWithPassword(c.Request.Context(), userID)
    if err != nil || !auth.CheckPassword(hash, req.Password) {
        response.Unauthorized(c, "Password konfirmasi salah")
        return
    }

    // 2. Jalankan anonimisasi atomik & shredding
    res, err := h.anonymizeSvc.AnonymizeUserData(c.Request.Context(), userID, &userID, req.Reason)
    if err != nil {
        response.InternalError(c, "Gagal memproses penghapusan akun")
        return
    }

    // 3. Cabut cookie sesi
    h.authHandler.clearCookies(c)

    response.OK(c, "Akun dan data pribadi berhasil dihapus sesuai UU PDP No. 27/2022", res)
}
```

### 3.2 Endpoint 2: `GET /api/v1/user/export-data` (Data Portability)
- **Metode**: `GET`
- **Output**: File zip atau arsip payload JSON terkompresi yang memuat:
  - `profile.json`: Nama, email, tanggal bergabung, histori role.
  - `transactions.json`: Seluruh daftar transaksi finansial non-soft-deleted.
  - `ai_conversations.json`: Seluruh riwayat pesan chat AI pengguna.
  - `billing_summary.json`: Ringkasan pembayaran dan faktur langganan.
- **Proteksi Rate Limit**: Maksimal 3 kali per hari per pengguna untuk mencegah ekstrasi data masif / saturasi database.

### 3.3 Penyesuaian Frontend UI
1. Tambahkan seksi **"Zona Privasi & Kepatuhan UU PDP"** di `frontend/app/(dashboard)/user/profile/page.tsx` atau `frontend/app/(dashboard)/user/settings/page.tsx`.
2. Sediakan tombol **"Ekspor Seluruh Data Pribadi (JSON)"** dengan unduhan langsung.
3. Sediakan tombol bahaya **"Hapus Akun & Data Pribadi"** dengan dialog modal konfirmasi password dua langkah dan peringatan edukatif tentang penghapusan permanen riwayat chat dan pseudonimisasi pembukuan.
