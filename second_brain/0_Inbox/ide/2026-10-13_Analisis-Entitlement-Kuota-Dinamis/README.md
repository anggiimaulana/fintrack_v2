# Analisis Entitlement, Kuota, & Dinamisasi Plan — FinTrack v2

| Field | Isi |
|---|---|
| **Dokumen** | Analisis Arsitektur Entitlement, Kuota Token, & Dinamisasi Admin |
| **Tanggal** | 2026-10-13 |
| **Status** | Analisis selesai — siap eksekusi |
| **Penulis** | Tim Rekayasa FinTrack v2 |
| **Cakupan** | `internal/entitlement`, `internal/billing`, `internal/delivery/http/middleware`, `internal/admin`, frontend billing & guards |

---

## 1. Latar Belakang

Dua pertanyaan muncul saat meninjau halaman Langganan:

1. **Bagaimana fitur plan bekerja?** Apakah levelnya mirip RBAC? Apa yang terjadi jika
   plan rendah mencoba mengakses fitur plan tinggi? Bagaimana race condition-nya?
2. **Bagaimana kuota dihitung?** Bagaimana implementasinya di sisi user & super admin,
   apa yang terjadi saat kuota menipis, dan bagaimana bila chat/voice/struk dijalankan
   bersamaan pada sisa kuota yang tipis?
3. **Permintaan tambahan:** semua metrik fitur, kuota, biaya/markup, subscription, dan plan
   harus **dinamis dari Super Admin — tanpa hardcode**.

Dokumen ini menjawab ketiganya dengan bukti kode + bukti uji live, lalu memetakan
**gap**, **rencana implementasi**, dan **rencana pengujian**.

---

## 2. Ringkasan Eksekutif (TL;DR)

> **Entitlement bukan RBAC.** RBAC = peran tetap (`user`/`admin`/`super_admin`, dicek di
> JWT+middleware). Entitlement = hak akses dinamis per-user yang **diturunkan dari
> langganan → plan → fitur/limit**. Keduanya independen dan berlapis: RBAC dulu, lalu
> entitlement.

> **Batas fitur di plan SUDAH aman** (backend otoritatif, fail-closed, super_admin bypass).
> Tidak ada cara user Free "menabrak" fitur Pro — API menolak dengan 403/402.

> **TAPI ada 4 celah serius pada kuota** yang membuat pembatasan **tidak benar-benar
> berjalan**:
> 1. 🔴 **OCR/vision tidak pernah menulis ke ledger** → `ocr_scans_per_cycle` **tidak pernah
>    bertambah** → OCR Pro/Business **tak terbatas** (terbukti live: 4 scan → 0 entri ledger).
> 2. 🔴 **Chat menagih 1 token tetap** per pesan, bukan token asli → kuota 150.000 token
>    nyaris tidak bergerak (terbukti: 15 commit bernilai `1`).
> 3. 🔴 **`ai_requests_per_cycle` tidak ditegakkan** di mana pun → angka 50/2.000/∞ **hiasan**.
> 4. 🔴 **Pricing engine tidak pernah di-wire** → `cost_idr` selalu `0` → biaya AI tidak
>    terlacak, dan `base_per_1k_idr`/`markup` **tidak berpengaruh**.

> **Hardcode yang tersisa:** `pricing_rules` **tidak punya endpoint admin** (harga/markup
> hanya bisa diubah via SQL). Plan, entitlement, dan limit **sudah** bisa diubah dari admin.

| Aspek | Status |
|---|---|
| Batas fitur (feature gate) | ✅ Berjalan |
| Batas limit transaksi/kategori/dompet | ⚠️ Sebagian (transaksi ✅; kategori/dompet belum — F4) |
| Batas kuota token | ✅ Diperbaiki (F1: token asli per AI call) |
| Batas kuota OCR | ✅ Diperbaiki (F1: vision dibebankan ke ledger) |
| Batas request AI | ✅ Diperbaiki (F1: `ai_requests_per_cycle` ditegakkan) |
| Biaya & markup (pricing) | 🔴 Tidak aktif (selalu 0) — F2 |
| Plan/entitlement/limit dinamis dari admin | ✅ Ya |
| Pricing rule dinamis dari admin | ❌ Tidak (SQL only) — F5 |

---

## 3. Daftar Dokumen

| # | Dokumen | Isi |
|---|---|---|
| **00** | `README.md` (dokumen ini) | Ringkasan eksekutif, peta temuan, indeks |
| **01** | [`01-analisis-arsitektur-entitlement.md`](01-analisis-arsitektur-entitlement.md) | Arsitektur entitlement vs RBAC, alur resolusi, cache berversi, penegakan, race condition, isolasi plan |
| **02** | [`02-kalkulasi-kuota-dan-ledger.md`](02-kalkulasi-kuota-dan-ledger.md) | Siklus kuota, rumus, token ledger 2-fase (reserve/commit/release), skenario kuota menipis & konkurensi |
| **03** | [`03-temuan-gap-dan-race-condition.md`](03-temuan-gap-dan-race-condition.md) | Semua temuan (bukti kode + uji live), dampak, akar masalah, analisis race condition |
| **04** | [`04-dinamisasi-admin-tanpa-hardcode.md`](04-dinamisasi-admin-tanpa-hardcode.md) | Inventaris hardcode, target "semua dinamis", desain endpoint & UI admin |
| **05** | [`05-implementation-plan.md`](05-implementation-plan.md) | Rencana implementasi berfase dengan checklist `[ ]` |
| **06** | [`06-testing-plan.md`](06-testing-plan.md) | Rencana pengujian lengkap dengan checklist `[ ]` |

---

## 4. Peta Temuan (Ringkas)

| ID | Temuan | Severity | Bukti |
|---|---|---|---|
| **GAP-01** | OCR/vision tidak menulis ledger → kuota OCR tak terbatas | 🔴 Kritis | ✅ **Diperbaiki (F1)** — live: OCR kini menulis `vision` entries |
| **GAP-02** | Chat menagih 1 token flat, bukan token asli | 🔴 Kritis | ✅ **Diperbaiki (F1)** — live: commit 9.945 token (sebelumnya 1) |
| **GAP-03** | `ai_requests_per_cycle` tidak ditegakkan | 🟠 Tinggi | ✅ **Diperbaiki (F1)** — live: 402 saat limit tercapai |
| **GAP-04** | Pricing engine tidak di-wire → `cost_idr=0` | 🟠 Tinggi | `NewPricingEngine`/`GetActiveRule` tidak dipanggil di `main.go`; DB: `cost_idr=0` |
| **GAP-05** | `max_categories`/`max_wallets` tidak punya usage resolver | 🟡 Sedang | `usage_resolvers.go` hanya 4 kasus (kategori/dompet belum) |
| **GAP-06** | `pricing_rules` tanpa endpoint admin (hardcode efektif) | 🟠 Tinggi | `internal/admin/` tidak menyentuh `pricing_rules` |
| **GAP-07** | User tidak melihat sisa kuota (tidak ada UI/monitoring mandiri) | 🟡 Sedang | grep frontend = 0 |
| **GAP-08** | Error kuota ledger (429) tidak dipetakan ke toast upsell di frontend | 🟡 Sedang | `useAiChat.ts` tanpa `notifyEntitlementError` |
| **BUG-01** | Sidebar: "Langganan" & "Pengaturan" aktif bersamaan | 🟢 Rendah | ✅ **Sudah diperbaiki** (`lib/nav.ts`, longest-prefix) |

---

## 5. Catatan Metodologi

- Semua klaim kode disertai `file:line`.
- Semua klaim perilaku diverifikasi terhadap **instance live** (PostgreSQL :5435, Redis :6380,
  API :8080) pada 2026-10-13.
- Kutipan SQL dari `docker exec fintrack_v2_db psql` disalin apa adanya.
- Dokumen 05 & 06 memakai checkbox `[ ]` untuk pelacakan eksekusi.

---

## 6. Keputusan Terkunci (Ringkas)

| Kode | Keputusan |
|---|---|
| **K-ENT-01** | Backend otoritatif; frontend hanya cermin UX (tidak pernah jadi gerbang) |
| **K-ENT-02** | Fail-closed: error resolusi → tolak akses, jangan lebarkan |
| **K-ENT-03** | super_admin bypass semua cek entitlement |
| **K-ENT-04** | Kuota paket = `402`; rate limit & kuota ledger = `429`; fitur = `403` |
| **K-ENT-05** | Semua nilai (harga, fitur, limit, markup) bersumber dari DB; admin otoritatif |
| **K-ENT-06** | Uang = integer Rupiah; token = integer; ledger append-only; mutasi idempoten |
| **K-ENT-07** | Token asli dari provider WAJIB di-commit (bukan estimasi tetap) |
