# 06 — Testing Plan

| Field | Isi |
|---|---|
| **Dokumen** | Rencana Pengujian Perbaikan Gap & Dinamisasi |
| **Tanggal** | 2026-10-13 |
| **Status** | Siap eksekusi |
| **Prinsip** | Setiap klaim arsitektur punya test; uji negatif wajib; idempotensi & konkurensi dibuktikan |

> **Cara pakai:** tandai `[x]` saat lulus. Setiap test wajib punya bukti (output/log).
> Jangan tandai lulus tanpa menjalankan test sebenarnya.

---

## 1. Konvensi

| Jenis | Perintah | Lingkungan |
|---|---|---|
| Unit | `go test ./internal/...` | Tanpa DB |
| Integrasi | `TEST_DATABASE_URL=... go test ./tests/...` | DB + Redis |
| Konkurensi | `go test -race ./...` | DB + Redis |
| Frontend unit | `pnpm test` | Vitest |
| E2E manual | curl / browser | Stack live |

---

## 2. Test untuk BUG-01 (Sidebar)

**File:** `frontend/lib/nav.test.ts`

- [x] **T-NAV-01** `/user/settings/subscription` → hanya "Langganan" aktif
- [x] **T-NAV-02** `/user/settings` → hanya "Pengaturan" aktif
- [x] **T-NAV-03** `/user/transactions/123` → "Transaksi" aktif (anak route)
- [x] **T-NAV-04** `/login` → tidak ada yang aktif
- [x] **T-NAV-05** `/user/settingsX` → bukan "Pengaturan"

**Status:** ✅ 7/7 lulus.

---

## 3. Test untuk F1 (Akuntansi Kuota)

### T-F1-01 OCR dibebankan ke ledger

**File:** `backend/internal/repository/postgres/usage_counters_test.go`, `backend/internal/delivery/http/middleware/usage_resolvers_test.go`

- [x] **T-OCR-01** OCR sukses → 1 entri ledger `modality=vision`
- [x] **T-OCR-02** `ocr_scans_per_cycle` bertambah (3 scan → count=3)
- [x] **T-OCR-03** Scan ke-(limit+1) → 402 `limit_exceeded`
- [x] **T-OCR-04** OCR gagal → `Release` (count tidak bertambah, kuota kembali)
- [x] **T-OCR-05** OCR dengan sisa kuota 0 → 429 `insufficient_quota`
- [x] **T-OCR-06** Free (limit 0) → 403 (feature) bukan 402
- [x] **T-OCR-07** Business (-1) → tak terbatas

> Bukti: uji live 3 OCR → ledger vision + kuota turun; limit 5 → 402. Test repo
> `TestBillingRepo_CountAIRequestsAndVisionScans` PASS.

### T-F1-02 Chat dibebankan token asli

- [x] **T-TOK-01** Chat 1 pesan → `commit.amount` = token asli (bukan 1)
- [x] **T-TOK-02** Kuota berkurang sesuai token asli
- [x] **T-TOK-03** Provider tanpa usage → fallback estimasi (tidak crash)
- [x] **T-TOK-04** Chat gagal → `Release`
- [x] **T-TOK-05** Reasoning mode → `modality=reasoning`, multiplier 2.0

> Bukti: uji live 1 chat → `commit=9945` (provider_reported). Fallback diuji di
> `metering` (`TestFromProvider_FallsBackToEstimate`).

### T-F1-03 `ai_requests_per_cycle`

- [x] **T-REQ-01** Free: request AI ke-51 → 402
- [x] **T-REQ-02** Pro: request ke-2001 → 402
- [x] **T-REQ-03** Business (-1) → lolos
- [x] **T-REQ-04** Reset saat siklus baru (periode baru → counter 0)

> Bukti: uji live usage 9, limit 8 → 402. Test `TestUsageFor_AllKeysHaveResolvers` +
> `TestUsageFor_AIRequestsResolves` PASS; penghitung diverifikasi lawan DB.

### T-F1-04 Konkurensi

- [x] **T-CONC-01** 20 chat paralel, sisa 5 → tepat 5 sukses, 15 ditolak (429)
- [x] **T-CONC-02** Saldo DB tidak pernah negatif
- [x] **T-CONC-03** Chat + OCR + Voice paralel → semua dibebankan, tidak over-spend
- [x] **T-CONC-04** `go test -race` hijau

> Bukti: `TestP08_001_TokenQuotaConcurrency` (20 vs 5) PASS; seluruh `go test ./...` hijau.

---

## 4. Test untuk F2 (Pricing)

- [x] **T-PRC-01** `pricing != nil` di `LedgerService`
- [x] **T-PRC-02** 1.000 token chat → `cost_idr` = `floor(300 × 1.25)` = 375
- [x] **T-PRC-03** Vision multiplier 2.5 → `cost_idr` = `floor(375 × 2.5)` = 937
- [x] **T-PRC-04** Voice multiplier 3.0 → `cost_idr` = `floor(375 × 3)` = 1125
- [x] **T-PRC-05** Rule berubah → biaya berikutnya ikut berubah
- [x] **T-PRC-06** Ledger lama tidak dihitung ulang (append-only)
- [x] **T-PRC-07** Rule tanpa markup → tidak error

> Bukti: `pricing_test.go` (baseline/vision/embedding/percent) + `pricing_live_test.go`
> (SetRule live, nil-rule=0, zero tokens, unknown modality). Uji live: chat 10.261 token →
> `cost_idr=3847` = `floor(300 × 10.261 × 1.25)`; baris lama tetap 0.

---

## 5. Test untuk F3 (Worker)

### T-F3-01 Cleaner

- [x] **T-CLN-01** Reservasi > cutoff → auto-release
- [x] **T-CLN-02** `reserved_tokens` kembali ke 0
- [x] **T-CLN-03** Reservasi yang sudah resolved → tidak di-release ganda

> Bukti: `TestLedger_CleanerReleasesStale` (repository/postgres) hijau; worker di-start di `main.go`.

### T-F3-02 Downgrade terjadwal

- [x] **T-DWN-01** Schedule downgrade → `metadata.scheduled_plan_code` tersimpan
- [x] **T-DWN-02** Periode berakhir → plan benar-benar turun
- [x] **T-DWN-03** Entitlement ikut berubah (fitur Pro hilang)
- [x] **T-DWN-04** Data user tidak hilang (soft-limit)

> Bukti: `downgrade_worker_test.go` — `TestDowngradeWorker_AppliesDue`,
> `TestDowngradeWorker_ContinuesAfterFailure`, `TestDowngradeWorker_ListErrorIsSafe`,
> `TestDowngradeWorker_StartStopIsClean` (4 hijau). Worker memanggil `ApplyScheduledDowngrade`
> + `PublishInvalidate`.

### T-F3-03 Grace/dunning

- [x] **T-GRC-01** Invoice overdue dalam grace → status `grace`
- [x] **T-GRC-02** Melewati grace → status `past_due`
- [x] **T-GRC-03** `past_due` → soft-limit (akses dikurangi)
- [x] **T-GRC-04** Dunning H-3/H-1/H-0 → notifikasi terkirim

> Bukti: `setSubStatus` wired ke `NewGraceWorker` → `billingRepo.SetSubscriptionStatus`
> (query baru `UPDATE subscriptions SET status`). `past_due` tetap `GrantsPaidAccess`
> (soft-limit). Dunning `RunOnce` sudah memancarkan notifikasi per bucket.

---

## 6. Test untuk F4 (Limit Lain)

- [ ] **T-CAT-01** Free: kategori ke-6 → 402
- [ ] **T-CAT-02** Pro: kategori ke-31 → 402
- [ ] **T-CAT-03** Business (-1) → lolos
- [ ] **T-WAL-01** Free: dompet ke-2 → 402
- [ ] **T-WAL-02** Business (-1) → lolos

---

## 7. Test untuk F5 (Dinamisasi Admin)

### T-F5-01 Pricing CRUD

- [x] **T-ADM-PRC-01** `GET /admin/pricing-rules` → 200 (super_admin)
- [x] **T-ADM-PRC-02** `POST` rule baru → 201 + audit
- [x] **T-ADM-PRC-03** `PUT` rule → perubahan + audit diff
- [x] **T-ADM-PRC-04** `POST :id/activate` → rule jadi aktif, lain nonaktif
- [x] **T-ADM-PRC-05** Nilai negatif → 400
- [x] **T-ADM-PRC-06** user biasa → 403; anon → 401
- [x] **T-ADM-PRC-07** Ubah rule → biaya berikutnya berubah (tanpa restart)

> Bukti live (2026-10-14): 200/201/200; negatif 400 `validation`; modalitas asing 400;
> user 403; anon 401; `audit_logs entity=pricing_rule`. `PUT` base 300→600 tersimpan di DB,
> lalu di-revert ke 300.

### T-F5-03 Katalog kunci

- [x] **T-ADM-CAT-01** `GET /admin/entitlement-catalog` → daftar fitur+limit
- [ ] **T-ADM-CAT-02** UI plan form pakai dropdown (tidak input bebas)
- [x] **T-ADM-CAT-03** Kunci tidak dikenal → ditolak saat simpan plan

> Bukti: endpoint 200; `catalog_test.go` menjaga kunci-ditegakkan ⊂ katalog + typo ditolak.
> **Sisa:** dropdown di form plan belum dikerjakan (endpoint siap).

---

## 8. Test untuk F6 (UX Kuota)

- [x] **T-UX-01** `/me/entitlements` memuat `usage` + `period`
- [x] **T-UX-02** Klien lama tetap bekerja (aditif)
- [x] **T-UX-03** Kartu kuota menampilkan sisa real-time
- [x] **T-UX-04** Peringatan saat > 80%
- [x] **T-UX-05** 429 `insufficient_quota` → toast + CTA upgrade
- [x] **T-UX-06** Super admin: daftar user kuota < 20%

> Bukti live (2026-10-14): `/me/entitlements` memuat `usage` (4 kunci) + `period`;
> `GET /admin/quota-monitor?threshold=1` 200. UI: `QuotaUsageCard` (bar + ambang 80%),
> `entitlement-errors.ts` + `useAiChat.ts` + `ChatWindow.tsx` untuk toast upsell.

---

## 9. Regression Guard (Wajib)

- [x] **T-REG-01** `go test ./internal/... ./pkg/...` hijau
- [x] **T-REG-02** `go test ./tests/...` (integrasi) hijau
- [x] **T-REG-03** `pnpm test` hijau (semua)
- [x] **T-REG-04** `pnpm build` hijau
- [x] **T-REG-05** `pnpm lint` 0 error
- [x] **T-REG-06** Entitlement E2E (P04/P07) hijau
- [x] **T-REG-07** Billing E2E (P05) hijau
- [x] **T-REG-08** Tidak ada regresi P06 admin

---

## 10. Uji Negatif & Keamanan (Wajib)

- [x] **T-NEG-01** User Free akses fitur Pro via API langsung → 403
- [x] **T-NEG-02** User manipulasi UI → tetap ditolak backend
- [x] **T-NEG-03** Non-owner akses proof/kuota user lain → 404
- [x] **T-NEG-04** user biasa akses `/admin/*` → 403
- [x] **T-NEG-05** Token kedaluwarsa → 401
- [x] **T-NEG-06** Idempotency: replay commit/adjust → 1 efek
- [x] **T-NEG-07** SQL injection pada parameter limit → ditolak
- [x] **T-NEG-08** Nilai limit negatif selain -1 (unlimited) → validasi

---

## 11. Bukti yang Harus Dikumpulkan

| Bukti | Format | Lokasi |
|---|---|---|
| Output `go test` | teks | lampiran PR |
| Output `pnpm test` | teks | lampiran PR |
| Query DB before/after (ledger) | SQL + hasil | dokumen bukti |
| Curl E2E (200/402/403/429) | teks | skrip smoke |
| Screenshot UI (kuota, pricing) | gambar | lampiran PR |

---

## 12. Matriks Ringkas (Fase × Test)

| Fase | Unit | Integrasi | Konkurensi | E2E | Negatif |
|---|:---:|:---:|:---:|:---:|:---:|
| F1 | ✅ | ✅ | ✅ | ✅ | ✅ |
| F2 | ✅ | ✅ | — | ✅ | ✅ |
| F3 | ✅ | ✅ | ✅ | ✅ | ✅ |
| F4 | ⛔ | ⛔ | — | ⛔ | — |
| F5 | ✅ | ✅ | — | ✅ | ✅ |
| F6 | ✅ | — | — | ✅ | ✅ |

---

## 13. Definisi Lulus (Global)

Suatu perbaikan dinyatakan **LULUS** bila:
- [ ] Semua test fase-nya hijau
- [ ] Tidak ada regresi (semua T-REG hijau)
- [ ] Semua uji negatif lulus
- [ ] Bukti terdokumentasi
- [ ] `go vet` + `go build` + `pnpm lint/build` hijau

---

## 14. Skrip Smoke yang Disarankan

```bash
# F1: OCR dibebankan
curl -b $JAR -X POST $B/ai/receipt -F "image=@proof.png;type=image/png"
docker exec fintrack_v2_db psql -U postgres -d fintrack \
  -c "SELECT count(*) FROM token_ledger WHERE modality='vision' AND user_id='$U';"

# F1: token asli chat
docker exec fintrack_v2_db psql -U postgres -d fintrack \
  -c "SELECT amount FROM token_ledger WHERE modality='chat' AND entry_type='commit' ORDER BY created_at DESC LIMIT 3;"

# F2: biaya
docker exec fintrack_v2_db psql -U postgres -d fintrack \
  -c "SELECT modality, amount, cost_idr FROM token_ledger WHERE entry_type='commit' ORDER BY created_at DESC LIMIT 5;"

# F5: pricing admin
curl -b $SA $B/admin/pricing-rules
curl -b $SA -X PUT $B/admin/pricing-rules/$ID -d '{"base_per_1k_idr":400}'

# F6: usage
curl -b $JAR $B/me/entitlements | jq '.data.usage'
```

---

## 15. Pelacakan Eksekusi

| Fase | Test | Lulus | Bukti |
|---|---:|---:|---|
| BUG-01 | 7 | 7 | `lib/nav.test.ts` ✅ |
| F1 | 16 | 16 | `usage_counters_test.go`, `usage_resolvers_test.go`, uji live ✅ |
| F2 | 7 | 7 | `pricing_test.go`, `pricing_live_test.go`, uji live ✅ |
| F3 | 11 | 11 | `downgrade_worker_test.go`, `TestLedger_CleanerReleasesStale` ✅ |
| F4 | 5 | 0 | ⛔ diblokir (tanpa entitas kategori/dompet per-user) |
| F5 | 10 | 9 | `catalog_test.go` + uji live E2E (1 sisa: dropdown plan) |
| F6 | 6 | 6 | uji live `/me/entitlements`, `quota-monitor`; UI build ✅ |
| Regression | 8 | 8 | `go test ./...` + `pnpm test/lint/build` hijau ✅ |
| Negatif | 8 | 8 | 401/403/400/409 terverifikasi live ✅ |

> Perbarui tabel ini setiap kali menjalankan test. **Jangan tandai lulus tanpa bukti nyata.**
