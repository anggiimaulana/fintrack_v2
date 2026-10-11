# 🤖 3_Areas — AI Gateway

Area tanggung jawab berkelanjutan untuk **gateway AI terpusat** (9router) FinTrack v2.

## Scope

- Pool endpoint & kunci AI (9router, provider langsung).
- Routing per-use-case (chat, vision, voice, embedding, reasoning).
- Failover, circuit breaker, health check, kill switch.
- Metering token & biaya.
- Rotasi kunci & keamanan.

## Dokumen

| Dokumen | Isi |
|---|---|
| `README.md` | File ini |
| `ARSITEKTUR_GATEWAY.md` | Arsitektur pool, routing, failover |
| `RUNBOOK_GATEWAY_DOWN.md` | Prosedur saat gateway/endpoint down |
| `RUNBOOK_ROTASI_KUNCI.md` | Prosedur rotasi kunci tanpa downtime |
| `KEBIJAKAN_METERING.md` | Kebijakan pencatatan token |

## Prinsip

1. **Tidak ada single point of failure** — selalu ada endpoint cadangan.
2. **Kill switch** siap untuk menghentikan endpoint bermasalah segera.
3. **Kunci terenkripsi**; hanya preview di API.
4. **Semua panggilan tercatat** (usage, endpoint, latensi, error).
5. **Embedding dikunci 768d** (cocok dengan `vector(768)`).

## Referensi

- Fase terkait: `../2_Projects/P03_AI-Gateway-Terpusat/`
- Keamanan kunci: `../keamanan/SECRET_MANAGEMENT.md`
