# Strategi Testing — FinTrack v2

## 1. Piramida Testing

```
        ▲  Manual / E2E (checklist siklus penuh)
       ███  Load & Chaos (k6, failover, replay webhook)
      █████  Kontrak API (envelope, kode status)
     ███████  Integrasi (Go, env-gated; webhook, ledger, isolasi)
    █████████  Unit (Go & Vitest; ledger, prorasi, resolver, breaker)
```

## 2. Unit (Go)

| Target | Yang diuji |
|---|---|
| Token ledger | reserve/commit/release, concurrency, idempotensi |
| Prorasi | perhitungan siklus per-hari, pembulatan ke bawah |
| Entitlement resolver | versi cache, invalidation |
| Circuit breaker | state machine closed/open/half-open |
| Signature webhook | valid/invalid/replay |
| Pricing engine | base + markup kali/persen + pengali modality |

Referensi pola yang sudah ada: `backend/internal/usecase/spk_queue_test.go`.

## 3. Integrasi (Go, env-gated)

- Reserve→call→commit end-to-end dengan endpoint AI **tiruan**.
- Webhook Midtrans replay 10× → 1 baris.
- Isolasi user tetap terjaga (tidak bocor antar-user).
- Optimistic lock (409).
- Referensi: `backend/internal/repository/postgres/integration_test.go`.

## 4. Kontrak API

- Envelope `{success, message, data, meta{total,page,per_page}}`.
- Kode status: 402 (kuota), 403 (fitur), 409 (konflik), 429 (rate), 503 (gateway down).

## 5. Frontend (Vitest)

- DataTable: sinkronisasi URL state (`?page=&sort=&q=`).
- Surgical cache update (setQueryData) & optimistic.
- SSE: reconnect & `Last-Event-ID`.
- Guard UI: menyembunyikan fitur di luar entitlement.
- Referensi: `frontend/lib/api.test.ts`, `format.test.ts`.

## 6. Load & Chaos (k6, di `tester/`)

| Skenario | Kriteria lulus |
|---|---|
| Kuota concurrency | 20 request paralel, kuota 5 → ≤ 5 lolos |
| Failover | matikan 1 endpoint → trafik pindah, user tanpa error |
| Replay webhook | 10× kirim → 1 baris pembayaran |
| Throughput API | p95 < 300ms (non-AI) |

## 7. Aturan

1. **Setiap klaim arsitektur punya test.** Tidak ada klaim tanpa bukti.
2. Test baru untuk setiap bug (regression).
3. Test tidak boleh bergantung urutan atau state global.
4. Data uji tidak boleh menyentuh produksi.

## Referensi

- `STANDAR_KODE.md`, `REVIEW_GUIDELINE.md`
- `../../2_Projects/P08_Hardening-GoLive/teknis/load-test.md`
