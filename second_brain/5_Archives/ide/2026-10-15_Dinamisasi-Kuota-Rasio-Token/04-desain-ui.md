# 04 — Desain UI

[← Index](./README.md) · [← Spesifikasi API](./03-spesifikasi-api.md)

---

## 1. Prinsip UI

| # | Prinsip | Alasan |
|---|---|---|
| U-1 | **Operator tidak boleh mengetik key** | Typo `ai_chatt` mematikan fitur diam-diam |
| U-2 | **Selalu tampilkan dampak, bukan hanya nilai** | "Kuota 5.000" tidak bermakna tanpa "= ~2 chat" |
| U-3 | **Angka kuota netral bagi user** | Satuan "kuota", bukan "token" (K-3) |
| U-4 | **Konsisten dengan design system** | Neo-brutalist + radius (ADR-011) |
| U-5 | **Wajib aksesibel** | Label eksplisit, `aria-*`, kontras AA |
| U-6 | **UI dibangun dari katalog** | Setting/metrik/fitur dirender dari DB, bukan daftar hardcoded di komponen |
| U-7 | **Editor untuk semua kebijakan** | Operator dapat mengubah default, pembulatan, label, dan metrik tanpa deploy |
| U-8 | **Perubahan berlaku segera** | Simpan → event SSE `config_changed` → UI memuat ulang tanpa refresh (P-13) |

---

## 2. Perubahan UI Super Admin

### 2.1 `PlanFormDialog` — editor plan

**Sebelum:** input teks bebas untuk key limit & fitur.

```
┌─────────────────────────────────────────────┐
│ Batas Penggunaan (Limits)      [+ Tambah]   │
│ ┌───────────────────────────┬─────────────┐ │
│ │ ai_chat_monthly  ← TEKS   │  5000       │ │  ← rawan typo
│ └───────────────────────────┴─────────────┘ │
└─────────────────────────────────────────────┘
```

**Sesudah:** dropdown dari katalog + satuan + peringatan + rasio.

```
┌──────────────────────────────────────────────────────────────┐
│ Rasio Kuota Token                                    [ ? ]   │
│ Token provider dibagi rasio ini sebelum ditagih ke kuota.    │
│  ┌──────────┐   Rasio 5 → 9.945 token ≈ 1.989 kuota          │
│  │    5     │                                                │
│  └──────────┘                                                │
├──────────────────────────────────────────────────────────────┤
│ Batas Penggunaan (Kuota)                       [+ Tambah]    │
│ ┌──────────────────────────┬──────────┬────────┬──────────┐  │
│ │ Kuota AI / siklus     ▾  │  5000    │ kuota  │ [🗑]     │  │
│ │ (dropdown dari katalog)  │          │        │          │  │
│ ├──────────────────────────┼──────────┼────────┼──────────┤  │
│ │ Permintaan AI / siklus ▾ │  50      │permint.│ [🗑]     │  │
│ └──────────────────────────┴──────────┴────────┴──────────┘  │
│  ⓘ Limit "Kuota AI" nilainya dipengaruhi rasio di atas.      │
├──────────────────────────────────────────────────────────────┤
│ Hak Akses Fitur (Entitlements)                 [+ Tambah]    │
│ ┌──────────────────────────┬───────┬───────────┐             │
│ │ Asisten AI (chat)     ▾  │  ●ON  │ [🗑]      │             │
│ │ Pindai struk (OCR)    ▾  │  ○OFF │ [🗑]      │             │
│ └──────────────────────────┴───────┴───────────┘             │
├──────────────────────────────────────────────────────────────┤
│ ⚠️ Limit "max_categories" belum ditegakkan backend.          │
│    Tidak akan terlihat oleh user.                            │
├──────────────────────────────────────────────────────────────┤
│ 📊 Simulasi: dengan kuota 5.000 & rasio 5 → ~2 chat/siklus   │
│    Biaya jual Rp 3.729/chat · margin Rp 2.238/chat (60%)     │
│                                              [ Lihat detail ]│
└──────────────────────────────────────────────────────────────┘
```

#### Komponen

| Elemen | Implementasi |
|---|---|
| Dropdown key | `useEntitlementCatalog()` → `catalog.limits` / `catalog.features` |
| Satuan | Dari `catalog.limits[].unit` (read-only, tampil sebagai teks) |
| Peringatan "belum ditegakkan" | `enforcedByKey[key] === false` (sudah ada pola) |
| Input rasio | `Input type="number" min={1} max={1000}` + helper text |
| Simulasi inline | Panggil `POST /admin/plans/estimate` (debounce 500 ms) |

#### Perilaku dropdown key

```tsx
// Konseptual — PlanFormDialog
<Select value={item.key} onValueChange={(v) => handleUpdateLimitKey(item.id, v)}>
  <SelectTrigger><SelectValue placeholder="Pilih limit…" /></SelectTrigger>
  <SelectContent>
    {catalog.limits.map((l) => (
      <SelectItem key={l.key} value={l.key}>
        <span className="flex items-center gap-2">
          {l.label}
          {!l.enforced && <Badge variant="outline">belum aktif</Badge>}
          {l.ratioAdjusted && <Badge variant="secondary">rasio</Badge>}
        </span>
      </SelectItem>
    ))}
  </SelectContent>
</Select>
```

**Penanganan key lama yang tidak ada di katalog:** tetap ditampilkan (agar data lama
tidak hilang) tetapi ditandai ⚠️ "tidak dikenal".

---

### 2.2 Halaman baru: `/super-admin/kuota` — Kalkulator & Monitor

Halaman khusus yang menjawab *"berapa chat & berapa biaya"*.

```
┌─────────────────────────────────────────────────────────────────────┐
│  KALKULATOR KUOTA & BIAYA                                           │
│  Simulasikan dampak kuota + rasio sebelum menyimpan perubahan.      │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─ Input ──────────────────┐  ┌─ Hasil ────────────────────────┐  │
│  │ Plan        [Free    ▾]  │  │ Kuota per chat     1.989 kuota │  │
│  │ Kuota       [ 5000    ]  │  │ Chat per siklus    ~2,5 chat   │  │
│  │ Rasio       [ 5       ]  │  │ Sisa akhir siklus  1.022 kuota │  │
│  │ Token/chat  [ 9945    ]  │  ├────────────────────────────────┤  │
│  │                          │  │ Biaya jual/chat    Rp 3.729    │  │
│  │ [ Aturan harga: Default▾]│  │ Biaya provider     Rp 1.491    │  │
│  └──────────────────────────┘  │ Margin/chat        Rp 2.238 60%│  │
│                                ├────────────────────────────────┤  │
│                                │ Biaya 1 siklus penuh Rp 2.982  │  │
│                                │ Pendapatan siklus    Rp 7.458  │  │
│                                │ Margin siklus        Rp 4.476  │  │
│                                └────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│  ⚠️ Rasio 5 memberi ~2,5 chat/siklus. Pertimbangkan kuota ≥ 10.000  │
│     bila ingin ≥ 5 chat.                                            │
├─────────────────────────────────────────────────────────────────────┤
│  Perbandingan:  tanpa rasio 0,50 chat  →  dengan rasio 2,51 chat ✅ │
└─────────────────────────────────────────────────────────────────────┘
```

#### Elemen interaktif

| Elemen | Perilaku |
|---|---|
| Pilih Plan | Mengisi kuota + rasio dari plan terpilih |
| Kuota / Rasio / Token per chat | Input angka; memicu hitung ulang (debounce 300 ms) |
| Aturan harga | Dropdown dari `GET /admin/pricing-rules` |
| Panel hasil | Diperbarui dari `POST /admin/plans/estimate` |
| Peringatan | Dari `warnings[]` |
| Perbandingan | Menonjolkan manfaat rasio (jawaban langsung ke kekhawatiran user) |

#### Aksesibilitas

- Setiap input punya `<Label htmlFor>`.
- Panel hasil memakai `aria-live="polite"` agar pembaca layar mengumumkan perubahan.
- Peringatan memakai `role="status"`.
- Tidak ada informasi yang hanya disampaikan lewat warna (selalu ada teks/ikon).

---

### 2.3 Halaman `/super-admin/tokens` — ledger

Tambahkan kolom **Token Nyata** dan **Kuota** agar operator bisa audit keduanya:

| Waktu | Tipe | Kuota | Token Nyata | Modalitas | Biaya |
|---|---|---|---|---|---|
| 10:20 | commit | 1.989 | 9.945 | chat | Rp 3.729 |
| 10:19 | reserve | 762 | — | chat | — |

> Agregat token nyata memakai `GREATEST(raw_tokens, ABS(amount))` (lihat [02 §6.5](./02-skema-database-dan-migrasi.md)).

---

### 2.4 Halaman `/super-admin/quota-monitor`

Tambahkan kolom **Token Nyata** di samping kuota, dan **biaya**:

| User | Plan | Kuota terpakai | Token nyata | Biaya | Sisa |
|---|---|---|---|---|---|
| Siti | Free | 1.989 / 5.000 | 9.945 | Rp 3.729 | 3.011 |

---

### 2.5 Halaman baru: `/super-admin/settings` — editor konfigurasi dinamis

Seluruh setting (termasuk default kuota, pembulatan, label) dirender **dari katalog**
`setting_definitions`. Menambah setting = tambah baris di katalog, **tanpa deploy**.

```
┌──────────────────────────────────────────────────────────────────────┐
│  PENGATURAN DINAMIS                                                  │
│  Semua nilai di bawah ini disimpan di database & dapat diubah        │
│  tanpa deploy.                                                       │
├──────────────────────────────────────────────────────────────────────┤
│  ▸ Kuota                                                             │
│    Mode pembulatan kuota      [ ceil          ▾]  ← enum dari katalog│
│      ⓘ ceil = jangan menagih kurang dari biaya nyata                 │
│    Rasio kuota default        [ 1             ]                      │
│    Kuota default / siklus     [ 10000         ]  ← default plan baru │
│    Label satuan kuota         [ kuota         ]                      │
│    Nilai pengganti unlimited  [ 1000000000    ]                      │
│    Lantai tagih minimum       [ 1             ]                      │
│                                                                      │
│  ▸ Estimasi AI                                                       │
│    Overhead estimasi (token)  [ 3000          ]                      │
│    Headroom keluaran          [ 800           ]                      │
│    Karakter per token         [ 4             ]                      │
│                                                                      │
│  ▸ Pajak / Mail / Storage / Gateway / …  (dari katalog, per group)   │
│                                                                      │
│                                        [ Simpan ]                    │
└──────────────────────────────────────────────────────────────────────┘
```

**Perilaku:**

| Elemen | Sumber |
|---|---|
| Grup (`▸ Kuota`) | `setting_definitions.group_key` |
| Label & deskripsi | `label`, `description` |
| Jenis input | `kind` (`bool`→Switch, `enum`→Select, `int`→number, `string`→text, `secret`→SecretField) |
| Batas validasi | `min_value`/`max_value`/`max_len` (client + server) |
| Urutan | `sort_order` |

> **Konsekuensi:** tidak ada lagi daftar setting di kode frontend. Komponen ini generik.

> **Kesegaran (P-13):** setelah Simpan, server mem-publish `config_changed` via SSE.
> Halaman ini (dan editor plan/metrik) memuat ulang katalog **segera** — tanpa
> refresh manual, tanpa menunggu TTL. Lihat
> [09 §12](./09-lapisan-konfigurasi-dinamis.md).

### 2.6 Halaman baru: `/super-admin/metrics` — editor metrik dinamis

"Metrix juga harus dinamis" → operator dapat menambah/mengubah metrik (limit).

```
┌──────────────────────────────────────────────────────────────────────┐
│  KATALOG METRIK                                    [+ Metrik Baru]   │
├──────────────────────────────────────────────────────────────────────┤
│  Key                    Label                 Unit    Enforced Resolver│
│  ai_tokens_per_cycle    Kuota AI / siklus     kuota   ● ya     token_…│
│  ai_requests_per_cycle  Permintaan AI / siklus permint. ● ya    ai_req…│
│  max_transactions       Transaksi             transaksi ● ya    active…│
│  ocr_scans_per_cycle    Pindai struk / siklus scan     ● ya     vision…│
│  max_categories         Kategori              kategori ○ tidak  —      │
│  max_wallets            Dompet                dompet   ○ tidak  —      │
├──────────────────────────────────────────────────────────────────────┤
│  [+ Metrik Baru]                                                     │
│   Key          [ budgets_created          ]                          │
│   Label        [ Anggaran / siklus        ]                          │
│   Unit         [ anggaran                 ]                          │
│   Deskripsi    [ Jumlah anggaran yang...  ]                          │
│   Enforced     [ ● ]  ← bila aktif, resolver wajib diisi             │
│   Resolver     [ budgets_created        ▾]  ← dari GET /admin/resolvers│
│   Ratio-aware  [ ○ ]  ← tandai bila dipengaruhi rasio token          │
│                                 [ Batal ]  [ Simpan ]                │
└──────────────────────────────────────────────────────────────────────┘
```

**Validasi di UI:** `enforced = true` tanpa `resolver` → tombol Simpan disabled +
peringatan. Resolver dipilih dari dropdown yang berasal dari kode (kemampuan).

> **Setelah metrik dibuat**, ia otomatis muncul di dropdown limit pada editor plan dan
> (bila ada usage) di halaman langganan user — **tanpa deploy**.

### 2.7 Halaman baru: `/super-admin/features` — editor fitur dinamis

Sama polanya: `feature_definitions` mengisi dropdown fitur di editor plan.

---

## 3. Perubahan UI user

### 3.1 Halaman langganan (`/user/settings/subscription`)

`QuotaUsageCard` kini menampilkan satuan dari katalog:

```
┌──────────────────────────────────────────────┐
│ 📊 Pemakaian Kuota                           │
│ Sisa kuota Anda pada siklus ini.             │
│ Periode berakhir 14 Nov 2026.                │
├──────────────────────────────────────────────┤
│ Kuota AI                 1.989 / 5.000       │
│ ████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░         │
│ Sisa 3.011 (60%)                             │
│                                              │
│ Permintaan AI            1 / 50              │
│ █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░         │
│ Sisa 49 (98%)                                │
└──────────────────────────────────────────────┘
```

**Perubahan kunci:**

| Sebelum | Sesudah |
|---|---|
| "Token AI" | **"Kuota AI"** |
| Angka = token mentah | Angka = unit kuota |
| Tidak ada penjelasan | Tooltip: "Kuota diukur dalam unit; token provider dibagi rasio plan." |

> **Wajib:** tambahkan catatan kecil di kartu agar tidak dianggap menyesatkan:
> *"Kuota diukur dalam unit, bukan token mentah."*

### 3.2 Badge kuota di header chat

Sudah ditambahkan pada perbaikan 2026-10-15 (`ChatWindow`). Tinggal:

| Sebelum | Sesudah |
|---|---|
| `Sisa token 3.011` | `Sisa kuota 3.011` |
| `Token habis` | `Kuota habis` |

Dan `LIMIT_LABELS` di `QuotaUsageCard`:

```ts
const LIMIT_LABELS: Record<string, string> = {
  ai_tokens_per_cycle: 'Kuota AI',        // ← dari 'Token AI'
  ai_requests_per_cycle: 'Permintaan AI',
  max_transactions: 'Transaksi',
  ocr_scans_per_cycle: 'Pindai struk',
}
```

### 3.3 Pesan error

| Sebelum | Sesudah |
|---|---|
| "Kuota token AI Anda untuk siklus ini sudah habis." | "Kuota AI Anda untuk siklus ini sudah habis." |
| "Terpakai 5.000 dari 5.000." | "Terpakai 5.000 dari 5.000 kuota." |

File: `frontend/lib/entitlement-errors.ts` → `LIMIT_MESSAGES`.

---

## 4. Komponen baru yang dibutuhkan

| Komponen | Lokasi | Fungsi |
|---|---|---|
| `LimitKeySelect` | `components/super-admin/plans/` | Dropdown limit dari katalog |
| `FeatureKeySelect` | `components/super-admin/plans/` | Dropdown fitur dari katalog |
| `QuotaRatioInput` | `components/super-admin/plans/` | Input rasio + helper text |
| `QuotaSimulationPanel` | `components/super-admin/plans/` | Panel simulasi inline |
| `QuotaCalculator` | `components/super-admin/quota/` | Kalkulator penuh (halaman) |
| `DynamicSettingsEditor` | `components/super-admin/settings/` | Render setting dari katalog |
| `MetricCatalogEditor` | `components/super-admin/metrics/` | CRUD metrik dari DB |
| `FeatureCatalogEditor` | `components/super-admin/features/` | CRUD fitur dari DB |
| `ResolverSelect` | `components/super-admin/metrics/` | Dropdown `resolver_key` dari API |

Semua memakai komponen UI yang ada (`Select`, `Input`, `Label`, `Badge`, `Card`) dan
mengikuti ADR-011 (radius aktif).

---

## 5. Hook baru / berubah

| Hook | Perubahan |
|---|---|
| `useSuperAdminPlans` | +`token_quota_ratio` di `PlanDTO` & `PlanPayload` |
| `useEntitlementCatalog` | +`description`, `ratioAdjusted` |
| `useQuotaEstimate` | **BARU** — panggil `POST /admin/plans/estimate` |
| `useEntitlements` | +`unit` di `UsageEntry` |
| `useSettingDefinitions` | **BARU** — katalog setting dari DB |
| `useMetricDefinitions` | **BARU** — katalog metrik + CRUD |
| `useFeatureDefinitions` | **BARU** — katalog fitur + CRUD |
| `useResolvers` | **BARU** — daftar `resolver_key` dari kode |

---

## 6. Alur kerja operator (end-to-end)

**A. Menyetel kebijakan (tanpa deploy)**

```
1. Operator buka /super-admin/settings → grup "Kuota"
2. Set: mode pembulatan = ceil, kuota default = 10.000, label = "kuota"
3. Simpan → tersimpan di tabel `settings` (bukan kode)
4. Operator buka /super-admin/metrics → pastikan metrik yang dibutuhkan ada
   (bila perlu tambah: key + resolver dari dropdown)
5. Operator buka /super-admin/pricing → isi biaya provider (default sudah ada)

**B. Menyetel plan**

6. Operator buka /super-admin/plans → klik "Edit" pada plan Free
7. Dialog terbuka: rasio saat ini 1, kuota 5.000
8. Operator ubah rasio → 5
9. Panel simulasi otomatis tampil: "~2,5 chat/siklus · margin Rp 2.238/chat"
10. Operator buka /super-admin/kuota untuk simulasi lebih detail
11. Puas → Simpan
12. Perubahan tersimpan di plans.token_quota_ratio
13. Berlaku pada SIKLUS BERIKUTNYA (siklus berjalan tetap memakai snapshot)
14. User melihat kuota yang sama (5.000) tetapi kini bertahan ~2–3 chat

> **Tidak ada langkah yang memerlukan deploy** kecuali menambah *cara hitung*
> (resolver) yang benar-benar baru. Seluruh kebijakan — default, pembulatan, label,
> metrik, fitur, kuota, rasio, harga — diubah dari UI.
```

---

## 7. Mockup teks lengkap (editor plan)

```
┌─ Edit Paket: Free ──────────────────────────────────── [✕] ─┐
│                                                             │
│ Kode        ┌──────────────┐   Nama  ┌──────────────────┐   │
│             │ free         │         │ Free             │   │
│             └──────────────┘         └──────────────────┘   │
│                                                             │
│ Deskripsi   ┌─────────────────────────────────────────────┐ │
│             │ Paket gratis untuk mencoba                  │ │
│             └─────────────────────────────────────────────┘ │
│                                                             │
│ Harga/bln   ┌──────────────┐   Harga/thn ┌──────────────┐    │
│             │ Rp 0         │             │ Rp 0         │    │
│             └──────────────┘             └──────────────┘    │
│                                                             │
│ ── Rasio Kuota Token ─────────────────────────────────────  │
│ Token provider dibagi rasio ini sebelum ditagih ke kuota.   │
│ Rasio 1 = 1 token : 1 kuota.                                │
│             ┌──────────────┐                                │
│             │ 5            │   → 9.945 token ≈ 1.989 kuota  │
│             └──────────────┘                                │
│                                                             │
│ ── Batas Penggunaan (Kuota) ──────────────── [+ Tambah] ──  │
│ ┌────────────────────────────┬─────────┬────────┬───────┐  │
│ │ Kuota AI / siklus       ▾  │ 5000    │ kuota  │  🗑   │  │
│ ├────────────────────────────┼─────────┼────────┼───────┤  │
│ │ Permintaan AI / siklus  ▾  │ 50      │ permnt.│  🗑   │  │
│ └────────────────────────────┴─────────┴────────┴───────┘  │
│  ⓘ "Kuota AI" dipengaruhi rasio di atas.                    │
│                                                             │
│ ── Hak Akses Fitur ───────────────────────── [+ Tambah] ──  │
│ ┌────────────────────────────┬────────┬───────┐            │
│ │ Asisten AI (chat)       ▾  │  ● ON  │  🗑   │            │
│ ├────────────────────────────┼────────┼───────┤            │
│ │ Pindai struk (OCR)      ▾  │  ○ OFF │  🗑   │            │
│ └────────────────────────────┴────────┴───────┘            │
│                                                             │
│ ── Simulasi (otomatis) ───────────────────────────────────  │
│ 📊 Kuota 5.000 · rasio 5 → ~2,5 chat/siklus                 │
│    Jual Rp 3.729/chat · margin Rp 2.238 (60%)               │
│                                        [ Simulasi detail ]  │
│                                                             │
│                                  [ Batal ]  [ Simpan ]      │
└─────────────────────────────────────────────────────────────┘
```

---

**Navigasi:** [← Spesifikasi API](./03-spesifikasi-api.md) · [Model Biaya →](./05-model-biaya-dan-margin.md)
