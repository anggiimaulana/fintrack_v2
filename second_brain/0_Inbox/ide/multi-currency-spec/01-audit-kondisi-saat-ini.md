# 01 — Audit Kondisi Saat Ini

Dokumen ini memetakan kondisi nyata repository `fintrack_v2` sebelum perubahan multi-kurs. Tujuannya bukan menilai kualitas kode, tetapi menemukan titik yang harus diubah dan risiko yang mengintai.

---

## 1. Arsitektur Ringkas

| Layer | Teknologi | Catatan relevan |
|---|---|---|
| Frontend | Next.js 16 App Router, React 19, TypeScript, TanStack Query | Format angka dilakukan di banyak komponen client |
| Backend | Go 1.26, Gin, Clean Architecture | Domain → usecase → repository → handler |
| Database | PostgreSQL 17 + pgvector | Nominal transaksi `numeric(15,2)` |
| Cache | Redis + versioned cache 30 detik | GET transaksi/laporan di-cache per user |
| Realtime | SSE + Redis Pub/Sub | Ada event `config_changed` untuk invalidasi UI |
| Auth/RBAC | JWT cookie + middleware role | `user < admin < super_admin` |
| Billing | IDR integer di ledger, invoice, payment | Tidak boleh ikut terkonversi |

Alur nominal saat ini:

```
Input pengguna
  → frontend mengirim angka (float64)
  → backend menyimpan transactions.amount numeric(15,2)
  → semua agregasi SUM(amount)
  → frontend memformat dengan Intl.NumberFormat currency IDR
```

Tidak ada konsep mata uang di domain transaksi maupun profil.

---

## 2. Skema Database Saat Ini

### 2.1 `transactions`

Sumber: `database/migrations/0002_tables.sql:157-184`.

```sql
CREATE TABLE transactions (
    id                  uuid PRIMARY KEY,
    user_id             uuid NOT NULL REFERENCES users(id),
    type                text NOT NULL CHECK (type IN ('income','expense','savings')),
    amount              numeric(15,2) NOT NULL CHECK (amount > 0),
    income_category_id  uuid REFERENCES income_categories(id),
    expense_category_id uuid REFERENCES expense_categories(id),
    savings_category_id uuid REFERENCES savings_categories(id),
    subcategory_id      uuid REFERENCES expense_subcategories(id),
    description         text,
    notes               text,
    tags                text[] NOT NULL DEFAULT '{}',
    transaction_date    date NOT NULL DEFAULT CURRENT_DATE,
    is_recurring        boolean NOT NULL DEFAULT false,
    recurring_interval  text CHECK (recurring_interval IN ('daily','weekly','monthly')),
    source              text NOT NULL DEFAULT 'manual'
                        CHECK (source IN ('manual','ai','voice','receipt')),
    is_deleted          boolean NOT NULL DEFAULT false,
    deleted_at          timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
```

Implikasi:

- Hanya ada **satu nilai** dan **satu mata uang implisit** (IDR).
- Tidak ada cara membedakan transaksi 1.000 JPY dari 1.000 IDR.
- Tidak ada snapshot kurs.
- Backfill nanti harus mengisi kolom baru tanpa mengubah `amount`.

### 2.2 `users`

Sumber: `database/migrations/0002_tables.sql:38-48` dan `0016_user_phone.sql:15`.

```sql
CREATE TABLE users (
    id            uuid PRIMARY KEY,
    email         text NOT NULL UNIQUE,
    password_hash text,
    full_name     text NOT NULL DEFAULT '',
    role          text NOT NULL DEFAULT 'user' CHECK (role IN ('admin','user')),
    avatar_url    text,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone text;
```

Tidak ada `display_currency`, `locale`, atau preferensi angka.

### 2.3 `budgets`

Sumber: `database/migrations/0020_budgets.sql` dan `backend/internal/domain/budget.go:11-20`.

```sql
-- domain
AmountIDR int64 `json:"amount_idr" db:"amount_idr"`
```

Budget adalah plafon per kategori, disimpan sebagai integer IDR. Belum ada konsep mata uang.

### 2.4 Tabel konfigurasi yang sudah ada

| Tabel | Fungsi | Relevansi |
|---|---|---|
| `settings` | key/value JSONB global | Menyimpan knob dinamis, tetapi bukan tempat ideal untuk katalog + riwayat kurs |
| `setting_definitions` | katalog setting yang bisa diedit | Pola dinamis yang bisa ditiru untuk validasi/editor |
| `metric_definitions` | katalog limit | Tidak terkait |
| `feature_definitions` | katalog fitur | Tidak terkait |
| `feature_flags` | rollout flag | Tempat menaruh `multi_currency_enabled` |

`settings` cocok untuk satu nilai global (mis. `base_currency_code`), tetapi **tidak cocok** untuk daftar mata uang + riwayat kurs karena:

- butuh relasi foreign key;
- butuh unique constraint per tanggal;
- butuh audit yang jelas;
- butuh query “kurs terbaru sebelum tanggal”;
- jumlah baris kurs akan tumbuh.

Karena itu desain baru memakai tabel khusus `currencies` dan `currency_rates`, bukan menumpuk JSON di `settings`.

---

## 3. Titik Hardcode di Frontend

### 3.1 Formatter pusat

`frontend/lib/format.ts:4-16`:

```ts
const rupiahFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

export function formatRupiah(value: number | string | null | undefined): string {
  const n = typeof value === "string" ? Number(value) : value;
  if (n == null || Number.isNaN(n)) return "Rp0";
  return rupiahFormatter.format(n);
}
```

Ini adalah satu-satunya helper pusat, tetapi tidak semua komponen memakainya. Banyak komponen membuat `Intl.NumberFormat` sendiri.

### 3.2 Daftar permukaan yang memformat IDR

| Area | File | Bukti |
|---|---|---|
| Komponen uang generik | `frontend/components/shared/Money.tsx` | memanggil `formatRupiah` |
| Kolom transaksi | `frontend/components/transactions/TransactionColumns.tsx:41-49` | `Intl.NumberFormat` currency IDR |
| Baris transaksi | `frontend/components/transactions/TransactionRow.tsx:31-33` | currency IDR |
| Halaman transaksi | `frontend/app/(dashboard)/user/transactions/page.tsx:16,362-413,514` | `formatRupiah` + `RupiahInput` |
| Kartu dashboard | `frontend/components/dashboard/StatCard.tsx:24-27` | currency IDR |
| Chart line | `frontend/components/charts/LineChartView.tsx:9-10` | `formatIDR` lokal |
| Chart bar | `frontend/components/charts/BarChartView.tsx:9-10` | `formatIDR` lokal |
| Chart area | `frontend/components/charts/AreaChartView.tsx:9-10` | `formatIDR` lokal |
| Chart pie | `frontend/components/charts/PieChartView.tsx:13-14` | `formatIDR` lokal |
| Multi-series | `frontend/components/charts/MultiSeriesChart.tsx:11-12` | `formatIDR` lokal |
| Halaman laporan | `frontend/app/(dashboard)/user/reports/page.tsx:54-55` | `formatIDR` lokal |
| Ringkasan laporan | `frontend/components/reports/ReportsSummaryTab.tsx:9-10` | `formatIDR` lokal |
| Top movers | `frontend/components/reports/TopMoversPanel.tsx:7-8` | `formatIDR` lokal |
| Tabel kategori | `frontend/components/reports/CategoryTable.tsx:14-15` | `formatIDR` lokal |
| Calendar heatmap | `frontend/components/reports/CalendarHeatmap.tsx:17-18` | `formatIDR` lokal |
| Budget | `frontend/components/reports/BudgetPanel.tsx:12-13,108` | `formatIDR` + `RupiahInput` |
| AI chat | `frontend/components/ai/ChatWindow.tsx:567-569` | `Intl.NumberFormat` IDR |
| AI inline | `frontend/components/ai/InlineAiAssistant.tsx:292-294` | `Intl.NumberFormat` IDR |
| Receipt scanner | `frontend/components/ai/ReceiptScanner.tsx:175,370,384` | `Rp` + `toLocaleString` |
| Konfirmasi AI | `frontend/components/ai/CrudConfirmCard.tsx:24-27` | currency IDR |
| Chat input | `frontend/components/ai/ChatInput.tsx:45` | teks `Total Rp` |
| Helper receipt | `frontend/lib/receipt.ts:138-145` | `formatRupiahIDR` |
| Input nominal | `frontend/components/shared/RupiahInput.tsx` | prefix `Rp`, hanya digit (tanpa desimal) |

### 3.3 Permukaan yang **sengaja** tetap IDR

| Area | File | Alasan |
|---|---|---|
| Billing/transfer | `frontend/components/billing/TransferInstructions.tsx:117` | Nominal pembayaran |
| Harga paket | `frontend/components/marketing/PricingCards.tsx`, `PricingTeaser.tsx` | Harga publik |
| Admin pembayaran | `frontend/components/super-admin/payments/*` | Nominal pembayaran |
| Admin pricing | `frontend/app/(dashboard)/super-admin/pricing/page.tsx` | Biaya token |
| Admin plan | `frontend/components/super-admin/plans/PlanFormDialog.tsx` | Harga paket |
| Admin token | `frontend/components/super-admin/tokens/TokenLedgerTable.tsx` | Biaya token |

Pemisahan ini penting: refactor tidak boleh menyapu semua `formatRupiah` secara membabi buta.

### 3.4 Profil dan preferensi

`frontend/app/(dashboard)/user/profile/page.tsx:19-25`:

```ts
const SECTIONS = [
  { id: "profile", ... },
  { id: "security", ... },
  { id: "notifications", ... },
  { id: "appearance", title: "Tampilan & Bahasa", ... },
  { id: "privacy", ... },
];
```

`frontend/components/profile/AppearanceSettings.tsx:13-76` hanya berisi:

- pilihan tema `light | dark | system`;
- select bahasa yang **disabled**, nilai tetap `id`.

Belum ada pilihan mata uang. Komponen ini adalah tempat paling natural untuk menambahkan “Mata Uang Pencatatan”, karena sudah menjadi section “Tampilan & Bahasa”.

### 3.5 Tipe sesi pengguna

`frontend/lib/auth-client.ts:9-17`:

```ts
export type AuthUser = {
  id: string;
  email: string;
  full_name: string;
  phone?: string | null;
  role: Role;
  avatar_url?: string | null;
  is_active: boolean;
};
```

Tidak ada `display_currency`. Session provider (`frontend/lib/session-context.tsx:27-32`) mengambil `GET /me` dan menyimpannya di query `["session","me"]`.

---

## 4. Titik Hardcode di Backend

### 4.1 Domain transaksi

`backend/internal/domain/transaction.go:46-66`:

```go
type Transaction struct {
    ID                uuid.UUID          `json:"id" db:"id"`
    UserID            uuid.UUID          `json:"user_id" db:"user_id"`
    Type              TransactionType    `json:"type" db:"type"`
    Amount            float64            `json:"amount" db:"amount"`
    // ...
}
```

`CreateTransactionRequest` (`:132-146`) dan `UpdateTransactionRequest` (`:148-164`) juga hanya memiliki `Amount float64`.

### 4.2 Agregasi transaksi

`backend/internal/repository/postgres/transaction_repo.go` menjumlahkan `amount` di banyak tempat:

| Baris | Query | Dampak |
|---|---|---|
| `299-301` | `SUM(CASE WHEN type='income' THEN amount ...)` | Statistik |
| `325` | `SUM(amount)` monthly expense | Dashboard/SPK |
| `344` | `SUM(t.amount)` category expenses | Laporan |
| `377` | `SUM(t.amount)` category incomes | Laporan |
| `410` | `SUM(t.amount)` category savings | Laporan |
| `503` | `SUM(t.amount)` category time series | Analitik |
| `557-559` | `SUM(amount)` time series | Chart |
| `146-147` (`user_repo.go`) | `SUM(t.amount)` admin user stats | Admin |

Semua harus dialihkan ke `amount_base` agar transaksi lintas mata uang tidak dijumlahkan mentah.

### 4.3 Embedding transaksi

`backend/internal/usecase/transaction_usecase.go:171-177`:

```go
// "{transaction_date} {type} {category_name} {description} Rp{amount}"
return fmt.Sprintf("%s %s %s %s Rp%.0f", ...)
```

Jika transaksi JPY, teks embedding tetap “Rp” dan nominal asli. Pencarian semantik bisa menyesatkan. Perlu format `{currency_code} {amount}` dan/atau nilai base.

### 4.4 AI prompt dan tool

| File | Baris | Hardcode |
|---|---|---|
| `backend/internal/usecase/ai_usecase.go` | `340` | `Pemasukan: Rp%.0f` |
| | `360` | Total tahunan `Rp%.0f` |
| | `377` | Kategori `Rp%.0f` |
| | `389` | Transaksi terbaru `Rp%.0f` |
| | `481` | Instruksi format `Rp 50.000` |
| | `850` | `formatted := fmt.Sprintf("Rp %.0f", amount)` |
| `backend/internal/usecase/agent_tools.go` | `89` | `"Amount in Rupiah"` |
| | `148-170` | Contoh tool call nominal Rupiah |
| | `448-466` | `amount` dibaca sebagai float, tanpa mata uang |

AI harus diberi konteks mata uang aktif dan kurs, lalu tool `create_transaction` harus mengembalikan/menerima nominal dalam mata uang aktif. Backend yang mengonversi ke base.

### 4.5 Ekspor laporan

`backend/internal/export/export.go:278-280`:

```go
func rupiah(v float64) string {
    return "Rp " + strconv.FormatFloat(v, 'f', 0, 64)
}
```

`ReportInput` tidak membawa mata uang atau kurs. CSV/XLSX/PDF selalu IDR.

### 4.6 Budget

`backend/internal/domain/budget.go:11-38`:

```go
AmountIDR int64 `json:"amount_idr" db:"amount_idr"`
```

`budget_usecase.go:63-73` menghitung `remaining` dan `used_pct` dari `amount_idr`. Jika pengguna memakai JPY, input dan tampilan budget perlu konversi, tetapi plafon tetap bisa disimpan sebagai base IDR.

### 4.7 Zona IDR yang harus dilindungi

| Modul | File | Alasan |
|---|---|---|
| Billing domain | `backend/internal/billing/domain.go` | `PriceMonthlyIDR`, `CostIDR` integer Rupiah |
| Pricing | `backend/internal/billing/pricing.go` | Biaya token |
| Invoice/payment | `backend/internal/admin/payments.go`, `payment_repo.go` | Nilai kontrak |
| Metrik admin | `backend/internal/admin/metrics.go` | MRR/revenue |
| Pricing rules | `backend/internal/admin/pricing.go` | Biaya provider |
| Token ledger | `backend/internal/repository/postgres/billing_repo.go` | `cost_idr` |

Jangan menambahkan konversi kurs di modul-modul ini pada rilis multi-kurs.

---

## 5. Konfigurasi Dinamis yang Sudah Ada

### 5.1 `setting_definitions`

Migrasi `0029_config_catalog.sql` memindahkan allowlist setting dari kode ke data. `UpdateSettings` (`backend/internal/admin/settings.go:226-369`) memvalidasi:

1. key ada di `writableSettings` **atau** `setting_definitions` dengan `is_writable=true`;
2. kind (`string|int|bool|enum|secret`) dan batas min/max/enum;
3. secret dienkripsi AES-256-GCM;
4. audit before/after;
5. publish `config_changed` via SSE.

Pola ini bagus untuk **satu nilai**, tetapi tidak cukup untuk katalog + riwayat kurs. Desain baru tetap memakai pola audit + `config_changed`, dengan tabel sendiri.

### 5.2 Cache

`backend/internal/delivery/http/middleware/cache.go:48-99`:

- key = user ID + user version + global version + hash(method:path:query);
- GET sukses di-cache 30 detik;
- invalidasi lewat bump version + Pub/Sub.

Rute yang di-cache antara lain dashboard, transaksi stats, laporan, budget, kategori (`router.go:340-361`). Perubahan kurs harus membump cache agar angka tidak tertinggal 30 detik.

### 5.3 Realtime

`pkg/contracts/events.go:30-45` memiliki `config_changed`. Frontend `useConfigCatalog` menginvalidasi katalog saat event masuk. Pola yang sama bisa dipakai untuk katalog mata uang/kurs.

---

## 6. Gap Analysis

| Kebutuhan | Kondisi saat ini | Gap |
|---|---|---|
| Pilihan mata uang di profil | Tidak ada | Tambah kolom + UI + endpoint |
| Katalog mata uang dinamis | Hardcode `IDR` di formatter | Tabel `currencies` + API |
| Kurs dinamis | Tidak ada | Tabel `currency_rates` + service |
| Riwayat kurs | Tidak ada | Append-only + `effective_at` |
| Nominal asli vs base | Hanya satu `amount` | Tambah `currency_code`, `exchange_rate`, `amount_base`, `rate_at` |
| Agregasi lintas mata uang | `SUM(amount)` | Ganti ke `SUM(amount_base)` |
| Laporan historis stabil | Tidak ada konsep kurs | Rate as of report end |
| Ekspor mengikuti mata uang | Selalu IDR | Pass currency + converter ke export |
| AI sadar mata uang | Prompt/tool hardcode Rp | Inject currency context + konversi tool |
| Billing tetap IDR | Sudah IDR | Jangan diubah; tambah regression test |
| Audit perubahan kurs | Tidak ada | Audit log + rate history |
| Cache invalidation | Ada versioned cache | Bump saat kurs berubah |
| Format desimal | `RupiahInput` hanya digit | Input/format harus baca `decimal_digits` |
| Fallback jika kurs hilang | Tidak ada | Validasi + error yang jelas |

---

## 7. Constraint Teknis yang Tidak Boleh Dilanggar

1. **Migrasi tidak boleh mengedit migrasi lama.** Gunakan nomor baru setelah `0037`.
2. **Expand → migrate → contract.** Jangan `SET NOT NULL` tanpa backfill.
3. **Jangan expose error mentah.** Gunakan `pkg/response`.
4. **Jangan campur cache SSE.** Stream realtime di luar grup cached.
5. **Jangan ubah perilaku billing.** Harga, invoice, payment, token tetap IDR.
6. **Jangan menghapus `amount` lama pada rilis pertama.** Kode lama dan baru harus bisa berjalan bersamaan saat rolling deploy.
7. **Jangan memakai `float64` untuk uang tanpa pembulatan eksplisit.** Saat ini transaksi memakai `float64`; pertahankan tetapi bulatkan ke 2 desimal pada base dan gunakan `numeric` di DB.
8. **Jangan mengandalkan `localStorage` untuk preferensi akun.** Data harus di database.

---

## 8. Kesimpulan Audit

Repository sudah memiliki fondasi yang tepat untuk fitur ini:

- katalog konfigurasi berbasis data (`setting_definitions`);
- audit log;
- versioned cache + Pub/Sub;
- SSE `config_changed`;
- migrasi bernomor dengan aturan expand/contract;
- pemisahan modul billing yang jelas.

Yang belum ada adalah **domain mata uang** dan **nilai base** pada transaksi. Pekerjaan terbesar bukan UI, melainkan:

1. migrasi data + kolom baru;
2. mengganti seluruh agregasi `amount` → `amount_base`;
3. menghapus asumsi IDR dari AI, ekspor, dan embedding;
4. memastikan billing tetap tidak tersentuh.

Detail rancangan ada di dokumen 03–07.
