# 04 — Arsitektur Backend

**Status:** DRAFT  
**Bergantung pada:** [`03-model-domain-dan-skema-database.md`](./03-model-domain-dan-skema-database.md)

---

## 1. Prinsip

1. Backend tetap **Go / Gin / clean architecture**. Tidak ada framework baru.
2. **Kebijakan di data** (tabel `currencies`, `currency_rates`), **kemampuan di kode** (`internal/currency`).
3. Base currency IDR di-hardcode **satu tempat saja** (`currency.BaseCode = "IDR"`), dan hanya sebagai fallback bootstrap — bukan sebagai nilai bisnis.
4. Zona IDR (billing/AI-quota/token/invoice/metrik admin) **tidak boleh import** paket konversi baru.

---

## 2. Package Baru: `internal/currency`

Satu paket kecil, tanpa dependensi ke usecase/http (menghindari siklus).

```
backend/internal/currency/
    currency.go      // Code, Catalog, Rate, Money
    service.go       // Service: RateAsOf, ConvertToBase, ConvertFromBase
    errors.go        // ErrInactiveCurrency, ErrNoRate
```

### 2.1 Tipe inti

```go
package currency

const BaseCode = "IDR" // ponytail: base tetap; upgrade = baca currencies.is_base

type Catalog struct {
    Code          string
    Name          string
    Symbol        string
    Locale        string
    DecimalDigits int
    IsActive      bool
    IsBase        bool
    SortOrder     int
}

type Rate struct {
    CurrencyCode string
    RateToBase   decimal.Decimal // 1 unit currency = N IDR
    EffectiveAt  time.Time
    Source       string
}
```

**Desimal:** gunakan `github.com/shopspring/decimal` atau `pgtype.Numeric` yang sudah tersedia transitif lewat pgx. Rekomendasi: `shopspring/decimal` (sudah umum, tepat, tanpa float error). Bila belum ada di `go.mod`, alternatif tanpa dependensi baru: hitung dengan `big.Rat`/integer minor units. **Keputusan dependensi ada di D10 (dokumen 09).**

### 2.2 Service

```go
type Repo interface {
    ListActive(ctx context.Context) ([]Catalog, error)
    Get(ctx context.Context, code string) (*Catalog, error)
    RateAsOf(ctx context.Context, code string, at time.Time) (*Rate, error)
    AllRatesAsOf(ctx context.Context, at time.Time) (map[string]Rate, error)
}

type Service struct {
    repo Repo
    // cache di level repo atau usecase; lihat §5
}

// ConvertToBase mengembalikan nilai IDR + rate yang dipakai (snapshot).
func (s *Service) ConvertToBase(ctx context.Context, code string, amount float64, at time.Time) (base float64, rate Rate, err error)

// ConvertFromBase mengonversi agregat IDR ke mata uang tampilan memakai kurs pada `at`.
func (s *Service) ConvertFromBase(ctx context.Context, code string, base float64, at time.Time) (float64, Rate, error)
```

Aturan:
- `code == BaseCode` → `rate = 1`, tanpa query rate, tanpa error.
- `ConvertToBase` memakai kurs **as-of `at`** (tanggal transaksi), bukan kurs terbaru.
- Mata uang nonaktif → `ErrInactiveCurrency`.
- Tidak ada rate pada `at` → `ErrNoRate` (jangan menebak 1).

### 2.3 Pembulatan

| Langkah | Aturan |
|---|---|
| Hitung base | `round(amount × rate, 2)` (IDR 2 desimal) |
| Tampilkan agregat ke JPY | `round(base / rate_as_of, 0)` (JPY 0 desimal) |
| Simpan rate | `numeric(20,10)` apa adanya |

Pembulatan mengikuti `decimal_digits` mata uang tujuan saat **tampilan**, bukan saat simpan. `amount_base` selalu 2 desimal IDR.

---

## 3. Perubahan Domain

### 3.1 `domain.Transaction`

```go
CurrencyCode string     `json:"currency_code" db:"currency_code"`
ExchangeRate float64    `json:"exchange_rate" db:"exchange_rate"`
AmountBase   float64    `json:"amount_base" db:"amount_base"`
RateAt       *time.Time `json:"rate_at,omitempty" db:"rate_at"`
```

### 3.2 Request

`CreateTransactionRequest`:
```go
Amount       float64 `json:"amount" binding:"required,gt=0"`
CurrencyCode string  `json:"currency_code"` // opsional; default = display_currency user
```

`UpdateTransactionRequest`:
```go
Amount       *float64 `json:"amount"`
CurrencyCode *string  `json:"currency_code"`
```

- Bila `amount` diubah tetapi `currency_code` tidak, server memakai ulang `exchange_rate` lama? **Tidak.** Aturan: bila nominal atau mata uang berubah, server menghitung ulang `amount_base` memakai kurs as-of `rate_at` transaksi (agar nilai dasar historis konsisten). Keputusan D11 (dokumen 09).
- Client **tidak pernah** mengirim `exchange_rate` atau `amount_base`; itu otoritas server.

### 3.3 Profil

`domain.Profile` bertambah:
```go
DisplayCurrency string `json:"display_currency"`
```

---

## 4. Repository

### 4.1 Baru: `repository/postgres/currency_repo.go`

| Method | SQL inti |
|---|---|
| `ListActive` | `SELECT ... FROM currencies WHERE is_active ORDER BY sort_order` |
| `Get` | `SELECT ... FROM currencies WHERE code=$1` |
| `RateAsOf` | `SELECT ... WHERE currency_code=$1 AND effective_at<=$2 ORDER BY effective_at DESC LIMIT 1` |
| `AllRatesAsOf` | `DISTINCT ON (currency_code) ... WHERE effective_at<=$1 ORDER BY currency_code, effective_at DESC` |
| `InsertRate` | `INSERT INTO currency_rates ...` (tidak ada update) |
| `ListRates` | riwayat paginasi untuk UI admin |

### 4.2 `transaction_repo.go`

- `Create`: tambah kolom `currency_code, exchange_rate, amount_base, rate_at`.
- `GetByID`, `List`, `GetRecentByUser`: `SELECT` tambah 4 kolom.
- `Update`: `currency_code = COALESCE(...)`, `exchange_rate`, `amount_base`, `rate_at`.
- Agregat: `amount` → `amount_base` (daftar lengkap di dokumen 03 §7).
- **Sortir `amount`**: nominal asli. Catat sebagai keterbatasan (K-4 dokumen 09).

### 4.3 `user_repo.go`

- `SELECT`/`INSERT`/`UPDATE` profil tambah `display_currency`.
- Tambah method sempit `UpdateDisplayCurrency(ctx, userID, code)` — **jangan** mengubah `Update` full agar `PUT /me` tidak menimpa preferensi.

---

## 5. Cache & Invalidation

Pola yang sudah ada: `middleware.CacheMiddleware(30s)` + versi cache + Pub/Sub.

| Data | Cache | Invalidasi |
|---|---|---|
| Katalog `currencies` | in-memory TTL 60s **atau** Redis versi | saat admin ubah katalog |
| Kurs as-of | in-memory TTL 60s per (code, hari) | saat admin tambah rate |
| Preferensi `display_currency` user | ikut cache `/me` yang ada | saat user ubah preferensi |

Invalidasi: gunakan mekanisme **cache version bump + Pub/Sub** yang sudah dipakai `config_changed`, dan pancarkan event SSE yang sama supaya FE refresh. Tidak ada mekanisme cache baru.

> Kurs harian boleh di-cache agresif; laporan historis memakai kurs as-of tanggal sehingga hasilnya deterministik dan aman di-cache panjang (immutable per tanggal).

---

## 6. API

### 6.1 Publik/internal pengguna

| Method | Path | Fungsi |
|---|---|---|
| `GET` | `/api/v1/currencies` | Daftar mata uang **aktif** + simbol/locale/decimal_digits (tanpa rate bila sensitif) |
| `GET` | `/api/v1/currencies/rates?at=YYYY-MM-DD` | Kurs as-of untuk mata uang aktif (dipakai FE konversi agregat) |
| `PUT` | `/api/v1/me/preferences/currency` | Set `display_currency` user |

`GET /me` diperluas menyertakan `display_currency` (tidak mengubah kontrak lain).

**Kenapa endpoint preferensi terpisah?** `PUT /me` bersifat full-update (mengosongkan phone/avatar bila kosong). Menambahkan `display_currency` ke sana akan menghapus preferensi setiap kali user menyimpan nama. Preferensi butuh endpoint sempit sendiri. Ini juga menyiapkan tempat preferensi lain (mis. locale) tanpa menggemukkan `PUT /me`.

Request:
```json
PUT /api/v1/me/preferences/currency
{ "display_currency": "JPY" }
```

Validasi:
- kode ada, `is_active = true`, dan (bukan base) memiliki rate → else `422`.
- base currency boleh selalu dipilih.

### 6.2 Super Admin

| Method | Path | Fungsi |
|---|---|---|
| `GET` | `/api/v1/admin/currencies` | Katalog lengkap (termasuk nonaktif) — read-only untuk admin |
| `POST` | `/api/v1/admin/currencies` | Tambah mata uang (Super Admin) |
| `PUT` | `/api/v1/admin/currencies/:code` | Ubah metadata/aktifkan/nonaktifkan |
| `GET` | `/api/v1/admin/currency-rates?code=JPY&page=1` | Riwayat kurs |
| `POST` | `/api/v1/admin/currency-rates` | Tambah kurs baru (append-only) |

Wiring mengikuti pola `adminRoutes` (`RequireAdmin`) dan `sa` (`RequireSuperAdmin`) di `router.go`:
- **GET katalog/rate** → `adminRoutes` (read-only; D7).
- **POST/PUT** → group `sa` dengan `RequireSuperAdmin`.

Aturan aktivasi (di usecase):
- `PUT currencies/JPY { is_active: true }` **ditolak** bila belum ada baris rate valid. Cegah mata uang aktif tanpa kurs.
- Menonaktifkan mata uang: baris `currencies` tetap ada; transaksi lama tidak berubah. User yang sedang memakai mata uang itu harus di-arahkan ke IDR saat load (fallback) — FE + GET `/me` mengembalikan `display_currency` efektif.

### 6.3 Realtime

Manfaatkan event SSE `config_changed` yang sudah ada. Payload opsional: `{ "domain": "currency" }`. FE mendengarkan lalu invalidate query katalog + rate.

---

## 7. Audit

Semua mutasi katalog/kurs dicatat memakai pola audit yang ada (`audit_logs`, seperti `config_changed` di `internal/admin/settings.go`).

| Aksi | Field audit |
|---|---|
| `currency_created` | code, payload |
| `currency_updated` | code, before/after (is_active, metadata) |
| `currency_rate_added` | code, rate_to_base, effective_at, source, note |
| `user_currency_changed` | user_id, from, to (opsional; ringan) |

Kurs tidak di-`UPDATE`, jadi audit kurs hanya append. Ini menyederhanakan jejak.

---

## 8. Integrasi AI

Zona AI terbagi dua:

1. **Konteks keuangan pengguna** (`ai_usecase.go`) → **ikut display currency**.
   - Baris `Rp%.0f` pada ringkasan bulanan/tahunan/kategori/recent diganti formatter yang mengambil `display_currency` + kurs as-of sekarang.
   - Prompt tetap menyebut mata uang agar model tidak salah tafsir (mis. `¥1.000` bukan `Rp1.000`).
   - `agent_tools.go` deskripsi tool `create_transaction`: `"Amount in Rupiah"` → `"Amount in the user's display currency (e.g. IDR/JPY)"`.
   - `create_transaction` tool: server mengisi `currency_code` = display currency user dan menghitung `amount_base`.
2. **Biaya AI, token, kuota, estimasi** → **tetap IDR**, tidak disentuh. Jangan lewatkan nilai ini ke konversi.

Guardrail: uji regresi memastikan `billing`/`token`/`quota` tidak pernah memanggil `currency.Service`.

`buildEmbeddingText` (`transaction_usecase.go`) `Rp%.0f` → format dengan kode mata uang, mis. `JPY1000` / `IDR1000`. Metadata embedding tambah `currency_code`.

---

## 9. Integrasi Ekspor (`internal/export/export.go`)

- `rupiah()` → formatter generik `money(v float64, cur currency.Meta) string` membaca simbol + desimal.
- `ReportInput` tambah `Currency` (meta tampilan) + `RateAsOf` untuk legenda kurs.
- Baris header ekspor tambah: "Mata uang: JPY (1 JPY = 110,0000 IDR, kurs 2026-10-19)".
- Nilai yang diekspor = hasil konversi dari base ke display currency (D4). XLSX menulis angka numerik + format sel `#,##0` (JPY 0 desimal).
- `export.go` tetap transport-free; meta mata uang di-inject dari handler.

---

## 10. Budget

- `budgets.amount_idr` (bigint) **tetap** nilai dasar IDR.
- Input budget di FE (mata uang tampilan) dikonversi ke IDR di usecase `budget` memakai kurs as-of hari ini.
- Perbandingan budget vs aktual memakai `amount_base` → konsisten IDR.
- Tampilan budget di FE dikonversi balik ke display currency.

---

## 11. Laporan

- `report` usecase membaca agregat IDR (`amount_base`) lalu mengonversi ke display currency memakai **kurs as-of akhir periode** (bukan hari ini).
- Rekap `Summary`, `Categories`, `Monthly` dikonversi setelah agregasi IDR terbentuk (lebih murah & akurat: satu rate per laporan, bukan per baris).
- Handler mengirim `currency` + `rate_used` + `rate_at` supaya FE mencetak legenda.

---

## 12. Wiring & Urutan Boot

```
main.go
  → currencyRepo := postgres.NewCurrencyRepo(pool)
  → currencySvc  := currency.NewService(currencyRepo, cache)
  → txUsecase    := usecase.NewTransactionUsecase(..., currencySvc)
  → userUsecase  := usecase.NewUserUsecase(..., currencySvc)
  → reportUsecase, budgetUsecase, aiUsecase, export handler: inject currencySvc/meta
```

Registry/DI mengikuti pola yang sudah ada di `cmd/`/`router.go`; tidak ada container baru.

---

## 13. Error & Response

| Kondisi | HTTP | Kode |
|---|---|---|
| Mata uang tak dikenal | 422 | `currency_unknown` |
| Mata uang nonaktif | 422 | `currency_inactive` |
| Tidak ada kurs untuk tanggal | 422 | `currency_no_rate` |
| Ubah preferensi ke mata uang nonaktif | 422 | `currency_inactive` |

Gunakan helper `response` yang sudah ada. Pesan bahasa Indonesia, mis. "Kurs JPY belum tersedia. Hubungi admin."

---

## 14. Test Backend

| Test | Isi |
|---|---|
| `currency/service_test.go` | konversi IDR↔JPY, pembulatan JPY 0 desimal, rate as-of, error nonaktif |
| `currency_repo_test.go` (opsional, integrasi) | `DISTINCT ON` rate as-of, append-only |
| `transaction_usecase_test.go` | create JPY menghasilkan `amount_base` benar & snapshot rate |
| `report` test | agregat IDR lalu konversi as-of akhir periode |
| **Regresi billing** | token/invoice/quota tetap IDR, tidak menyentuh `currency.Service` |

---

## 15. Ringkasan Keputusan Backend

| # | Keputusan |
|---|---|
| B-1 | Paket baru `internal/currency`; base code satu konstanta |
| B-2 | Otoritas konversi di server; client tidak kirim rate/base |
| B-3 | Endpoint preferensi terpisah `PUT /me/preferences/currency` |
| B-4 | Endpoint katalog/rate admin: GET untuk admin, POST/PUT untuk Super Admin |
| B-5 | Event SSE `config_changed(domain=currency)` untuk refresh |
| B-6 | Agregat pakai `amount_base`; konversi display sekali per hasil |
| B-7 | AI konteks ikut display currency; biaya/token/quota tetap IDR |
| B-8 | Ekspor konversi + legenda kurs |
| B-9 | Aktivasi mata uang butuh rate valid |
| B-10 | Budget tetap IDR di storage |
