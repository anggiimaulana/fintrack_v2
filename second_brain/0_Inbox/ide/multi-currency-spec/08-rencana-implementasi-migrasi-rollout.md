# 08 — Rencana Implementasi, Migrasi & Rollout

**Status:** DRAFT  
Strategi: **expand → migrate → contract**, deploy bertahap, feature flag, selalu dapat di-rollback tanpa kehilangan data.

---

## 1. Prinsip Rilis

1. **Skema lebih dulu, kode menyusul.** Semua perubahan DB bersifat additive dengan DEFAULT → backend lama tetap berjalan.
2. **Default tetap IDR.** Tanpa aktifkan apa pun, perilaku = sebelum perubahan (zero behaviour change).
3. **Feature flag** menggerbang permukaan pengguna (`multi_currency`), memakai `feature_flags` yang sudah ada (migrasi `0006`).
4. **Billing/AI-quota adalah zona beku.** Ada uji regresi khusus.
5. **Setiap fase punya rollback** yang tidak menyentuh data pengguna.

---

## 2. Fase

### Fase 0 — Keputusan
- Setujui D1–D12 (README + dokumen 09).
- Freeze ruang lingkup rilis.

### Fase 1 — Fondasi DB (tanpa UI)
Migrasi `0039`–`0042` (dokumen 03).
- Katalog + seed IDR/JPY (JPY nonaktif).
- Kolom `users.display_currency`, kolom transaksi.
- Backfill `amount_base = amount`.
- Constraint `NOT VALID` → `VALIDATE`.
- Flag `multi_currency` **off**.

Verifikasi:
```sql
SELECT COUNT(*) FROM transactions WHERE is_deleted=false AND amount_base <> amount;  -- 0
SELECT code, is_active, is_base FROM currencies ORDER BY sort_order;                 -- IDR aktif, JPY nonaktif
```

Rollback: drop kolom/tabel (belum dipakai kode baru).

### Fase 2 — Backend (kode, di belakang flag)
- Paket `internal/currency`.
- Repository + usecase + endpoint katalog/preferensi.
- Agregat `amount` → `amount_base`.
- AI konteks, ekspor, budget.
- Flag off → endpoint katalog tetap ada (aman), tetapi UI belum menampilkan picker.

Verifikasi: `go test ./...`, uji regresi billing.

Rollback: redeploy image sebelumnya; kolom baru diabaikan. **Tidak ada migrasi balik** (data tetap valid).

### Fase 3 — Frontend (di belakang flag)
- `CurrencyProvider` + `lib/money.ts` (default IDR = output identik).
- Migrasi permukaan bertahap: dashboard → transactions → reports → budgets → charts → AI.
- Halaman Super Admin `currencies`.
- Flag `multi_currency` tetap off sampai verifikasi.

Verifikasi: `pnpm lint && pnpm test && pnpm build`; snapshot IDR tidak berubah.

### Fase 4 — Aktifkan (canary)
- Aktifkan `multi_currency` untuk Super Admin dulu → uji manual end-to-end.
- Tambah kurs JPY nyata (Super Admin) → aktifkan JPY.
- Aktifkan untuk sebagian user (bila flag mendukung persentase/allowlist) → pantau.

### Fase 5 — Rilis penuh
- Aktifkan untuk semua.
- Umumkan perubahan.

### Fase 6 — Contract (opsional, rilis berikutnya)
- Setelah tidak ada trafik versi lama: perketat constraint (`VALIDATE` selesai), hapus kode fallback, bereskan formatter IDR sisa.

---

## 3. Matriks Deploy

| Urutan | Artefak | Aman tanpa pasangan? |
|---|---|---|
| 1 | Migrasi 0039–0042 | Ya (additive) |
| 2 | Backend baru | Ya (kolom baru diabaikan = perilaku lama) |
| 3 | Frontend baru (flag off) | Ya |
| 4 | Flag on + kurs JPY | Ya, dapat dimatikan kembali |

Tidak ada fase yang memerlukan downtime.

---

## 4. Feature Flag

| Flag | Default | Efek |
|---|---|---|
| `multi_currency` | off | Menampilkan picker & permukaan multi-mata-uang |

- Mengikuti pola `feature_flags` yang ada (`0006_create_feature_flags.sql`).
- Dimatikan → picker disembunyikan; `display_currency` diabaikan di UI; backend tetap default IDR.
- **Jangan** mematikan flag sebagai satu-satunya rollback setelah transaksi JPY tersimpan: data JPY tetap valid, hanya tampilannya kembali IDR (nilai dasar tetap benar). Ini aman karena model dual-value.

---

## 5. Backfill & Performa

| Aspek | Rencana |
|---|---|
| Ukuran tabel | Bila besar, batch `UPDATE ... WHERE amount_base = 0` per 5.000 baris |
| Lock | `NOT VALID` constraint → tidak memblokir baca/tulis |
| Index | `CREATE INDEX CONCURRENTLY` bila memungkinkan (atau di jendela sepi) |
| Rollback data | Tidak perlu; backfill idempoten & benar secara semantik |

---

## 6. Observability

| Metrik/log | Tujuan |
|---|---|
| Counter konversi gagal (`currency_no_rate`) | Deteksi miskonfigurasi kurs |
| Counter transaksi per mata uang | Adopsi JPY |
| Latency query rate-as-of | Jaga index tetap efektif |
| Log audit rate ditambah | Jejak perubahan kurs |
| Log panggilan konversi dari billing | Harus **nol** (guardrail) |

---

## 7. Checklist Pra-Deploy

- [ ] D1–D12 disetujui.
- [ ] `go test ./...` hijau (termasuk regresi billing).
- [ ] `pnpm lint`, `pnpm test`, `pnpm build` hijau.
- [ ] Snapshot tampilan IDR identik.
- [ ] Backfill diverifikasi (0 selisih).
- [ ] Flag `multi_currency` off pada deploy pertama.
- [ ] Kurs JPY diisi Super Admin sebelum aktivasi.
- [ ] Runbook rollback tertulis.

---

## 8. Rollback Playbook

| Gejala | Aksi |
|---|---|
| Tampilan aneh / agregat salah | Matikan `multi_currency` (kembali IDR) |
| Backend error pada katalog | Redeploy image sebelumnya (kolom diabaikan) |
| Backfill belum tuntas | Hentikan job; kolom default tetap IDR; lanjutkan nanti |
| Kurs salah input | Tambah baris koreksi (append-only); jangan hapus |
| Masalah billing | Investigasi terpisah; billing tidak menyentuh `currency.Service` |

Tidak ada langkah rollback yang menghapus data transaksi.

---

## 9. Perkiraan Ukuran Pekerjaan (kasar)

| Area | Berat |
|---|---|
| Migrasi + backfill | S |
| Paket `internal/currency` + test | M |
| Repository + usecase + endpoint | M |
| AI/ekspor/budget integrasi | M |
| Provider FE + migrasi permukaan | L |
| Halaman Super Admin | M |
| Uji & QA | M |

**Total: besar (L).** Disarankan dipecah per fase, bukan satu PR raksasa.

---

## 10. Urutan PR yang Disarankan

1. PR-1: migrasi 0039–0041 + backfill + index.
2. PR-2: `internal/currency` + repository + test.
3. PR-3: transaksi (create/update/list/agregat) + preferensi user.
4. PR-4: AI + ekspor + budget.
5. PR-5: FE provider + `money.ts` + permukaan inti (default IDR).
6. PR-6: FE sisa permukaan + `CurrencyInput`.
7. PR-7: halaman Super Admin + audit + realtime.
8. PR-8: constraint 0042 + uji regresi + dokumentasi.

Tiap PR hijau mandiri dan tidak mengaktifkan flag.

---

## 11. Ringkasan Keputusan Rollout

| # | Keputusan |
|---|---|
| R-1 | Expand → migrate → contract; skema dulu |
| R-2 | Default IDR, zero behaviour change saat flag off |
| R-3 | Flag `multi_currency` menggerbang UI |
| R-4 | JPY di-seed nonaktif; aktivasi manual + kurs |
| R-5 | Rollback utama = matikan flag |
| R-6 | Billing/AI-quota tidak pernah masuk jalur konversi |
| R-7 | 8 PR bertahap, tiap PR hijau mandiri |
