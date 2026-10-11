# Batas Modul & Prinsip Isolasi — FinTrack v2

## 1. Prinsip

Setiap modul harus bisa dijawab tiga pertanyaan ini **tanpa membaca isinya**:
1. **Apa** yang dikerjakan?
2. **Bagaimana** memakainya (interface)?
3. **Apa** ketergantungannya?

Modul kecil & berbatas jelas lebih mudah dipahami, diuji, dan diubah.

## 2. Modul Baru (Backend)

| Modul | Tanggung jawab | Interface kunci | Dependensi |
|---|---|---|---|
| `internal/aigateway` | Pool endpoint, routing, failover, circuit breaker | `Router.Resolve(useCase)`, `Router.Execute(...)` | domain, crypto |
| `internal/billing` | Plan, subscription, ledger token, harga | `Ledger.Reserve/Commit/Release`, `Plans.Resolve(user)` | domain, entitlement |
| `internal/entitlement` | Resolver hak akses berversi + guard | `Guard.Require(featureKey)`, `Guard.WithinLimit(limitKey)` | billing, cache |
| `internal/payment` | Gateway (Midtrans/manual), webhook, invoice, prorasi | `Gateway.CreateCharge/VerifyWebhook/GetStatus` | billing |
| `internal/realtime` | Hub SSE + jembatan Redis pub/sub | `Hub.Publish(topic,event)`, `Hub.Subscribe(userID)` | redis |
| `internal/audit` | Audit log append-only | `Audit.Record(entry)` | domain |

## 3. Aturan Batas

1. **Satu modul, satu tujuan.** Modul tidak boleh mengerjakan dua hal berbeda.
2. **Komunikasi lewat interface**, bukan mengakses internal modul lain langsung.
3. **Dependensi searah** — hindari siklus (mis. `entitlement` → `billing`, bukan sebaliknya).
4. **Dapat diuji independen** — setiap modul punya test tanpa perlu seluruh sistem.
5. **Modul besar = sinyal untuk dipecah.**

## 4. Adapter ke Kode Lama

- `ai_usecase.go` yang lama **tidak ditulis ulang**; ia memanggil `aigateway` lewat adapter tipis.
- `middleware/rbac.go` diperluas, `middleware/cache.go` diganti implementasi Redis (interface sama).
- Ini menjaga zero downtime & mengurangi risiko.

## 5. Contoh Aliran

```
handler → entitlement.Guard.Require(feature)
        → billing.Ledger.Reserve(...)
        → aigateway.Router.Execute(...)
        → billing.Ledger.Commit(...)
        → audit.Record(...)
        → realtime.Hub.Publish(...)
```

## Referensi

- `STANDAR_KODE.md`
- `../../2_Projects/P02_Fondasi-Platform/teknis/`
