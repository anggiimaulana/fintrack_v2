# 05 — Model Biaya & Margin

[← Index](./README.md) · [← Desain UI](./04-desain-ui.md)

---

## 1. Prinsip ekonomi

| # | Prinsip |
|---|---|
| E-1 | **Rasio hanya mengubah KUOTA, tidak mengubah BIAYA.** Margin per chat tetap. |
| E-2 | Biaya dihitung dari **token nyata** (`raw_tokens`), bukan unit kuota |
| E-3 | Uang selalu **integer Rupiah** (ADR-003); pembulatan ke bawah (floor) untuk biaya |
| E-4 | Pembulatan kuota ke **atas** (ceil) — jangan menagih kurang dari biaya nyata |
| E-5 | Harga plan (Rp/bulan) adalah **revenue tetap**; biaya token adalah **biaya variabel** |

> **Inilah keindahan ide user:** menaikkan rasio membuat user lebih senang (kuota
> lebih tahan lama) **tanpa** mengubah margin per pemakaian. Yang berubah hanya
> berapa banyak chat yang "muat" dalam kuota.

---

## 2. Rumus

### 2.1 Konversi kuota

```
unit_per_request = ceil(raw_tokens / token_quota_ratio)

chats_per_cycle_exact = ai_tokens_per_cycle / unit_per_chat
chats_per_cycle       = floor(chats_per_cycle_exact)
leftover_units        = ai_tokens_per_cycle - (chats_per_cycle × unit_per_chat)
```

### 2.2 Biaya (dari token nyata)

```
sell_price_per_1k   = price_unit == "per_10k" ? base_per_1k_idr / 10 : base_per_1k_idr

sell_per_request    = floor( sell_price_per_1k × (raw_tokens / 1000)
                             × markup_multiplier
                             × (1 + markup_percent/100)
                             × modality_multiplier )

provider_cost       = floor( provider_cost_per_1k_idr × (raw_tokens / 1000)
                             × modality_multiplier )

margin_per_request  = sell_per_request − provider_cost
margin_percent      = margin / sell × 100        (0 bila sell = 0)
```

> Rumus ini **identik** dengan `PricingEngine.CostIDR` & `ProviderCostIDR` yang sudah
> ada (`backend/internal/billing/pricing.go`). Tidak ada rumus baru.

### 2.3 Ekonomi siklus

```
biaya_siklus_penuh    = chats_per_cycle × provider_cost_per_request
pendapatan_token      = chats_per_cycle × sell_per_request
margin_kotor_siklus   = pendapatan_token − biaya_siklus_penuh
net_per_user          = harga_plan_bulanan + margin_kotor_siklus
```

> **Catatan:** `sell_per_request` adalah *harga internal* token (biaya yang
> "seharusnya" dibayar user). User tidak membayarnya langsung — ia membayar
> `harga_plan_bulanan`. Nilai `sell_per_request` dipakai untuk **mengukur nilai** dan
> **membandingkan** antar plan, serta sebagai dasar pricing di masa depan
> (mis. top-up token).

---

## 3. Data aktual (dari seed & dump)

### 3.1 Aturan harga aktif

Dari `database/dump/fintrack_v2.sql:1304` (rule `default`):

| Field | Nilai |
|---|---|
| `base_per_1k_idr` | `300` |
| `price_unit` | `per_1k` |
| `markup_multiplier` | `1.2500` |
| `markup_percent` | `0` |
| `modality_multipliers.chat` | `1.0` |
| `provider_cost_per_1k_idr` | **`0` (belum diisi!)** |

> ⚠️ **Temuan:** `provider_cost_per_1k_idr = 0` berarti **margin belum bisa dihitung
> dari data nyata**. Operator harus mengisi biaya provider dulu di
> `/super-admin/pricing`. Dokumen ini memakai asumsi ilustratif
> `provider_cost_per_1k_idr = 150` agar margin terlihat.
>
> **Revisi (keputusan P-11):** sebagai ganti `0`, sistem menyediakan **default
> dinamis** via setting `pricing_default_provider_cost_per_1k_idr` (diisi nilai awal
> wajar, mis. `150`). Nilai ini:
> - dipakai saat **membuat** aturan harga baru (agar margin langsung terukur),
> - dapat diubah operator kapan pun di `/super-admin/settings`,
> - **bukan** konstanta kode — hanya fallback bila setting kosong.

### 3.2 Plan aktual

Dari `database/seeds/0002_subscription_token_seeds.sql`:

| Plan | Harga/bln | Kuota | Permintaan | Fitur AI |
|---|---|---|---|---|
| Free | Rp 0 | 5.000 | 50 | chat saja |
| Pro | Rp 39.000 | 150.000 | 2.000 | + OCR, voice, RAG, laporan, budgeting |
| Business | Rp 129.000 | 1.000.000 | unlimited | + API |

---

## 4. Biaya satu chat

**Asumsi:** chat nyata = **9.945 token** (dari audit live), modalitas `chat`.

| Komponen | Rumus | Nilai |
|---|---|---|
| Harga jual/1k | `300 × 1.25` | Rp 375 |
| **Biaya jual/chat** | `floor(375 × 9,945)` | **Rp 3.729** |
| **Biaya provider/chat** | `floor(150 × 9,945)` | **Rp 1.491** |
| **Margin/chat** | `3.729 − 1.491` | **Rp 2.238 (60%)** |

> Angka ini berlaku **sama** untuk semua rasio. Rasio tidak menyentuh baris ini.

---

## 5. Simulasi kuota (inti jawaban untuk user)

### 5.1 Free (kuota 5.000) — efek rasio

| Rasio | Unit/chat | Chat/siklus | Sisa kuota | Biaya provider ditanggung |
|---|---|---|---|---|
| **1** (sekarang) | 9.945 | **0** ❌ | 5.000 | Rp 0 (user tak bisa chat) |
| 2 | 4.973 | 1 | 27 | Rp 1.491 |
| 3 | 3.315 | 1 | 1.685 | Rp 1.491 |
| **5** (usul) | 1.989 | **2** | 1.022 | Rp 2.982 |
| 10 | 995 | 5 | 25 | Rp 7.455 |
| 20 | 498 | 10 | 20 | Rp 14.910 |

> **Ini persis yang user maksud:** *"per 5k token itu menggunakan pemakaian dari sisi
> plan = 1k"* → rasio 5 membuat 9.945 token hanya menagih 1.989 kuota.

### 5.2 Pro (kuota 150.000)

| Rasio | Unit/chat | Chat/siklus | Sisa |
|---|---|---|---|
| 1 | 9.945 | 15 | 825 |
| **2** (usul) | 4.973 | **30** | 810 |
| 5 | 1.989 | 75 | 825 |

### 5.3 Business (kuota 1.000.000)

| Rasio | Unit/chat | Chat/siklus |
|---|---|---|
| **1** (usul, 1:1 transparan) | 9.945 | **100** |
| 5 | 1.989 | 502 |

---

## 6. Ekonomi per plan (usulan rasio)

**Asumsi:** chat = 9.945 token; rasio Free 5, Pro 2, Business 1; biaya provider Rp 1.491/chat.

| Plan | Harga/bln | Rasio | Kuota | Chat penuh | Biaya provider maks | Nilai token internal | Net margin |
|---|---|---|---|---|---|---|---|
| **Free** | Rp 0 | 5 | 5.000 | 2 | Rp 2.982 | Rp 7.458 | **−Rp 2.982** (subsidi) |
| **Pro** | Rp 39.000 | 2 | 150.000 | 30 | Rp 44.730 | Rp 111.870 | **Rp 72.870** |
| **Business** | Rp 129.000 | 1 | 1.000.000 | 100 | Rp 149.100 | Rp 372.900 | **Rp 243.900** |

### 6.1 Analisis

| Temuan | Implikasi |
|---|---|
| **Free disubsidi ~Rp 3.000/user/bulan** | Wajar sebagai biaya akuisisi — **tetapi** rasio tinggi (10–20) membuatnya Rp 7.500–15.000. Perlu batas |
| **Pro margin 65%** | Sehat. Bahkan bila user memakai seluruh kuota |
| **Business margin 65%** | Sehat. Bila provider naik 2×, margin turun ke ~30% — masih positif |
| **Rasio tidak mengubah margin/chat** | Hanya mengubah *berapa banyak* yang bisa dipakai |

### 6.2 Titik impas (break-even)

```
Break-even Free  = Rp 2.982 biaya/user/bulan (rasio 5)
Break-even Pro   = biaya provider maks Rp 44.730 < harga Rp 39.000?
                   → TIDAK impas bila user memakai 100% kuota (rugi Rp 5.730)
                   → impas bila user memakai ≤ 26 chat (87% kuota)
Break-even Business = Rp 149.100 > Rp 129.000
                   → TIDAK impas bila user memakai 100% kuota (rugi Rp 20.100)
                   → impas bila user memakai ≤ 86 chat (86% kuota)
```

> ⚠️ **PERINGATAN PENTING:** dengan asumsi biaya provider Rp 1.491/chat, plan **Pro dan
> Business bisa RUGI** bila user memakai seluruh kuota. Ini perlu keputusan bisnis:
> **naikkan harga plan**, **turunkan kuota**, **naikkan markup**, atau **isi
> `provider_cost_per_1k_idr` dengan nilai nyata** lalu evaluasi ulang.
>
> Angka riil bergantung pada biaya provider sebenarnya yang **belum diisi** di
> `pricing_rules`. Kalkulator Super Admin akan menampilkan ini secara otomatis.

---

## 7. Kalkulator Super Admin — output yang diharapkan

Dengan input: rasio 5, kuota 5.000, chat 9.945 token, rule `default`:

```
Kuota
  Unit per chat        1.989 kuota
  Chat per siklus      ~2 (tepatnya 2,51)
  Sisa akhir siklus    1.022 kuota

Biaya
  Harga jual/chat      Rp 3.729
  Biaya provider/chat  Rp 1.491
  Margin/chat          Rp 2.238 (60%)
  Biaya siklus penuh   Rp 2.982
  Nilai token siklus   Rp 7.458

Perbandingan
  Tanpa rasio (1:1)    0,50 chat
  Dengan rasio 5       2,51 chat   ✅ +402%

Peringatan
  ⚠️ Rasio 5 memberi ~2,5 chat/siklus. Pertimbangkan kuota ≥ 10.000
     bila ingin ≥ 5 chat.
```

---

## 8. Rekomendasi konfigurasi

| Plan | Kuota | Rasio | Chat efektif | Catatan |
|---|---|---|---|---|
| Free | 5.000 | **5** | ~2 | Minimal agar user bisa mencoba |
| Free (alternatif) | 10.000 | 5 | ~5 | Bila ingin pengalaman lebih baik (subsidi ~Rp 7.500) |
| Pro | 150.000 | **2** | ~30 | Sesuai kuota yang sudah ada |
| Business | 1.000.000 | **1** | ~100 | 1:1 agar transparan untuk power user |

### 8.1 Alternatif bila ingin "pertahankan" (tanpa rasio)

Jika tim memutuskan **tidak** memakai rasio, kompensasinya:

| Opsi | Efek |
|---|---|
| Naikkan kuota Free → 30.000 | ~3 chat, tapi angka terlihat besar |
| Batasi 1 permintaan AI per siklus untuk Free | Sangat ketat, tapi jelas |
| Ubah Free menjadi "trial 7 hari" | Model bisnis berbeda |
| Biarkan (rasio 1) | User Free tidak bisa chat sama sekali ❌ |

---

## 9. Sensitivitas (biaya provider naik-turun)

Rasio Free = 5, kuota 5.000, 2 chat penuh:

| Biaya provider/1k | Biaya/chat | Biaya siklus (2 chat) | Status |
|---|---|---|---|
| Rp 100 | Rp 994 | Rp 1.988 | Aman |
| **Rp 150** (asumsi) | Rp 1.491 | Rp 2.982 | Aman |
| Rp 300 | Rp 2.983 | Rp 5.966 | Perlu perhatian |
| Rp 500 | Rp 4.972 | Rp 9.944 | ⚠️ Subsidi besar |
| Rp 1.000 | Rp 9.945 | Rp 19.890 | ⚠️ Tidak berkelanjutan |

> Karena `provider_cost_per_1k_idr` saat ini **0**, kalkulator akan menampilkan margin
> 100% (menyesatkan). **Langkah pertama operator:** isi biaya provider nyata.

---

## 10. Keputusan yang perlu dikonfirmasi

| # | Pertanyaan | Opsi | Rekomendasi |
|---|---|---|---|
| **P-2** | Rasio awal | Free 5 / Pro 2 / Business 1 | Sesuai tabel §8 |
| **P-9** | Kuota Free | tetap 5.000 / naikkan 10.000 | **10.000** — memberi ~5 chat, subsidi wajar |
| **P-10** | Harga Pro/Business | tetap / naikkan | **Evaluasi ulang** setelah biaya provider diisi; saat ini berpotensi rugi |
| **P-11** | Isi `provider_cost_per_1k_idr` | — | **WAJIB** — diisi **default dinamis** (`pricing_default_provider_cost_per_1k_idr`), dapat diubah operator |

---

**Navigasi:** [← Desain UI](./04-desain-ui.md) · [Rencana Implementasi →](./06-rencana-implementasi.md)
