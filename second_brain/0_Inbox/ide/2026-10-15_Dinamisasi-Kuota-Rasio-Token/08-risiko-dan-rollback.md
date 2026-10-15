# 08 — Risiko, Rollback, & Mitigasi

[← Index](./README.md) · [← Rencana Testing](./07-rencana-testing.md)

---

## 1. Daftar risiko

| ID | Risiko | Probabilitas | Dampak | Skor | Mitigasi utama |
|---|---|---|---|---|---|
| **R-1** | Salah satuan (RAW vs unit) tercampur | Sedang | **Kritis** | 🔴 | Konversi **hanya** di ledger; test I-4/I-6 |
| **R-2** | Kuota user berubah mendadak mid-siklus | Rendah | Tinggi | 🟠 | Snapshot rasio di `token_periods` |
| **R-3** | Margin salah → bisnis rugi tanpa sadar | Sedang | Tinggi | 🟠 | Kalkulator + isi `provider_cost_per_1k_idr` |
| **R-4** | Rasio terlalu tinggi → user tidak sadar biaya | Sedang | Sedang | 🟡 | Peringatan di kalkulator; batas 1000 |
| **R-5** | Migrasi mengunci tabel besar | Rendah | Sedang | 🟡 | `NOT VALID` + `VALIDATE` |
| **R-6** | Test lama rusak karena signature berubah | Tinggi | Rendah | 🟡 | Fixture default ratio=1; ubah satu commit |
| **R-7** | User bingung satuan "kuota" | Sedang | Sedang | 🟡 | Catatan di UI + tooltip |
| **R-8** | Operator salah set rasio → user komplain | Sedang | Sedang | 🟡 | Simulasi sebelum simpan; audit log |
| **R-9** | Backfill `raw_tokens` melanggar append-only | Rendah | Sedang | 🟡 | Pilihan B (tanpa UPDATE) |
| **R-10** | Perubahan tidak konsisten antar-tab (cache) | Sedang | Rendah | 🟢 | Invalidasi cache setelah simpan |
| **R-11** | Operator salah konfigurasi → sistem rusak | Sedang | Tinggi | 🟠 | Validasi dari katalog + batas min/max + audit |
| **R-12** | Tabel katalog kosong/terhapus → sistem mati | Rendah | **Kritis** | 🔴 | **Fallback konstanta permanen** (P-12) |
| **R-13** | Metrik disalahkonfigurasi (resolver salah) | Sedang | Sedang | 🟡 | Validasi silang + dropdown resolver |
| **R-14** | `floor` membuka celah spam kuota | Sedang | Sedang | 🟡 | `quota_min_charge_units` (dinamis) |
| **R-15** | Terlalu dinamis → sulit diaudit/di-debug | Sedang | Sedang | 🟡 | Audit log setiap perubahan definisi |
| **R-16** | Query katalog tiap request → beban DB | Sedang | Sedang | 🟡 | Cache ber-versi + invalidasi event-driven (P-13) |
| **R-17** | Event SSE/Redis hilang → klien basi | Sedang | Sedang | 🟡 | Jaring: `BumpVersion` + TTL 5 menit + resync saat reconnect |

---

## 2. Analisis risiko kritis

### 2.1 R-1 — Salah satuan (risiko tertinggi)

**Skenario gagal:**

```go
// SALAH: handler mengonversi, lalu ledger mengonversi LAGI
handler:  estUnits := RoundUnits(raw, ratio, mode)   // 762
ledger:   units := RoundUnits(762, ratio, mode)      // 152  ← DOUBLE CONVERSION
// Kuota bocor: user dapat 5× lebih banyak chat dari seharusnya
```

**Dampak:** kuota bocor → biaya provider meledak, margin negatif tanpa terdeteksi.

**Mitigasi berlapis:**

| Lapis | Tindakan |
|---|---|
| 1 | **Keputusan tegas:** konversi **hanya** di `LedgerService`. Handler mengirim RAW |
| 2 | Nama parameter eksplisit: `estRawTokens`, `rawTokens` — bukan `tokens`/`amount` |
| 3 | Komentar di setiap fungsi: `// rawTokens = token provider, BUKAN unit kuota` |
| 4 | Test I-6: ubah rasio, biaya **harus** tetap (menangkap double conversion) |
| 5 | Test §3.1: verifikasi angka persis (762 unit, bukan 152) |

**Deteksi dini:** tambahkan assertion di test bahwa `reserve.Amount == RoundUnits(raw, ratio, mode)`
**sekali** — bukan dua kali.

### 2.2 R-2 — Kuota berubah mendadak

**Skenario gagal:**

```
Hari 1: user punya 5.000 kuota, sudah pakai 3.000 (rasio 1)
Hari 2: operator ubah rasio → 5
        → sisa 2.000 kuota kini "bernilai" 10.000 token
        → user merasa kuotanya bertambah tiba-tiba (atau sebaliknya)
```

**Mitigasi:** `token_periods.quota_ratio` di-snapshot saat siklus dibuat. Perubahan
rasio **hanya** berlaku pada siklus berikutnya. Siklus berjalan tetap memakai rasio
lama → tidak ada perubahan mendadak.

**Konsekuensi yang harus dikomunikasikan:** operator **tidak bisa** memperbaiki rasio
yang salah di tengah siklus untuk user aktif. Jika ini diperlukan, sediakan
**"grant"** manual (sudah ada `EntryGrant` di domain) untuk menambah kuota user
terdampak.

### 2.3 R-3 — Margin salah

**Kondisi saat ini:** `provider_cost_per_1k_idr = 0` di rule `default` → kalkulator
menampilkan margin **100%** (menyesatkan).

**Analisis dari [05](./05-model-biaya-dan-margin.md):** dengan asumsi biaya provider
Rp 1.491/chat, plan Pro dan Business **berpotensi rugi** bila user memakai seluruh
kuota.

**Mitigasi:**

| # | Tindakan |
|---|---|
| 1 | **Wajib** isi `provider_cost_per_1k_idr` sebelum produksi (P-11) |
| 2 | Kalkulator memperingatkan bila `provider_cost = 0` → "margin tidak dapat dihitung" |
| 3 | Tampilkan margin per plan di `/super-admin/quota-monitor` |
| 4 | Evaluasi ulang harga plan (P-10) setelah data nyata masuk |

### 2.4 R-12 — Tabel katalog kosong (risiko kritis baru)

**Skenario gagal:**

```
1. Migrasi 0029 dijalankan, tetapi seed GAGAL (mis. constraint)
2. setting_definitions KOSONG
3. Operator membuka /super-admin/settings → tidak ada yang bisa diubah
4. Lebih buruk: bila kode TIDAK punya fallback, pembacaan setting gagal
   → kuota tidak dapat dihitung → semua request AI error 500
```

**Mitigasi (WAJIB):**

| Lapis | Tindakan |
|---|---|
| 1 | **Fallback konstanta permanen** di kode (P-12). Kode TIDAK PERNAH bergantung penuh pada tabel |
| 2 | Seed idempoten + diverifikasi di CI (hitung baris setelah migrasi) |
| 3 | Health check: `/admin/config-status` melaporkan bila katalog kosong |
| 4 | Alert bila `count(setting_definitions) = 0` |

```go
// Pola fallback yang WAJIB dipakai
func (c *SettingsCatalog) Int(ctx context.Context, key string, fallback int64) int64 {
    if def, ok := c.defs[key]; ok {
        if v, ok := c.values[key]; ok { return v }
        if v, ok := def.DefaultValue.(int64); ok { return v }
    }
    return fallback   // ← konstanta kode sebagai jaring terakhir
}
```

> **Prinsip:** "dinamis" berarti **kebijakan** ada di DB, tetapi kode tetap menyimpan
> **default aman**. Ini yang membedakan sistem yang dinamis dengan sistem yang rapuh.

### 2.5 R-17 — Event SSE/Redis hilang (klien basi)

**Skenario:** Redis restart, jaringan putus, atau klien offline saat operator mengubah
konfigurasi. Klien tidak menerima `config_changed` → menampilkan nilai lama.

**Mengapa ini BUKAN masalah kritis:** SSE adalah **akselerator**, bukan sumber
kebenaran. Tiga jaring pengaman memastikan sistem tetap benar:

| Jaring | Cara kerja |
|---|---|
| **Versi cache** | `BumpVersion` terbaca saat request berikutnya → server memuat katalog baru, bahkan tanpa event |
| **Buffer replay** | Event yang terlewat disimpan; klien menerimanya saat resume (`Last-Event-ID`) |
| **TTL 5 menit** | Jaring terakhir: klien memuat ulang paling lambat 5 menit |

**Mitigasi tambahan:**

| Tindakan | Efek |
|---|---|
| `stream_reconnected` → resync | Klien memuat ulang setelah koneksi pulih |
| `POST /admin/config/reload` | Operator dapat memicu invalidasi manual |
| Health check Redis | Deteksi dini bila pub/sub mati |

> **Prinsip:** *event-driven untuk kesegaran, versi untuk konsistensi, TTL panjang
> untuk jaring terakhir.* Tidak ada satu titik kegagalan.

### 2.6 R-14 — Celah spam pada mode `floor`

Dengan `quota_rounding_mode = floor`, request kecil dapat menagih **0 unit**:

```
RoundUnits(4, 5, floor) = 0     → request 4 token tidak menagih apa pun
```

**Mitigasi:** setting `quota_min_charge_units` (default `1`) sebagai lantai:

```go
units := RoundUnits(rawTokens, ratio, mode)
if units < minCharge && rawTokens > 0 {
    units = minCharge     // minimal 1 unit per request
}
```

Karena `quota_min_charge_units` **juga dinamis**, operator dapat menaikkannya (mis. 5)
bila menemukan penyalahgunaan.

---

## 3. Prosedur rollback

### 3.1 Prinsip

| Lapisan | Rollback |
|---|---|
| **Skema (expand)** | Aman di-rollback (kolom baru belum dipakai penuh) |
| **Kode** | Deploy versi sebelumnya; rasio default 1 ⇒ perilaku lama |
| **Data** | **TIDAK** di-rollback (ledger append-only; audit harus utuh) |

### 3.2 Rollback kode (paling mungkin dipakai)

```bash
# 1. Deploy ulang versi sebelumnya
git revert <commit-rasio>   # atau checkout tag sebelumnya
# 2. Kode lama mengabaikan kolom baru (SELECT eksplisit tanpa kolom baru)
#    → sistem kembali ke perilaku rasio=1
```

**Syarat agar rollback kode aman:** kode lama **tidak** melakukan `SELECT *` pada
`token_periods`/`token_ledger`/`plans`. Berdasarkan audit, repository memakai SELECT
**eksplisit** per kolom → **aman**. ✅

> **Verifikasi wajib sebelum deploy:** pastikan semua query memakai daftar kolom
> eksplisit, bukan `SELECT *`. Bila ada yang `SELECT *` + `Scan` posisional, rollback
> bisa gagal.

### 3.3 Rollback skema

```sql
-- Jalankan blok `down` dari migrasi 0028 (lihat 02 §2)
ALTER TABLE token_ledger DROP CONSTRAINT IF EXISTS token_ledger_raw_tokens_check;
ALTER TABLE token_ledger DROP COLUMN IF EXISTS raw_tokens;
ALTER TABLE token_periods DROP CONSTRAINT IF EXISTS token_periods_quota_ratio_check;
ALTER TABLE token_periods DROP COLUMN IF EXISTS quota_ratio;
ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_token_quota_ratio_check;
ALTER TABLE plans DROP COLUMN IF EXISTS token_quota_ratio;

-- Migrasi 0029 (katalog dinamis). JANGAN di-drop bila kode masih membacanya.
-- Bila di-drop, kode WAJIB punya fallback konstanta (P-12) agar tetap jalan.
-- DROP TABLE IF EXISTS metric_definitions;
-- DROP TABLE IF EXISTS feature_definitions;
-- DROP TABLE IF EXISTS setting_definitions;
```

> ⚠️ **Jangan** jalankan rollback skema saat kode baru masih berjalan — kode baru
> membaca kolom yang sudah dihapus → error 500.

**Urutan rollback yang benar:**
```
1. Rollback KODE (deploy versi lama)     ← kode lama tidak pakai kolom baru
2. Tunggu konfirmasi stabil
3. (Opsional) Rollback SKEMA             ← hanya bila perlu
```

### 3.4 Rollback data (tidak disarankan)

`token_ledger` bersifat **append-only** dan menjadi bukti audit keuangan. **Tidak
boleh** di-rollback. Bila ada kesalahan akuntansi:

| Situasi | Tindakan |
|---|---|
| Kuota user salah tagih | Terbitkan `EntryAdjust` atau `EntryGrant` (bukan hapus) |
| Kuota user terlalu sedikit | `EntryGrant` dengan alasan |
| Kuota user terlalu banyak | **Biarkan** (jangan hukum user karena bug internal) |

---

## 4. Rencana mitigasi per fase

| Fase | Risiko khusus | Mitigasi |
|---|---|---|
| F1 Migrasi | Locking tabel besar | `NOT VALID`; jalankan di jam sepi; uji di `fintrack_test` |
| F2 Ledger | Salah satuan | Test I-1…I-8; review kode oleh 2 orang |
| F3 Wiring | Pemanggil terlewat | `go build ./...` menangkap semua |
| F4 API | Kalkulator salah | Test rumus vs `PricingEngine` (harus sama) |
| F5 UI | Operator salah set | Simulasi wajib sebelum simpan; konfirmasi |
| F6 UI User | User bingung | Catatan + tooltip + FAQ |
| F7 Rollout | Dump tertinggal | Regenerate + verifikasi kolom |

---

## 5. Fitur pengaman (safety nets)

| # | Pengaman | Tujuan |
|---|---|---|
| S-1 | Rasio default `1` | Deploy tanpa mengubah perilaku |
| S-2 | Snapshot rasio di period | Tidak ada perubahan mid-siklus |
| S-3 | `RoundUnits` + clamp | Tidak menagih kurang; kuota tidak negatif |
| S-4 | `CHECK (ratio >= 1)` di DB | Data tidak valid ditolak di lapisan DB |
| S-5 | Validasi 1–1000 di API | Rasio ekstrem ditolak |
| S-6 | Peringatan kalkulator | Operator diberi tahu sebelum menyimpan |
| S-7 | Audit log pada perubahan plan | Jejak siapa mengubah apa |
| S-8 | `provider_cost = 0` → peringatan | Mencegah margin menyesatkan |
| S-9 | Ledger append-only | Audit keuangan tidak bisa dimanipulasi |
| S-10 | Idempotency key | Tidak ada double-charge |
| S-11 | **Fallback konstanta permanen** | Tabel katalog kosong tidak mematikan sistem |
| S-12 | Validasi dari katalog DB | Operator tidak bisa menyimpan nilai tidak valid |
| S-13 | Cache ber-versi + invalidasi SSE/Redis | Perubahan segera, hemat resource, konsisten lintas-instance |
| S-14 | `quota_min_charge_units` | Menutup celah spam pada mode `floor` |
| S-15 | Audit log perubahan definisi | Jejak siapa mengubah kebijakan apa |
| S-16 | Validasi silang metrik↔resolver | Mencegah metrik tanpa cara hitung |
| S-17 | **Invalidasi event-driven (SSE + Redis)** | Perubahan berlaku segera, hemat resource (P-13) |
| S-18 | **Jaring berlapis** (versi + buffer + TTL 5m) | Event hilang tidak membuat klien basi permanen |

---

## 6. Monitoring pasca-rilis

| Metrik | Ambang | Tindakan |
|---|---|---|
| Rasio `used/quota` rata-rata Free | > 90% | Pertimbangkan naikkan kuota |
| Biaya provider/user/bulan | > Rp 10.000 (Free) | Evaluasi rasio/kuota |
| Margin per plan | < 20% | Naikkan markup atau harga |
| Error `insufficient_quota` | Naik tajam | Cek rasio & kuota |
| `raw_tokens` = 0 pada commit baru | > 0 | Bug konversi — investigasi |
| Selisih `sum(commit) - sum(refund)` vs `used` | ≠ 0 | Bug ledger — **kritis** |

### 6.1 Query monitoring

```sql
-- Rasio pemakaian Free (rata-rata)
SELECT AVG(used_tokens::float / NULLIF(quota_tokens,0)) AS avg_utilization
  FROM token_periods p
  JOIN subscriptions s ON s.user_id = p.user_id
  JOIN plans pl ON pl.id = s.plan_id
 WHERE pl.code = 'free' AND p.period_end > now();

-- Biaya provider per user per siklus
SELECT p.user_id,
       COALESCE(SUM(GREATEST(l.raw_tokens, ABS(l.amount))), 0) AS raw_tokens,
       COALESCE(SUM(l.cost_idr), 0) AS revenue_idr
  FROM token_periods p
  JOIN token_ledger l ON l.period_id = p.id AND l.entry_type = 'commit'
 WHERE p.period_end > now()
 GROUP BY p.user_id
 ORDER BY raw_tokens DESC;

-- Deteksi bug konversi (commit baru tanpa raw_tokens)
SELECT count(*) FROM token_ledger
 WHERE entry_type = 'commit' AND raw_tokens = 0
   AND created_at > now() - interval '1 day';
-- Harus 0 (kecuali baris lama sebelum migrasi)
```

---

## 7. Komunikasi ke user

### 7.1 Yang perlu diberitahukan

| Pesan | Di mana |
|---|---|
| "Kuota diukur dalam unit, bukan token mentah" | Catatan di `QuotaUsageCard` |
| "Kuota AI Anda: 3.011 tersisa" | Badge chat + halaman langganan |
| "Kuota habis — tingkatkan paket" | Banner + toast |
| FAQ: "Kenapa kuota tidak sama dengan token?" | Halaman bantuan |

### 7.2 Yang TIDAK perlu diberitahukan

- Detail rasio per plan (internal)
- Biaya provider & margin (rahasia bisnis)
- Rumus perhitungan

> **Prinsip transparansi:** user berhak tahu **apa yang mereka dapat** (kuota, berapa
> chat), bukan **bagaimana kita menghitungnya** secara internal. Namun satuan harus
> jelas dan tidak menyesatkan — karena itu "kuota" dipilih, bukan mengklaim "token".

---

## 8. Keputusan yang menunggu

| # | Pertanyaan | Dampak bila salah | Rekomendasi |
|---|---|---|---|
| **P-1** | Pembulatan kuota | ceil → bisnis aman; floor → bisnis rugi | **ceil** |
| **P-2** | Rasio awal | Terlalu tinggi → subsidi besar | Free 5 / Pro 2 / Business 1 |
| **P-3** | Label satuan | Menyesatkan user | **"kuota"** |
| **P-4** | Rasio plan unlimited | Biaya tak terukur | tetap dibagi |
| **P-5** | Migrasi plan lama | Perubahan perilaku tak terduga | default 1 |
| **P-6** | Backfill `raw_tokens` | Melanggar append-only | **Pilihan B** (tanpa UPDATE) |
| **P-7** | Constraint pada tabel besar | Locking | `NOT VALID` bila > 10 juta baris |
| **P-8** | Migrasi seed rasio | Menimpa keputusan operator | serahkan ke operator |
| **P-9** | Kuota Free | Terlalu kecil → user frustrasi | **10.000** |
| **P-10** | Harga Pro/Business | Berpotensi rugi | evaluasi setelah biaya nyata |
| **P-11** | Isi `provider_cost_per_1k_idr` | Margin tak terukur | **WAJIB** — diisi **default dinamis** |
| **P-12** | Konstanta lama | Sistem mati bila katalog kosong | **Simpan sebagai fallback permanen** |
| **P-13** | Kesegaran konfigurasi | Klien basi / beban DB | **Invalidasi event-driven (SSE + Redis)**; TTL 5 menit jaring terakhir |
| **P-14** | Resolver baru | Butuh deploy (seam) | Boleh; dokumentasikan |
| **P-15** | `quota_min_charge_units` | Celah spam pada `floor` | **Ada** (default 1, dinamis) |
| **P-16** | Hapus definisi | Memutus data | **Soft delete** (`is_active = false`) |

---

## 9. Ringkasan keputusan arsitektur (untuk ADR)

| Keputusan | Alasan | Alternatif ditolak |
|---|---|---|
| Konversi di ledger, bukan handler | Satu sumber kebenaran; hindari double-conversion | Konversi di handler (rawan) |
| Snapshot rasio di `token_periods` | Siklus beku; tidak ada perubahan mendadak | Baca rasio plan live (retroaktif) |
| `RoundUnits` (mode dinamis) | Jangan menagih kurang dari biaya nyata; operator dapat menyesuaikan | Mode tetap di kode |
| Biaya dari `raw_tokens` | Margin akurat & tidak bisa "disembunyikan" rasio | Biaya dari unit |
| Rasio default 1 | Deploy aman tanpa mengubah perilaku | Default 5 (berisiko) |
| Kolom di `plans` (per-plan) | Fleksibel Free vs Business | Global |
| **Katalog jadi data** (setting/metrik/fitur) | Operator mengubah kebijakan tanpa deploy | Konstanta kode (butuh deploy) |
| **Fallback konstanta permanen** | Tabel kosong tidak mematikan sistem | Tanpa fallback (rapuh) |
| **Seam diakui** (resolver, rute, bootstrap) | Kemampuan tetap di kode; kebijakan di DB | "Semua" di DB (tidak mungkin) |

---

## 10. Kesimpulan

Fitur ini **layak dikerjakan** karena:

| Alasan | Bukti |
|---|---|
| Menyelesaikan masalah nyata | User Free tidak bisa chat sama sekali (rasio 1) |
| Risiko terkendali | Default rasio 1; expand-only; snapshot |
| Ekonomi tidak berubah | Margin per chat tetap; hanya kuota yang berubah |
| Sudah ada fondasinya | Ledger, pricing engine, katalog, editor plan |
| Reversible | Rollback kode aman (SELECT eksplisit) |
| **Sepenuhnya dinamis** | Kebijakan (setting/metrik/fitur/kuota/rasio/harga) di DB |

**Prasyarat sebelum produksi:**

1. ✅ Semua keputusan P-1…P-16 dikonfirmasi
2. ✅ Test invarian (I-1…I-8) hijau
3. ✅ `provider_cost_per_1k_idr` diisi (default dinamis tersedia)
4. ✅ Dump di-regenerate
5. ✅ ADR ditulis
6. ✅ **Fallback konstanta terbukti bekerja** (test tabel kosong)

---

**Navigasi:** [← Rencana Testing](./07-rencana-testing.md) · [Index](./README.md)
