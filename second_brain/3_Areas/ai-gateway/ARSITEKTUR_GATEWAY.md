# Arsitektur AI Gateway — FinTrack v2

## 1. Ikhtisar

```
                         ┌──────────────────────────────┐
   Request AI  ────────▶ │  internal/aigateway.Router   │
   (chat/vision/voice/   │  - Resolve(use_case)         │
    embedding/reasoning) │  - Execute(endpoint, ...)    │
                         └──────────────┬───────────────┘
                                        │ pilih endpoint sehat
                                        ▼
                    ┌───────────────────────────────────────┐
                    │  Pool Endpoint (ai_endpoints)          │
                    │  #1 9router-A  (priority 1, health up) │
                    │  #2 9router-B  (priority 2, health up) │
                    │  #3 provider langsung (cadangan)       │
                    └───────────────────────────────────────┘
                                        │
                                        ▼
                            9router / LLM providers
```

## 2. Komponen

| Komponen | Tanggung jawab |
|---|---|
| **Router** | Resolve use-case → endpoint; eksekusi; failover |
| **Endpoint Pool** | Daftar endpoint + prioritas + health + status |
| **Routing Table** (`ai_routes`) | Pemetaan use-case → endpoint/model/combo |
| **Circuit Breaker** | Menonaktifkan endpoint yang gagal berulang (cooldown) |
| **Health Checker** | Ping berkala; memperbarui `health` & `last_check_at` |
| **Kill Switch** | Mematikan endpoint/use-case segera (manual) |
| **Metering** | Mencatat usage token per panggilan |

## 3. Alur Resolusi

```
Resolve(use_case):
  1. Ambil kandidat dari ai_routes WHERE use_case=? AND is_active=true
     ORDER BY priority ASC
  2. Saring: health=up AND breaker=closed AND status=active
  3. Jika kosong → fallback ke route default / error terkontrol
  4. Kembalikan endpoint terbaik
```

## 4. Failover

```
Execute(endpoint):
  - panggil endpoint
  - sukses  → breaker closed, health up, catat usage
  - 5xx/timeout → catat gagal; jika melewati ambang → breaker open; coba kandidat berikutnya
  - 429/quota   → tandai quota_exceeded; coba berikutnya
  - semua gagal → kembalikan 503 (pesan ramah)
```

## 5. Circuit Breaker (state machine)

| State | Perilaku |
|---|---|
| `closed` | Normal; meloloskan trafik |
| `open` | Gagal berulang → lewati endpoint (cooldown 30s) |
| `half-open` | Setelah cooldown → uji 1 request; sukses → `closed`, gagal → `open` |

Parameter (dapat disesuaikan): ambang gagal, cooldown, ukuran jendela.

## 6. Health Check

- Berkala (mis. setiap 30–60s) ping endpoint ringan 9router.
- Memperbarui `ai_endpoints.health` (`up`/`down`/`degraded`) & `last_check_at`.
- Status dipublikasikan ke dashboard super admin (realtime).

## 7. Routing per Use-Case

| Use-case | Contoh model/combo |
|---|---|
| `chat` | model percakapan cepat |
| `vision` | model multimodal (struk) |
| `voice` | transkripsi/audio |
| `embedding` | **dikunci 768d** (`text-embedding-004`) |
| `reasoning` | model penalaran lebih kuat |

Super admin dapat mengubah pemetaan tanpa deploy.

## 8. Embedding

- **Dimensi dikunci 768** agar cocok dengan `transaction_embeddings.embedding vector(768)`.
- Mengganti model embedding non-768 → **migrasi kolom + backfill** (bukan perubahan diam-diam).
- Jika embedder tidak tersedia → mode *disabled-graceful* (RAG dilewati, aplikasi tetap jalan).

## 9. 9router

- Default `base_url`: `http://localhost:20128` (dapat diubah super admin).
- Bersifat **OpenAI-compatible** → klien disederhanakan (chat/completions).
- Dapat memakai combo/kuota/key spesifik per use-case.

## 10. Keamanan

- Kunci disimpan terenkripsi (`ENCRYPTION_KEY`); API hanya menampilkan preview.
- Rotasi tanpa downtime (lihat `RUNBOOK_ROTASI_KUNCI.md`).
- Audit akses kunci.

## 11. Satu Transport & Bootstrap Dinamis

- **BYOK dihapus total.** Tidak ada lagi kunci AI per-user; `gemini_api_keys`
  sudah di-drop (migrasi `0014`). Gateway adalah satu-satunya transport AI, jadi
  bila gateway mati, fitur AI mengembalikan pesan ramah (bukan fallback).
- **Bootstrap env → DB.** `internal/aigateway/bootstrap` menyemai
  `ai_endpoints`/`ai_routes`/flag dari environment **hanya saat tabel kosong**.
  Begitu ada data (Super Admin atau bootstrap sebelumnya), DB otoritatif dan env
  diabaikan — semua perubahan berikutnya lewat `/api/v1/admin/ai-gateway/*`
  tanpa redeploy. Health checker menjalankan probe sinkron saat boot agar
  keputusan ON/OFF tidak membaca `unknown` yang basi.
- **Embedder mengikuti route** (`ai_routes` use_case `embedding`), bukan
  hardcode; gateway mengunci `vector(768)`.
- Prosedur lengkap: `RUNBOOK_AKTIVASI_GATEWAY.md`.

## Referensi

- `RUNBOOK_GATEWAY_DOWN.md`, `RUNBOOK_ROTASI_KUNCI.md`, `RUNBOOK_AKTIVASI_GATEWAY.md`, `KEBIJAKAN_METERING.md`
- `../../2_Projects/P03_AI-Gateway-Terpusat/teknis/`
