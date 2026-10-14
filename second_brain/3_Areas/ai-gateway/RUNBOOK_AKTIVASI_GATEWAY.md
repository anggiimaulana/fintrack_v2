# Runbook — Aktivasi & Verifikasi AI Gateway (9router)

Runbook ini menjelaskan cara **menyalakan**, **memverifikasi**, dan **mengubah**
AI Gateway terpusat. Sejak P03 contract phase, gateway adalah **satu-satunya
transport AI**: jalur BYOK per-user sudah dihapus, sehingga bila gateway mati
maka fitur AI mati (dengan pesan ramah, bukan fallback).

---

## 1. Prasyarat

- 9router berjalan dan dapat dijangkau dari backend. Contoh lokal:
  `http://127.0.0.1:20128/v1` (`GET /` → 307, `GET /health` → 404 cepat; keduanya
  berarti "hidup").
- Kunci API 9router (dibuat di dashboard 9router → API Keys). Endpoint `/v1`
  mengembalikan **401 "Missing API key"** tanpa kunci — itu normal.

Cek cepat:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:20128/        # 307
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:20128/health  # 404
```

## 2. Menyalakan (dua cara)

### Cara A — Bootstrap dari environment (sekali saja)

Isi env lalu start backend:

```dotenv
AI_GATEWAY_BASE_URL=http://127.0.0.1:20128/v1
AI_GATEWAY_API_KEY=sk-xxxx
AI_GATEWAY_ENABLED=true
AI_GATEWAY_MODEL_CHAT=gemini/gemini-3.5-flash-lite
AI_GATEWAY_MODEL_REASONING=deepseek-r1
AI_GATEWAY_MODEL_VISION=gemini/gemini-3.5-flash
AI_GATEWAY_MODEL_VOICE=whisper-1
AI_GATEWAY_MODEL_EMBEDDING=gemini/gemini-embedding-001
```

Bootstrap menyemai `ai_endpoints`, `ai_routes`, dan flag
`feature_ai_gateway_centralized` **hanya bila tabelnya kosong**. Log akan
menampilkan `AI gateway bootstrap: seeded from environment defaults`.

> **Penting:** setelah ada data di DB (dari Super Admin atau bootstrap
> sebelumnya), env **diabaikan** — database otoritatif. Jangan andalkan env untuk
> mengubah konfigurasi yang sudah berjalan.

### Cara B — 100% dari Super Admin UI/API

Biarkan `AI_GATEWAY_BASE_URL` kosong. Login sebagai `super_admin`, lalu:

1. `POST /api/v1/admin/ai-gateway/endpoints` — daftarkan endpoint
   (`base_url`, `api_key`, `provider`, `weight`). Kunci dienkripsi AES-256-GCM.
2. `PUT /api/v1/admin/ai-gateway/routes` — tetapkan rute per use-case
   (`chat`, `reasoning`, `vision`, `voice`, `embedding`) ke endpoint + model.
3. `PUT /api/v1/admin/feature-flags` — set `feature_ai_gateway_centralized`
   `is_enabled=true` (rollout > 0).

## 3. Syarat Gateway ON

Gateway hanya aktif bila **ketiganya** terpenuhi (lihat `cmd/server/main.go`):

1. Flag `feature_ai_gateway_centralized` ON (`IsEnabledForPlatform`).
2. Ada ≥1 rute `chat` aktif.
3. Ada endpoint `health = 'healthy'` untuk use case `chat`.

`unknown` **tidak** dihitung sehat. Health checker menjalankan satu probe
**sinkron** saat startup (`RunOnce`) lalu setiap 60s, jadi endpoint yang baru
di-seed langsung diprobe dan gateway bisa naik di boot yang sama.

## 4. Verifikasi

```bash
# 1) Health endpoint backend
curl -s http://localhost:8080/health/ready | jq .

# 2) Log startup harus memuat:
#    "AI gateway centralized: ENABLED"

# 3) Status endpoint (super_admin)
curl -s -b jar.txt http://localhost:8080/api/v1/admin/ai-gateway/endpoints
#    → "health":"healthy", "breaker_state":"closed", "has_api_key":true

# 4) Uji chat nyata (login user biasa lebih dulu)
curl -s -N -b jar.txt -X POST \
  "http://localhost:8080/api/v1/ai/chat?source=chatbot" \
  -H 'Content-Type: application/json' \
  -d '{"message":"Berapa total pengeluaran saya bulan ini?"}'
#    → event "done" memuat "model":"<model yang melayani>"

# 5) UJI RUTE (super_admin): probe model ASLI di setiap rute aktif.
#    Menangkap "endpoint sehat tapi model mati" (mis. 503 untuk model tak dikenal)
#    yang TIDAK terdeteksi oleh health check reachability.
curl -s -b jar.txt http://localhost:8080/api/v1/admin/ai-gateway/verify-routes | jq .
#    → { "results":[{...,"ok":true|false,"error":...}], "broken_count":N, "broken":[...] }
```

> **Verifikasi rute** juga berjalan otomatis sekali saat startup (di background,
> non-blocking) dan menuliskan peringatan `AI route verify: MODEL NOT RESPONDING`
> untuk setiap rute yang modelnya tidak menjawab. Read-only: tidak pernah
> mengubah rute/health/breaker, jadi tidak bisa mematikan AI.

## 5. Mengubah Konfigurasi (tanpa redeploy)

Semua perubahan berlaku langsung setelah disimpan (rute di-cache 60s dan
di-invalidate saat mutasi admin):

| Ingin mengubah | Endpoint |
|---|---|
| Model per use-case | `PUT /api/v1/admin/ai-gateway/routes` |
| Endpoint / kredensial / bobot | `POST` / `PUT /api/v1/admin/ai-gateway/endpoints` |
| Aktif/nonaktif endpoint | `PUT .../endpoints/:id` (`status=active|inactive`) |
| Flag gateway ON/OFF | `PUT /api/v1/admin/feature-flags` |
| Stop darurat semua AI | `POST /api/v1/admin/ai-gateway/kill-switch` |
| Reset circuit breaker | `POST /api/v1/admin/ai-gateway/circuit-breaker/reset` (tambahkan `use_case`+`model` untuk reset karantina satu rute) |
| Uji model tiap rute | `GET /api/v1/admin/ai-gateway/verify-routes` |

## 6. Catatan Operasional

- **Model embedding** ditentukan oleh route `embedding` (bukan hardcode).
  Gunakan model yang mengembalikan 768d, mis. `gemini/gemini-embedding-001`
  (gateway mengirim `dimensions:768`). Model `text-embedding-004` **tidak**
  tersedia di 9router.
- **`gemini/gemini-3.5-flash-lite`** = model chat **CEPAT** (terukur 2026-10-14:
  5/5 sukses, ~0.7–1.5s, dukung tool-call). Dipakai untuk route `chat` prioritas 1.
- **`gemini/gemini-3.5-flash`** = failover chat + model **vision** (baca struk).
  Lebih lambat (~2–10s) tapi andal dan mendukung vision + tool-call.
- **`gemini-2.5-flash`** = **JANGAN dipakai lagi**: per 2026-10-14 model ini
  mengembalikan **503 (0/5)** via 9router — penyebab chat/vision mati. Migrasi
  `0017_fix_ai_route_models.sql` memindahkan chat & vision ke model di atas.
- **`deepseek-r1`** = model **PENALARAN**; chain-of-thought-nya lambat
  (terukur 4s–103s), karena itu route `reasoning` memakai `timeout_ms=120000`
  (dinaikkan dari 30000). Sebagian akun OpenRouter bisa kehabisan kredit →
  failover mencoba rute `chat` berikutnya bila combo reasoning gagal.
- **Failover cerdas (route-level breaker)**: kegagalan dikarantina **per-rute**
  (endpoint + model), bukan per-endpoint. Jadi satu model mati (mis. 503) tidak
  mematikan model lain yang sehat di endpoint yang sama. Ambang: 2 kegagalan
  beruntun → rute dilewati selama cooldown 60s, lalu dicoba lagi otomatis
  (pulih sendiri tanpa restart). Hanya kegagalan **transport** (connection
  refused/DNS) yang menandai endpoint-down dan mengarantina seluruh endpoint.
  Status terlihat di UI (badge "Dikarantina") dan field `route_state` pada
  `GET /admin/ai-gateway/routes`.
- **Alert realtime**: saat sebuah rute dikarantina (atau pulih), gateway
  memancarkan event `ai_route_quarantined` ke kanal admin (SSE). Super Admin
  melihatnya sebagai **toast** ("Model AI dikarantina" / "Rute AI pulih") di
  halaman mana pun di area super-admin, dan matriks rute otomatis ter-refresh.
  Berbeda dari `ai_endpoint_down` (level endpoint).
  Catatan: event ini bersifat operasional dan **tidak** masuk ke lonceng notifikasi
  per-user (`/me/notifications`) — memang hanya untuk admin.
- **`whisper-1`** bergantung pada kredit OpenAI upstream; jika 429, voice
  mengembalikan pesan ramah (bukan crash).
- Kill switch berlaku pada semua transport dan disiarkan via Redis Pub/Sub
  dalam ~100ms.
