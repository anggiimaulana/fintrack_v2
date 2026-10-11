# Migrasi Database — Aturan & Pola Expand/Contract

Direktori ini memuat migrasi bernomor kanonik (`000N_*.sql`). Dokumen ini mengikat: aturan di
bawah **bukan saran**, melainkan syarat agar deploy tanpa downtime tetap mungkin.

> Konteks operasional (cara menjalankan, dump, backup/restore) ada di `../README.md`.
> Dokumen ini khusus **aturan menulis migrasi**.

---

## 1. Prinsip: Expand → Migrate → Contract

Aplikasi lama dan baru berjalan **bersamaan** selama rolling deploy. Karena itu setiap perubahan
skema harus dibagi menjadi fase yang aman untuk kedua versi:

| Fase | Isi | Aman untuk versi lama? |
|---|---|---|
| **Expand** | Tambah struktur baru (kolom nullable, tabel, indeks) — tidak menghapus/mengubah yang lama | ya |
| **Migrate** | Isi/backfill data; aplikasi mulai menulis ke struktur baru | ya (kedua versi berjalan) |
| **Contract** | Hapus struktur lama **setelah** tidak ada versi aplikasi yang memakainya | ya, jika fase sebelumnya sudah rilis |

Satu migrasi **tidak boleh** melakukan expand dan contract sekaligus.

---

## 2. Dilarang

| Larangan | Alasan | Gantinya |
|---|---|---|
| `DROP COLUMN` pada migrasi yang sama dengan perubahan pemakaiannya | Instance lama masih membaca kolom itu → error 500 saat rolling deploy | Berhenti memakai kolom di satu rilis, `DROP COLUMN` di rilis **berikutnya** |
| `ALTER TABLE ... ALTER COLUMN ... SET NOT NULL` tanpa default/backfill | Baris lama bernilai NULL → migrasi gagal, atau constraint meledak saat insert | `ADD COLUMN` nullable → backfill → `SET NOT NULL` di migrasi terpisah |
| `ALTER TABLE ... ADD COLUMN ... NOT NULL` tanpa `DEFAULT` | Mengunci tabel dan gagal bila ada baris lama | `ADD COLUMN` nullable, atau `NOT NULL DEFAULT <nilai>` |
| `CREATE INDEX` biasa pada tabel besar | Mengunci penulisan selama pembuatan | `CREATE INDEX CONCURRENTLY` |
| Mengedit migrasi yang **sudah di-commit** | Checksum/urutan berbeda antar lingkungan; DB dev dan prod menyimpang | Migrasi bernomor **baru** |
| `ALTER TABLE ... RENAME COLUMN` langsung | Instance lama memakai nama lama | Tambah kolom baru, backfill, ganti pemakaian, drop yang lama |
| DDL di luar transaksi yang mengasumsikan nol traffic | Beban aktif bisa memblokir | Pola non-blocking di bawah |

---

## 3. Wajib

1. **Aditif & idempoten.** Gunakan `CREATE TABLE IF NOT EXISTS`, `ADD COLUMN IF NOT EXISTS`,
   `CREATE OR REPLACE FUNCTION`, `DROP TRIGGER IF EXISTS ... ; CREATE TRIGGER ...`. Migrasi harus
   aman dijalankan ulang (pola ini sudah dipakai 0001–0010).
2. **Satu migrasi = satu perubahan bernama jelas.** Jangan gabungkan beberapa perubahan tak
   berkaitan.
3. **Bernomor berurutan**, tanpa celah yang membingungkan: `0011_...`, `0012_...`.
4. **Uji di `fintrack_test` dulu**, baru ke `fintrack`.
5. **Regenerate dump** setelah perubahan skema: `database/scripts/dump.sh`.
6. **Non-blocking untuk tabel besar:**
   ```sql
   -- Index: hindari lock penulisan
   CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_foo_bar ON foo (bar);

   -- Constraint: tambahkan sebagai NOT VALID, validasi terpisah
   ALTER TABLE foo ADD CONSTRAINT chk_foo CHECK (bar > 0) NOT VALID;
   ALTER TABLE foo VALIDATE CONSTRAINT chk_foo;
   ```
   Catatan: `CREATE INDEX CONCURRENTLY` **tidak boleh** berjalan di dalam transaksi. Migrasi yang
   memakainya harus dijalankan di luar blok transaksi eksplisit.

---

## 4. Pola yang direkomendasikan

### 4.1 Menambah kolom

```sql
-- 0011_add_foo_baz.sql  (EXPAND)
ALTER TABLE foo ADD COLUMN IF NOT EXISTS baz text;
```
Nullable dulu. Backfill menyusul (migrasi terpisah atau skrip), baru constraint bila perlu.

### 4.2 Mengganti nama kolom

Tiga rilis, jangan satu:

1. **Expand:** `ADD COLUMN new_name ...`
2. **Migrate:** backfill dari `old_name`; aplikasi menulis ke **keduanya**
3. **Contract:** berhenti menulis `old_name`, lalu `DROP COLUMN old_name`

### 4.3 Mengubah nilai/enum

Jangan ubah `CHECK` lama secara langsung bila instance lama masih menulis nilai lama. Tambah
constraint baru yang menerima **kedua** nilai, pindahkan data, lalu persempit di rilis berikutnya.

### 4.4 Tabel baru

Aman: tidak ada konsumen lama. Tetap idempoten (`IF NOT EXISTS`), tambahkan indeks yang
diperlukan sejak awal, dan pertimbangkan RLS/trigger yang konsisten dengan tabel lain.

---

## 5. Rollback

Migrasi **expand** dapat di-rollback dengan aman (kolom/tabel baru belum dipakai). Migrasi
**contract** tidak dapat di-rollback tanpa kehilangan data — karena itu contract selalu terpisah
dan dirilis belakangan, setelah yakin tidak ada yang kembali.

Sediakan pernyataan `down` di komentar migrasi bila operasinya reversible, mis.:

```sql
-- down: ALTER TABLE foo DROP COLUMN IF EXISTS baz;
```

---

## 6. Checklist sebelum commit migrasi

- [ ] Hanya berisi satu perubahan yang jelas namanya
- [ ] Idempoten (`IF NOT EXISTS` / `OR REPLACE` / `DROP ... IF EXISTS`)
- [ ] Tidak ada `DROP COLUMN`, `SET NOT NULL`, atau `RENAME` yang membahayakan instance lama
- [ ] Indeks pada tabel besar memakai `CONCURRENTLY`
- [ ] Sudah diuji di `fintrack_test` (maju; ulang untuk membuktikan idempoten)
- [ ] `database/dump/` sudah di-regenerate
- [ ] Tidak mengedit migrasi yang sudah di-commit
