# Monitoring & Observability — FinTrack v2

## 1. Four Golden Signals

| Sinyal | Metrik | Sumber |
|---|---|---|
| **Latency** | p50/p95/p99 per endpoint | middleware/log |
| **Traffic** | request/detik, request AI/detik | log & gateway |
| **Errors** | 4xx/5xx rate, error gateway | middleware & aigateway |
| **Saturation** | pool DB, memori Redis, concurrency | infra |

## 2. Metrik Bisnis (wajib)

| Metrik | Definisi | Ambang alert |
|---|---|---|
| Token usage per user/plan | total token/commit per siklus | lonjakan > 3× rata-rata |
| Biaya LLM | token × harga dasar | mendekati/melampaui harga plan |
| Payment success rate | paid / total | < 95% |
| Failover count | perpindahan endpoint | > N/jam |
| Gateway error rate | error/response 5xx | > 5% |
| Dunning rate | langganan past_due | naik tajam |
| Entitlement cache hit/miss | rasio | hit rendah = beban DB naik |
| Kuota hampir habis | user > 90% kuota | untuk tawaran upgrade |

## 3. Kesehatan AI Gateway

- Status tiap `ai_endpoints` (`health`: up/down/degraded) — tampil di dashboard super admin.
- `last_check_at` & `last_error` per endpoint.
- Circuit breaker state (closed/open/half-open) — jumlah endpoint terbuka.

## 4. Dashboard

- **Super admin**: MRR, user aktif, token usage, payment success, status gateway, antrean review.
- **Operasional**: latency, error rate, saturasi, failover.
- **User**: kuota token terpakai/sisa, riwayat pembayaran.

## 5. Logging

- Structured (JSON), level sesuai lingkungan.
- **Jangan** mencatat secret/token/data kartu.
- Audit log terpisah (append-only) untuk aksi sensitif.

## 6. Alerting

| Alert | Ambang | Aksi |
|---|---|---|
| Gateway error rate | > 5% (5 menit) | Periksa endpoint, failover |
| Semua endpoint down | ≥ 2 endpoint | Kill switch + eskalasi |
| Payment success turun | < 95% | Periksa webhook/gateway |
| Lonjakan token | > 3× rata-rata | Periksa penyalahgunaan |
| DB pool saturasi | > 80% | Skala/pool tuning |
| Cache hit rendah | < 50% | Periksa kunci/Redis |

## Referensi

- Google SRE Book — *Four Golden Signals*
- `../../2_Projects/P08_Hardening-GoLive/teknis/load-test.md`
