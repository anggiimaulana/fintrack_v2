# P07 Slice E — Routing AI Dinamis: Combo, Weighted Round-Robin & Model per Kategori

**Date:** 2026-10-12
**Status:** Approved (pending written-spec review)
**Phase:** P07 (Rollout RBAC, Realtime & Standar) — slice 5
**Related:** TASK-P07-016/017 (gateway ops), P03 (AI Gateway) — memperbaiki agar P03 benar-benar berjalan

---

## 1. Problem

Permintaan: routing AI harus **dinamis** — banyak model per kategori (chat, vision,
embedding, reasoning, voice), dengan **failover** saat satu model limit, dan kemampuan
memilih model per kategori dari panel admin (bukan hardcode).

### 1.1 Yang SUDAH ada (hasil recon, bukan asumsi)

| Kapabilitas | Status | Bukti |
|---|---|---|
| Failover berjenjang | ✅ ada | `service.go` mengiterasi route berurut `priority ASC`; tiap gagal → `recordFailure` + lanjut |
| Banyak route per kategori | ✅ didukung | skema `ai_routes` tidak unik per `use_case`; UI punya `handleAddRow` |
| Pilih model per kategori | ✅ ada | kolom `model` per route, dapat diedit via `PUT /admin/ai-gateway/routes` |
| Circuit breaker | ✅ ada | `eligibleRoutes` melewati endpoint yang breaker-nya `open` |
| Validasi dimensi embedding | ✅ ada | `service.go` menolak vektor ≠ 768 (`ErrInvalidEmbeddingDimension`) |

**Kesimpulan:** failover berurut priority **sudah berjalan**. Yang belum ada hanya pembagian
beban dan pengelolaan `combo`.

### 1.2 Yang BELUM ada

| Kekurangan | Dampak |
|---|---|
| **Round-robin / weighted** | `weight` ada di skema & domain tetapi **tidak pernah dibaca** — dead field. Tidak ada cara membagi beban antar model setara |
| **`combo` tidak disimpan** | `PutRoutes` mengabaikannya; UI mengirim `combo`, handler membuangnya → nilai hilang setiap simpan |
| **`endpoint_id` tidak divalidasi** | `PutRoutes` hanya memvalidasi format UUID, tidak keberadaan/status endpoint |
| **Embedding tanpa `dimensions`** | `gemini-embedding-001` default **3072d** sedangkan kolom pgvector `vector(768)`; client tidak mengirim `dimensions` → vektor selalu ditolak |
| **Route default menunjuk model yang tidak dilayani** | terukur: vision `gpt-4o-mini` → 503, embedding `text-embedding-004` → 404, voice `whisper-1` → menggantung |

### 1.3 Fakta gateway (terukur 2026-10-11/12)

| Model | chat | vision | embedding |
|---|---|---|---|
| `deepseek-r1` | **200** | 400 | — |
| `gemini-2.5-flash` | **200** | **200** | — |
| `gemini-embedding-001` | — | — | **200** (3072d default; 768 dengan `dimensions`) |
| `gpt-4o-mini` | 503 | 503 | — |
| `text-embedding-004` | — | — | 404 |
| `whisper-1` (voice) | — | — | menggantung/429 — **belum ada yang bekerja** |

9router memiliki **17 provider** (cc, cx, gc, qw, if, ag, gh, kr, openai, anthropic, gemini,
openrouter, glm, kimi, minimax, …) sehingga katalog model luas; hanya sebagian tersedia
bergantung kredensial/kuota upstream.

---

## 2. Goals / Non-goals

**Goals**

1. **Weighted smooth round-robin antar `combo`**, failover berurut `priority` **di dalam** combo.
2. **`combo` dapat dikelola** lewat API admin (tersimpan, tervalidasi).
3. **Model per kategori dapat dipilih admin** tanpa deploy (sudah didukung; diperbaiki agar andal).
4. **Embedding mengirim `dimensions`** sehingga vektor cocok dengan kolom 768.
5. **Route default diarahkan ke model yang terbukti bekerja**, dengan failover ke cadangan.

**Non-goals**

- **Tidak** menaruh counter round-robin di Redis (keputusan eksplisit: state lokal + seed acak).
- **Tidak** menyelesaikan voice (belum ada model yang terbukti bekerja) — dicatat sebagai gap.
- Tidak mengubah alur bisnis chat/vision/voice selain pemilihan transport.
- Tidak menghapus BYOK (slice D2).

---

## 3. Design

### 3.1 Model pemilihan route

`eligibleRoutes` (kini hanya menyaring) diperluas menjadi **menyaring dan mengurutkan**:

```
1. Filter: endpoint.status == "active" DAN breaker.Allow(endpoint)
2. Kelompokkan per route.combo (kosong -> "default")
3. Hitung bobot tiap combo = jumlah endpoint.weight anggotanya
4. Pilih combo dengan SMOOTH weighted round-robin
5. Di dalam combo terpilih: urutkan priority ASC  (failover berurut)
6. Bila semua route combo habis/gagal -> lanjut combo berikutnya menurut urutan WRR
```

**Smooth weighted round-robin (algoritma nginx)**, bukan "weight sebagai rentang":

```
current[c] += weight[c]           // tiap langkah
pilih c dengan current[c] terbesar
current[c] -= totalWeight
```

**Mengapa bukan rentang naif:** dengan bobot 200/100, cara rentang menghasilkan **200 request
pertama semuanya ke combo A** (burst). Smooth WRR menyelang-seling (`A B A A B A A B A`) dengan
rasio tetap tepat 2:1. Diverifikasi lewat simulasi: distribusi 3000 request = 66.7% / 33.3%
(persis ideal).

### 3.2 State round-robin: in-memory per instance

Counter WRR disimpan **di memori `Service`**, dilindungi `sync.Mutex`, **di-seed acak saat
konstruksi**.

**Keputusan & alasannya (didiskusikan eksplisit):**

| Aspek | In-memory (dipilih) | Redis (ditolak) |
|---|---|---|
| Round-trip per request | **0** | 1 (`INCR`) |
| Latensi | operasi angka di memori | ~0.1–1ms + risiko timeout |
| Redis mati | tetap jalan | perlu fallback, kalau tidak gateway mati |
| Konsistensi | per-instance | global |

**Mengapa konsistensi global tidak diperlukan di sini:** pembagian beban **menginginkan**
keputusan lokal. Simulasi membuktikan distribusi agregat tetap tepat bahkan ketika beberapa
instance berjalan "lockstep" (66.7/33.3). Sebaliknya, **kuota** memang butuh state global —
tetapi itu kebutuhan berbeda dan dapat ditambahkan terpisah (dicatat sebagai non-goal).

Preseden codebase: rate limiter pun memakai `memory.NewStore()` lokal, bukan Redis.

**Seed acak wajib:** tanpa itu, instance yang start bersamaan memilih combo yang sama di setiap
langkah (terbukti di simulasi: 18 request berturut-turut ke combo yang sama). Seed acak
menghilangkan korelasi tersebut.

**Trade-off yang dicatat:** urutan tidak global-koheren antar instance; distribusi agregat tetap
benar. Naikkan ke Redis hanya bila nanti butuh kuota global.

### 3.3 Perbaikan API admin

| Perbaikan | Detail |
|---|---|
| Simpan `combo` | tambahkan `Combo string` ke `routeDTO` + `INSERT ... combo` |
| Validasi `endpoint_id` | cek endpoint ada **dan** `status='active'`; tolak dengan 400 `invalid_endpoint` |
| Validasi duplikat | tolak `(use_case, endpoint_id, model, combo)` duplikat dalam satu payload |
| Urutan aman | **validasi seluruh payload sebelum transaksi**, agar tidak terjadi `DELETE` lalu gagal |
| `weight` | sudah tersimpan di CRUD endpoint (`COALESCE($7, weight)`); dipastikan ikut terbaca di `ListGatewayEndpoints` (sudah) |

### 3.4 Embedding: kirim `dimensions`

- `client.Embedding` menerima parameter dimensi opsional; payload menyertakan
  `"dimensions": 768` bila diisi.
- `gateway.Service.CreateEmbedding` meneruskan 768 (konstanta bersama dengan validasi kolom).
- Validasi ketat `len(vec) != 768` yang sudah ada **dipertahankan** — itu jaring pengaman, bukan
  pengganti parameter request.

### 3.5 Route default (migrasi baru, bukan mengedit yang lama)

Migrasi `0012_ai_routes_multi_model.sql` mengisi route berbasis model yang **terbukti bekerja**:

| use_case | combo | model | priority | status bukti |
|---|---|---|---|---|
| chat | default | `deepseek-r1` | 1 | 200 |
| chat | default | `gemini-2.5-flash` | 2 | 200 (failover di dalam combo) |
| chat | backup | `gemini-2.5-flash` | 1 | 200 |
| vision | default | `gemini-2.5-flash` | 1 | 200 |
| embedding | default | `gemini-embedding-001` | 1 | 200 (dengan `dimensions=768`) |
| reasoning | default | `deepseek-r1` | 1 | 200 |
| voice | default | `whisper-1` | 1 | **BELUM terbukti** — ditandai, tidak diklaim |

Migrasi bersifat aditif & idempoten (pola `NOT EXISTS`), mengikuti aturan
`database/migrations/README.md` yang baru ditetapkan.

### 3.6 Konsistensi dengan pengaman gateway (D1)

Pengaman di `main.go` (flag ON **DAN** route chat ada **DAN** endpoint sehat) **tidak diubah**.
Fallback BYOK untuk chat/vision/voice (slice D1) tetap berlaku, sehingga kegagalan gateway tidak
mematikan fitur.

---

## 4. Testing

### 4.1 Unit
- **Smooth WRR**: rasio tepat pada bobot 2:1; tidak ada burst (urutan menyelang-seling); bobot
  sama → bergiliran rata; satu combo → selalu combo itu; combo tanpa route eligible dilewati.
- **Grouping combo**: route tanpa `combo` dianggap `"default"`; beberapa combo diurutkan benar.
- **Failover di dalam combo**: route pertama gagal → route kedua combo yang sama dipakai
  (bukan langsung lompat ke combo lain).
- **Breaker**: endpoint `open` dilewati sebelum pemilihan.
- **Thread-safety**: pemilihan dari banyak goroutine tidak menghasilkan data race (`-race`).
- **`PutRoutes`**: menyimpan `combo`; menolak `endpoint_id` tidak ada; menolak duplikat; payload
  tidak valid tidak menghapus data lama.
- **Embedding `dimensions`**: payload memuat `dimensions`; vektor ≠ 768 tetap ditolak.

### 4.2 Integrasi (DB nyata)
- Route multi-model tersimpan & terbaca; urutan mengikuti WRR + priority.
- Migrasi `0012` idempoten (dijalankan dua kali → hasil sama).

### 4.3 Gerbang
`go vet ./... && go test ./... && go build ./...` hijau; `go test -race` pada paket gateway.

---

## 5. Dokumentasi

`teknis/` gateway (perilaku routing baru + diagram pemilihan), `tracking/tracking.md`,
`log_aktivitas.md`, dan `database/migrations/README.md` (contoh migrasi aditif multi-baris).

---

## 6. Risks

| Risiko | Mitigasi |
|---|---|
| Round-robin mengubah perilaku: request pertama tidak selalu ke combo utama | Disengaja; dicatat. `priority` di dalam combo tetap menentukan urutan utama |
| Burst akibat bobot naif | Smooth WRR (diverifikasi simulasi), bukan rentang |
| Multi-instance tidak koheren | Seed acak; distribusi agregat tetap benar (disimulasikan) |
| Data race pada counter | `sync.Mutex` + uji `-race` |
| Migrasi `0012` menghapus route manual admin | Migrasi hanya **menambah** route yang belum ada (idempoten, tanpa `DELETE`) |
| Embedding tetap gagal bila provider menolak `dimensions` | Fallback antar route; validasi 768 dipertahankan; kegagalan dicatat |
| Voice belum bekerja | **Tidak diklaim beres**; ditandai sebagai gap |
| `PutRoutes` gagal di tengah setelah `DELETE` | Validasi penuh **sebelum** transaksi |
