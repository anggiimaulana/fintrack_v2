# 00 — Ringkasan Eksekutif

[← Kembali ke index](./README.md)

---

## 1. Latar belakang

User melaporkan: *"chat sekali di akun Free langsung habis 5k dan tidak bisa lagi"*.
Audit (lihat `../2026-10-15_AI-Chat-Kuota-Render-Sesi/00-audit-ai-chat.md`) menemukan
dua sebab yang saling memperkuat:

| Sebab | Detail |
|---|---|
| **Estimasi reserve tidak proporsional** | Reserve flat 2.000 token; chat nyata 5.000–10.000 |
| **Commit meng-clamp senyap** | Tagihan dipotong ke sisa kuota → seluruh siklus habis dalam 1 chat |

Perbaikan 2026-10-15 sudah mengatasi *estimasi* (kini proporsional). Namun **masalah
struktural tetap ada**: kuota Free 5.000 token memang lebih kecil daripada biaya satu
percakapan nyata. Menaikkan angka kuota menyelesaikan gejala, tetapi:

- angka kuota jadi terlihat tidak realistis ("1.000.000 token"?), dan
- setiap kenaikan kuota = kenaikan eksposur biaya tanpa kendali.

## 2. Permintaan user

> *"jumlah kuota dan matrix fitur dari masing-masing plan itu dinamis dari sisi super
> admin agar tidak hardcode dan dapat disesuaikan seiring berjalannya waktu."*

> *"biaya atau penggunaan token itu saya ingin ada kalkulasinya... misalnya kita
> katakan 5000 kuota untuk free... di super admin ada perhitungannya, misalnya per 5k
> token itu menggunakan pemakaian dari sisi plan = 1k... biar ga terkesan boros."*

Dua permintaan:

1. **Dinamisasi penuh** kuota + matriks fitur dari Super Admin (tanpa hardcode).
2. **Rasio konversi** token → kuota, sehingga kuota plan tidak cepat habis dan
   terkesan boros.

## 3. Solusi

### 3.1 Dinamisasi (sebagian besar sudah ada, dilengkapi)

Sudah tersedia: tabel `plan_limits`, `plan_entitlements`, `pricing_rules`, katalog
entitlement, dan endpoint admin. **Yang ditambahkan:**

- Editor limit & fitur memakai **dropdown katalog** (anti-typo), bukan input teks bebas.
- **Kalkulator kuota** di halaman Super Admin: dari kuota + rasio + harga token,
  tampilkan estimasi "berapa chat", "berapa biaya", "berapa margin".
- Nilai rasio menjadi kolom `plans.token_quota_ratio` yang ikut editor plan.

### 3.2 Rasio token (inti)

Memisahkan **dua besaran** yang saat ini tercampur:

```
        ┌──────────────────────────────────────────────┐
        │  Provider melaporkan 9.945 token             │
        └──────────────────┬───────────────────────────┘
                           │
              ┌────────────┴────────────┐
              │                         │
   ┌──────────▼──────────┐   ┌──────────▼───────────┐
   │ raw_tokens = 9.945  │   │ kuota = ceil(9945/5) │
   │ → hitung BIAYA      │   │      = 1.989 unit    │
   │   (Rp, margin)      │   │ → kurangi KUOTA user │
   └─────────────────────┘   └──────────────────────┘
```

| Aspek | Sebelum | Sesudah |
|---|---|---|
| Kuota Free | 5.000 token | 5.000 **unit** |
| 1 chat (9.945 token) | menagih 9.945 → habis + clamp | menagih **1.989 unit** → sisa 3.011 |
| Chat per siklus Free | ~0,5 (langsung habis) | **~2–3** |
| Biaya Rupiah tercatat | dari token nyata | **tetap** dari token nyata ✅ |
| Angka kuota yang dilihat user | membengkak kalau dinaikkan | tetap kecil & ramah |

## 4. Dampak

### 4.1 Dampak ke user

| Sebelum | Sesudah |
|---|---|
| "Kuota 5.000 token", habis setelah 1 chat | "Kuota 5.000", habis setelah ~2–3 chat |
| Pesan error setelah kuota habis mendadak | Badge sisa kuota + peringatan ≤20% |
| Angka kuota membingungkan (satuan token) | Angka kuota netral ("unit kuota") |

> **Catatan transparansi:** karena K-3 (user melihat unit kuota saja), user tidak
> melihat token provider. Ini disengaja agar angka tidak menakutkan, tetapi
> **wajib** didokumentasikan di halaman bantuan/langganan bahwa "kuota diukur dalam
> unit, bukan token mentah" supaya tidak dianggap menyesatkan.

### 4.2 Dampak ke operasional

| Aspek | Perubahan |
|---|---|
| Ledger | +1 kolom `raw_tokens` (nullable, backfill dari `amount`) |
| Plan | +1 kolom `token_quota_ratio` (default 1 = perilaku lama) |
| Commit | Dibagi rasio sebelum `AdjustPeriod` |
| Refund/Release | Dikalikan rasio balik (invarian: total kembali persis) |
| Pricing | **Tidak berubah** — tetap dari `raw_tokens` |
| Backward compat | Rasio default `1` ⇒ perilaku identik dengan sekarang |

### 4.3 Dampak biaya (contoh, asumsi harga jual Rp 3.000/10k token)

| Plan | Kuota | Rasio | Chat efektif | Biaya jual/chat | Biaya provider/chat* | Margin |
|---|---|---|---|---|---|---|
| Free | 5.000 | 5 | ~2 | Rp 3.729 | ~Rp 1.491 | ~Rp 2.238 (60%) |
| Pro | 150.000 | 2 | ~30 | Rp 3.729 | ~Rp 1.491 | ~Rp 2.238 (60%) |
| Business | 1.000.000 | 1 | ~100 | Rp 3.729 | ~Rp 1.491 | ~Rp 2.238 (60%) |

\* asumsi provider Rp 150/1k token (`provider_cost_per_1k_idr` saat ini **0**, lihat [05](./05-model-biaya-dan-margin.md)).

**Poin kunci:** margin per chat **tidak berubah** oleh rasio — rasio hanya mengubah
berapa banyak chat yang "muat" dalam kuota. Ini yang membuat ide user elegan:
kita bisa bermurah hati soal kuota tanpa mengubah ekonomi per pemakaian.

## 5. Keputusan yang sudah dikunci

| # | Keputusan | Nilai | Konsekuensi |
|---|---|---|---|
| K-1 | Mekanisme | Rasio token; biaya dari token nyata | +kolom `raw_tokens`, +kolom rasio |
| K-2 | Cakupan | Per-plan, **dinamis** | `plans.token_quota_ratio` |
| K-3 | Tampilan user | Satuan **"kuota"** (dinamis via setting) | Token nyata hanya di admin |
| K-4 | Urutan | Dokumen desain dulu | Dokumen ini |
| **K-5** | **Prinsip utama** | **SEMUA dinamis dari Super Admin — tanpa hardcode** | Katalog setting/metrik/fitur jadi **data** (lihat [09](./09-lapisan-konfigurasi-dinamis.md)) |

## 6. Jawaban atas keputusan P-1 … P-11

| # | Pertanyaan | **Jawaban user** | Implementasi (semua dinamis) |
|---|---|---|---|
| **P-1** | Pembulatan kuota | **`ceil`** | Setting `quota_rounding_mode` (default `ceil`, bisa diubah) |
| **P-2** | Rasio awal | **Dinamis, jangan hardcode** | `plans.token_quota_ratio` per plan + default dari setting `quota_default_ratio` |
| **P-3** | Label satuan | **"kuota"** | Setting `quota_unit_label` (bisa diubah) |
| **P-4** | Plan unlimited | **Tetap dibagi** | Rasio tetap diterapkan pada plan `-1` |
| **P-5** | Migrasi plan lama | **Default 1** | `ADD COLUMN ... NOT NULL DEFAULT 1` |
| **P-6** | Backfill `raw_tokens` | **Pilihan B** (tanpa UPDATE) | Helper `RawTokensOrAmount()`, trigger append-only utuh |
| **P-7** | Constraint tabel besar | **`NOT VALID` bila > 10 juta** | Dokumentasi prosedur kondisional |
| **P-8** | Migrasi seed rasio | **Serahkan ke operator** | Tidak ada migrasi seed rasio; diisi via UI |
| **P-9** | Kuota Free | **Default 10k, dinamis, untuk semua plan** | Setting `quota_default_tokens_per_cycle` + editor plan |
| **P-10** | Harga Pro/Business | **Evaluasi setelah biaya nyata — dinamis** | Editor `pricing_rules` (sudah ada) + kalkulator |
| **P-11** | Biaya provider | **WAJIB, isi default** | Setting `pricing_default_provider_cost_per_1k_idr` (ada nilai default) |

### 6.1 Prinsip "tanpa hardcode"

> **"semua keputusan, logic, flow, dan kalkulasi itu dari super admin dan dinamis."**

Yang dipindahkan dari kode ke data (detail di [09](./09-lapisan-konfigurasi-dinamis.md)):

| Hardcode | Menjadi | Tabel/Setting |
|---|---|---|
| `writableSettings` (allowlist setting) | `setting_definitions` | tabel baru |
| `UsageFor()` switch | `metric_definitions.resolver_key` | tabel baru |
| `FeatureCatalog` / `LimitCatalog` | `feature_definitions` / `metric_definitions` | tabel baru |
| `ceilDiv` (mode tetap) | `quota_rounding_mode` | setting |
| `chatOverheadTokens`, `charsPerToken` | `ai_estimate_*` | setting |
| Sentinel unlimited `1e9` | `quota_unlimited_sentinel` | setting |
| Default kuota/rasio/label | `quota_default_*`, `quota_unit_label` | setting |
| Biaya provider default | `pricing_default_provider_cost_per_1k_idr` | setting |

**Seam yang diakui (tetap di kode):** cara menghitung usage (resolver),
struktur rute, dan config bootstrap (`DATABASE_URL`, `JWT_SECRET`, …).
Developer menyediakan **kemampuan**; operator memilih **kebijakan**.

### 6.2 Keputusan tambahan P-12 … P-16 (akibat prinsip "semua dinamis")

| # | Pertanyaan | **Keputusan** |
|---|---|---|
| P-12 | Konstanta lama | **Simpan sebagai fallback permanen** — tabel katalog kosong tidak mematikan sistem |
| P-13 | Kesegaran konfigurasi | **Invalidasi event-driven via SSE + Redis pub/sub** — bukan TTL. Perubahan berlaku segera, hemat resource, konsisten lintas-instance (TTL 5 menit hanya jaring terakhir) |
| P-14 | Resolver baru | Boleh ditambah via kode (seam diakui); memilih yang sudah ada tidak perlu deploy |
| P-15 | Lantai tagih minimum | **Ada** (`quota_min_charge_units`, default 1) — menutup celah spam mode `floor` |
| P-16 | Hapus definisi | **Soft delete** (`is_active = false`) — jangan putus data |

Detail: [09 §14](./09-lapisan-konfigurasi-dinamis.md).

## 7. Ruang lingkup

**Termasuk:**
- Kolom rasio per plan + kolom `raw_tokens` di ledger
- Konversi di `Commit`/`Refund`/`Reserve` beserta invarian
- API admin untuk rasio & kalkulator
- **Katalog konfigurasi dinamis** ([09](./09-lapisan-konfigurasi-dinamis.md)):
  `setting_definitions`, `metric_definitions`, `feature_definitions`
- **Pembulatan dinamis** (`quota_rounding_mode`), **estimasi dinamis**, **default
  dinamis**, **label satuan dinamis**, **sentinel unlimited dinamis**
- Editor plan berbasis katalog (dropdown), editor settings, editor metrik
- Tampilan kuota di halaman langganan & badge chat
- Migrasi + backfill + dump

**Tidak termasuk (usulan terpisah):**
- Perubahan harga plan (hanya *alat hitung*, bukan perubahan harga)
- Kuota per-modalitas (chat/vision/voice beda rasio) — lihat [08](./08-risiko-dan-rollback.md) sebagai opsi lanjutan
- Billing berbayar nyata (Midtrans) — tidak tersentuh
- Model ML untuk prediksi pemakaian
- **Resolver usage baru** (menambah *cara hitung* tetap butuh deploy — seam diakui)

## 8. Estimasi usaha

| Fase | Isi | Perkiraan |
|---|---|---|
| F1 | Migrasi DB + domain + ledger | 1–2 hari |
| F2 | Ledger (konversi + pembulatan dinamis) | 1–2 hari |
| F3 | Repository + wiring | 0,5–1 hari |
| **F3b** | **Lapisan konfigurasi dinamis** (3 tabel katalog + resolver registry) | **1–2 hari** |
| F4 | API admin + kalkulator + katalog | 1–2 hari |
| F5 | UI Super Admin (plan, settings, metrik) | 1–2 hari |
| F6 | UI user (label dinamis) | 0,5–1 hari |
| F7 | Testing menyeluruh | 1–2 hari |
| F8 | Docs + dump + rollout | 1 hari |
| **Total** | | **8–13 hari kerja** |

---

**Navigasi:** [Index](./README.md) · [Desain Teknis →](./01-desain-teknis.md)
