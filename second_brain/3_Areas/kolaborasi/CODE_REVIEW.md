# Code Review — FinTrack v2

## 1. Tujuan

Memastikan perubahan **benar, aman, dapat dipelihara**, dan sesuai keputusan yang sudah dikunci —
khususnya untuk area sensitif: **billing, token, pembayaran, keamanan, migrasi**.

## 2. Kapan Review Wajib

| Perubahan | Review |
|---|---|
| Logika uang/token | **Wajib** |
| Webhook/verifikasi pembayaran | **Wajib** |
| Migrasi DB | **Wajib** |
| Perubahan otorisasi/entitlement | **Wajib** |
| UI biasa | Dianjurkan |

## 3. Tanggung Jawab

| Peran | Tanggung jawab |
|---|---|
| Penulis | Menyertakan deskripsi dampak + bukti test |
| Reviewer | Mengecek checklist `../kualitas-kode/REVIEW_GUIDELINE.md` |
| Integrator | Menjaga konsistensi antar-modul |

## 4. Checklist Cepat

- [ ] Backend otoritatif (bukan frontend).
- [ ] Uang integer; ledger append-only; idempoten.
- [ ] Tidak ada secret di kode/log.
- [ ] Migrasi aditif & replayable.
- [ ] Fitur di balik feature flag.
- [ ] Ada test untuk klaim kritis.
- [ ] Audit log untuk aksi sensitif.
- [ ] Tidak ada data user yang hilang.

## 5. Alur

```
Penulis → ajukan → Reviewer cek checklist → temuan? → perbaiki → review ulang → merge
```

## 6. Anti-Pattern (langsung tolak)

- Logika bisnis di frontend.
- Menghapus data user karena billing.
- Entitlement di JWT.
- Cache billing/entitlement tanpa versi.
- Migrasi destruktif satu langkah.
- Percaya status pembayaran dari klien.

## Referensi

- `../kualitas-kode/REVIEW_GUIDELINE.md`, `../kualitas-kode/TESTING_STRATEGY.md`
