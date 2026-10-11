# Kebijakan Metering Token — FinTrack v2

## 1. Tujuan

Mencatat pemakaian token secara **akurat, adil, idempoten**, dan dapat diaudit — sebagai dasar
penagihan kuota user dan pengendalian biaya platform.

## 2. Sumber Angka Token

| Prioritas | Sumber | Catatan |
|---|---|---|
| 1 | **Usage dari respons provider** | Paling akurat; diutamakan |
| 2 | **Estimasi lokal** | Fallback bila usage tidak tersedia |
| 3 | Pengali per-modality | Menyesuaikan vision/voice/embedding |

## 3. Alur Wajib (reserve → commit → release)

```
1. Pre-check entitlement (fitur diizinkan?)
2. RESERVE  → token_ledger(entry_type=reserve, amount=estimasi_maks)
              token_periods.reserved += estimasi
3. CALL     → aigateway.Execute(...)
4. COMMIT   → token_ledger(entry_type=commit, amount=usage_nyata)
              token_ledger(entry_type=release, amount=sisa_reserve)
              token_periods.used += usage_nyata; reserved -= estimasi
5. GAGAL    → token_ledger(entry_type=release, amount=estimasi)
```

## 4. Aturan Idempotensi

- `token_ledger.idempotency_key` **UNIQUE** (mis. `request_id`).
- Retry request yang sama tidak menggandakan commit.
- Rekonsiliasi berkala: total commit vs usage provider.

## 5. Kuota

- Kuota per siklus (`token_periods.quota`).
- Habis → **soft-limit**: blokir request berikutnya (402 + tawaran upgrade).
- Request yang sedang berjalan **tidak dipotong**.
- **Tidak ada rollover** antar-siklus.

## 6. Harga

```
harga = base_per_1k_idr × (token/1000) × markup_multiplier × (1 + markup_percent/100) × modality_multiplier
```

- Nilai uang = **integer rupiah**.
- Dapat berbeda per plan.

## 7. Audit & Transparansi

- User dapat melihat pemakaian token & sisa kuota.
- Super admin dapat melihat pemakaian per user & menyesuaikan (dengan audit).
- Semua penyesuaian manual (`adjust`) tercatat.

## 8. Anti-Penyalahgunaan

- Rate limit request AI.
- Deteksi lonjakan pemakaian tidak wajar (> 3× rata-rata) → alert.
- Batas per hari (`max_ai_requests_per_day`) per plan.

## Referensi

- `../../2_Projects/P04_Subscription-Token-Core/teknis/token-ledger.md`
- `../billing-pembayaran/KEBIJAKAN_BILLING.md`
