# Review Guideline & Definition of Done — FinTrack v2

## 1. Kriteria Review (Checklist)

### Keamanan
- [ ] Backend otoritatif; tidak ada keputusan hak akses di frontend.
- [ ] Tidak ada secret di kode/log.
- [ ] Endpoint sensitif melewati guard yang sesuai.
- [ ] Mutasi uang/token idempoten.

### Data
- [ ] Uang = integer; tidak ada float.
- [ ] Ledger append-only (tidak ada UPDATE/DELETE).
- [ ] Migrasi aditif & replayable.
- [ ] Tidak ada data user yang hilang.

### Kualitas
- [ ] Modul kecil ber-interface; tanggung jawab jelas.
- [ ] Tidak ada `: any` (TS) / `context.Background()` (Go).
- [ ] Ada test untuk klaim kritis.
- [ ] Tidak ada duplikasi logika bisnis.

### Operasional
- [ ] Fitur baru di balik feature flag.
- [ ] Observability: metrik relevan ditambahkan.
- [ ] Audit log untuk aksi sensitif.
- [ ] Rollback jelas.

## 2. Definition of Done (per task)

- [ ] Kode & test ditulis; `go vet/test/build` hijau.
- [ ] `pnpm build/lint/test` hijau.
- [ ] Migrasi (bila ada) replayable; dump di-regenerate.
- [ ] Feature flag (bila fitur baru).
- [ ] Audit log (bila mutasi sensitif).
- [ ] `tracking/tracking.md` diperbarui.
- [ ] `log_aktivitas.md` dicatat.

## 3. Definition of Done (per fase)

- [ ] Semua task fase selesai & terverifikasi.
- [ ] Kriteria penerimaan fase terpenuhi (lihat README fase).
- [ ] Tidak ada regresi pada fase sebelumnya.
- [ ] Zero downtime terverifikasi.
- [ ] Dokumentasi fase diperbarui.

## 4. Alur Review

1. Penulis mengajukan perubahan + deskripsi dampak.
2. Reviewer mengecek checklist di atas.
3. Jika ada temuan → perbaiki → review ulang.
4. Setelah lolos → merge; catat di log aktivitas.

## 5. Anti-Pattern yang Dilarang

- Menaruh logika bisnis di frontend.
- Menghapus data user karena alasan billing.
- Mengubah entitlement via JWT (harus cache berversi).
- Cache billing/entitlement dengan TTL panjang tanpa versi.
- Migrasi destruktif dalam satu langkah rilis.
- Mengandalkan status pembayaran dari klien.

## Referensi

- `STANDAR_KODE.md`, `TESTING_STRATEGY.md`
- `../../CLAUDE.md`
