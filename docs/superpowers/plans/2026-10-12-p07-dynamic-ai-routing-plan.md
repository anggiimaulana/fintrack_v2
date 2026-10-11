# Implementation Plan — P07 Slice E: Routing AI Dinamis

| Field | Isi |
|---|---|
| **Fase** | P07 — Rollout RBAC, Realtime & Standar (**Slice E**) |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-12 |
| **Prasyarat** | Slice A–D1 selesai; gateway punya pengaman + fallback BYOK |
| **Spec desain** | `docs/superpowers/specs/2026-10-12-p07-dynamic-ai-routing-design.md` |
| **Status** | Siap dieksekusi |

---

## 1. Tujuan & Definition of Done

Membuat routing AI **dinamis**: banyak model per kategori, pembagian beban berbobot antar combo,
failover berurut di dalam combo, dan pengelolaan penuh dari panel admin.

**Definition of Done:**
- [ ] `combo` = grup failover; smooth weighted round-robin **antar** combo.
- [ ] Failover berurut `priority` **di dalam** combo (dipertahankan).
- [ ] Counter WRR in-memory per instance, **di-seed acak**, aman terhadap goroutine.
- [ ] `PutRoutes` menyimpan `combo`, memvalidasi `endpoint_id`, menolak duplikat, dan memvalidasi
      seluruh payload **sebelum** transaksi.
- [ ] Embedding mengirim `dimensions=768`; validasi 768 dipertahankan.
- [ ] Migrasi `0012` aditif & idempoten; route diarahkan ke model yang terbukti bekerja.
- [ ] `go vet`/`go test ./...`/`go build` hijau; `go test -race` pada paket gateway hijau.
- [ ] Dokumentasi + tracking diperbarui; gap (voice) dicatat jujur.

---

## 2. Urutan Eksekusi

| Wave | Langkah | Deskripsi | Prasyarat |
|---|---|---|---|
| **W1** | E-1 | `combo` pada `routeDTO` + `PutRoutes` menyimpan combo | — |
| **W1** | E-2 | Validasi `endpoint_id` (ada & active) + tolak duplikat + validasi sebelum transaksi | E-1 |
| **W1** | E-3 | Test `PutRoutes` (combo tersimpan, validasi, payload buruk tidak menghapus data) | E-2 |
| **W2** | E-4 | Smooth weighted round-robin + grouping combo di gateway | — |
| **W2** | E-5 | State WRR in-memory + seed acak + mutex | E-4 |
| **W2** | E-6 | Test WRR (rasio, anti-burst, failover dalam combo, breaker, race) | E-5 |
| **W3** | E-7 | `dimensions` pada client embedding + diteruskan service | — |
| **W3** | E-8 | Test embedding `dimensions` | E-7 |
| **W4** | E-9 | Migrasi `0012` route multi-model (aditif, idempoten) | E-1, E-4 |
| **W4** | E-10 | Terapkan ke `fintrack_test` + `fintrack`; verifikasi idempoten | E-9 |
| **W5** | E-11 | Verifikasi end-to-end terhadap 9router nyata | E-10 |
| **W5** | E-12 | Gerbang penuh (`vet`/`test`/`build`/`race`) | E-1…E-11 |
| **W5** | E-13 | Dokumentasi + tracking + log | E-12 |
| **W5** | E-14 | Commit | E-13 |

**Jalur kritis:** E-4 → E-6 (pemilihan route harus benar sebelum data diisi) dan E-9 → E-11
(migrasi sebelum verifikasi nyata).

---

## 3. Detail Langkah

### W1 — Pengelolaan combo lewat API admin

#### E-1 — Simpan `combo`
- **Target:** `backend/internal/admin/aigateway.go`
- **Langkah:**
  1. Tambah `Combo string \`json:"combo"\`` ke `routeDTO`.
  2. Sertakan `combo` di `INSERT INTO ai_routes (...)`; default `"default"` bila kosong.
  3. Sertakan `combo` di `SELECT` pembacaan `before` (audit diff) dan di `ListRoutes`.
- **Verifikasi:** unit test — combo tersimpan & terbaca kembali.

#### E-2 — Validasi payload
- **Target:** `backend/internal/admin/aigateway.go`
- **Langkah:**
  1. Validasi **seluruh** payload sebelum membuka transaksi: `use_case` valid, `endpoint_id`
     UUID valid **dan endpoint ada dengan `status='active'`**, `timeout_ms`/`max_retries` ≥ 0,
     dan tidak ada duplikat `(use_case, endpoint_id, model, combo)`.
  2. Bila validasi gagal → 400 dengan kode spesifik (`invalid_endpoint`, `duplicate_route`),
     **tanpa** menyentuh tabel.
  3. Baru setelah lolos: transaksi `DELETE` + `INSERT`.
- **Verifikasi:** E-3.

#### E-3 — Test `PutRoutes`
- **Kasus:** combo tersimpan; `endpoint_id` tidak ada → 400 dan **data lama utuh**; endpoint
  non-active → 400; duplikat → 400; payload valid → 200 dan matriks tersimpan; audit tercatat.

### W2 — Weighted round-robin antar combo

#### E-4 — Pemilihan route
- **Target:** `backend/internal/aigateway/gateway/service.go`
- **Langkah:**
  1. Pisahkan menjadi: `eligibleRoutes` (filter, tetap) → `orderedRoutes(ctx, useCase)` (filter +
     **urutkan** menurut combo WRR lalu priority).
  2. Kelompokkan per `route.Combo` (kosong → `"default"`).
  3. Bobot combo = jumlah `Endpoint.Weight` anggotanya (weight ≤ 0 dianggap 1 agar tidak hilang).
  4. Pilih combo dengan **smooth WRR**; di dalam combo urut `priority ASC`.
  5. Combo yang seluruh route-nya tidak eligible dilewati.
  6. Terapkan di keempat jalur: `ChatStream`, `ChatSync`, `CreateEmbedding`, `TranscribeAudio`.
- **Verifikasi:** E-6.

#### E-5 — State WRR
- **Target:** `backend/internal/aigateway/gateway/service.go`
- **Langkah:** field `rrMu sync.Mutex` + `rrState map[string]map[string]int64` (per `use_case` →
  per combo → `current`). Seed acak saat `New()`. Algoritma:
  ```
  current[c] += weight[c]
  pilih c dengan current[c] terbesar (seri → nama terkecil, deterministik)
  current[c] -= totalWeight
  ```
- **Verifikasi:** E-6 termasuk `-race`.

#### E-6 — Test WRR
- **Kasus:** bobot 2:1 → rasio ≈66.7/33.3 pada banyak panggilan; **tidak ada burst** (9 panggilan
  pertama bukan semuanya combo berbobot besar); bobot sama → bergiliran; satu combo → selalu itu;
  combo tidak eligible dilewati; **failover di dalam combo** (route 1 gagal → route 2 combo sama,
  bukan lompat combo); endpoint breaker `open` dilewati; aman pada `-race` dengan banyak goroutine.

### W3 — Embedding `dimensions`

#### E-7 — Kirim dimensi
- **Target:** `backend/internal/aigateway/client/openai_client.go`,
  `backend/internal/aigateway/gateway/service.go`
- **Langkah:** `Embedding(ctx, model, text string, dims int, timeout)`; payload menyertakan
  `"dimensions"` bila `dims > 0`. Service meneruskan `768` (konstanta bersama validasi).
  Validasi `len(vec) != 768` **dipertahankan**.
- **Verifikasi:** E-8.

#### E-8 — Test
- Payload memuat `dimensions` saat diminta; tidak memuat saat 0; vektor ≠ 768 tetap ditolak.

### W4 — Data route

#### E-9 — Migrasi `0012`
- **Target:** `database/migrations/0012_ai_routes_multi_model.sql`
- **Langkah:** tambahkan route (idempoten, `NOT EXISTS`, **tanpa** `DELETE`) untuk model yang
  terbukti bekerja; lihat tabel di spec §3.5. Set `combo` eksplisit.
- **Verifikasi:** E-10.

#### E-10 — Terapkan & idempotensi
- Terapkan ke `fintrack_test` dan `fintrack`; jalankan **dua kali** untuk membuktikan idempoten
  (jumlah baris tidak berubah).
- **Catatan:** migrasi hanya menambah; route manual admin tidak dihapus.

### W5 — Verifikasi & finalisasi

#### E-11 — Verifikasi terhadap 9router nyata
- Jalankan pemilihan route nyata (bukan mock) untuk chat/vision/embedding; catat model terpilih
  dan hasilnya. **Voice tidak diklaim** (belum ada model yang terbukti).
- Bila embedding tetap gagal, catat pesan upstream apa adanya.

#### E-12 — Gerbang penuh
`go vet ./... && go test ./... && go build ./... && go test -race ./internal/aigateway/...`

#### E-13 — Dokumentasi
`teknis/` gateway (perilaku routing baru), `tracking/tracking.md`, `log_aktivitas.md`.

#### E-14 — Commit

---

## 4. Matriks Bukti

| Langkah | Artefak | Bukti |
|---|---|---|
| E-1..E-3 | `admin/aigateway.go` + test | test `PutRoutes` |
| E-4..E-6 | `gateway/service.go` + test | test WRR + `-race` |
| E-7..E-8 | `client/openai_client.go`, `gateway/service.go` | test payload embedding |
| E-9..E-10 | `database/migrations/0012_*.sql` | idempotensi (2×) |
| E-11 | — | log pemilihan route nyata |

---

## 5. Perintah Verifikasi

```bash
cd backend && go vet ./... && go test ./... && go build ./...
cd backend && go test -race ./internal/aigateway/...
docker exec -i fintrack_v2_db psql -U postgres -d fintrack -f - < database/migrations/0012_ai_routes_multi_model.sql
```

---

## 6. Rollback

Revert commit slice E. Migrasi `0012` bersifat aditif; untuk membatalkan, hapus route yang
ditambahkan (disediakan pernyataan `down` di komentar migrasi). Perilaku routing kembali ke
priority berurut murni — aman, karena failover tidak berubah.

---

## 7. Gap yang dicatat (bukan diklaim)

1. **Voice belum punya model yang terbukti bekerja** — endpoint voice menggantung/429. Route voice
   dibiarkan dan **ditandai**, tidak diklaim beres.
2. **Kuota global lintas instance** (Redis) tidak dikerjakan — kebutuhan berbeda, non-goal.
3. **BYOK belum dihapus** (slice D2).
4. Template k8s & uji rolling restart 200 RPS (dari D1) tetap terbuka.
5. **Katalog model bergantung kredensial upstream** — 9router punya 17 provider, tetapi hanya
   sebagian model tersedia dengan kredensial saat ini; admin dapat menambah route kapan saja
   tanpa deploy.
