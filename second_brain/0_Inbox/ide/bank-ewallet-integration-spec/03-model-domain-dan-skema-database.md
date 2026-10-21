# 03 — Model Domain & Skema Database

**Topik:** Rancangan Skema Relasional PostgreSQL, DDL Migrasi, Invariant Saldo, & Struct Go  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Skema Relasional PostgreSQL

Integrasi ini menambahkan 2 tabel baru dan memperluas tabel `transactions` yang telah ada, serta mendaftarkan metrik baru pada sistem langganan.

```
┌─────────────────────────┐
│          users          │
└────────────┬────────────┘
             │ 1
             │
             │ *
┌────────────▼────────────┐           ┌─────────────────────────┐
│   financial_accounts    │ 1       * │     import_batches      │
│ ─────────────────────── │───────────│ ─────────────────────── │
│ id (PK)                 │           │ id (PK)                 │
│ user_id (FK)            │           │ user_id (FK)            │
│ name                    │           │ account_id (FK)         │
│ institution_code        │           │ file_hash_sha256        │
│ current_balance         │           │ status                  │
└────────────┬────────────┘           └────────────┬────────────┘
             │ 1                                   │ 1
             │                                     │
             │ *                                   │ *
┌────────────▼─────────────────────────────────────▼────────────┐
│                         transactions                          │
│ ───────────────────────────────────────────────────────────── │
│ id (PK)                                                       │
│ user_id (FK)                                                  │
│ account_id (FK -> financial_accounts)                         │
│ import_batch_id (FK -> import_batches)                        │
│ fingerprint_hash (SHA-256 UNIQUE per user)                    │
│ type ('income', 'expense', 'savings', 'transfer')             │
│ clean_merchant (VARCHAR)                                      │
│ transfer_pair_id (FK self-reference)                          │
└───────────────────────────────────────────────────────────────┘
```

---

## 2. DDL Migrasi SQL

File migrasi berikutnya (contoh: `database/migrations/0044_bank_ewallet_accounts.sql`):

```sql
-- Up Migration

-- 1. Enum jenis akun keuangan
DO $$ BEGIN
    CREATE TYPE financial_account_type AS ENUM ('bank', 'ewallet', 'cash', 'investment');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 2. Enum tipe transaksi (menambahkan 'transfer')
ALTER TYPE transaction_type ADD VALUE IF NOT EXISTS 'transfer';

-- 3. Tabel akun keuangan pengguna
CREATE TABLE IF NOT EXISTS financial_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    account_type financial_account_type NOT NULL,
    institution_code VARCHAR(50) NOT NULL,       -- 'bca', 'mandiri', 'jago', 'seabank', 'gopay', 'dana', 'shopeepay', 'custom'
    account_number_mask VARCHAR(30),             -- Format: '****1234' (UU PDP: tidak boleh simpan nomor rekening utuh)
    currency_code VARCHAR(3) NOT NULL DEFAULT 'IDR' REFERENCES currencies(code),
    current_balance NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    initial_balance NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    color_hex VARCHAR(7) DEFAULT '#2563EB',      -- Warna representasi visual di dashboard
    icon_key VARCHAR(50) DEFAULT 'wallet',       -- Identifier icon di frontend
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_financial_account_name UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_fin_accounts_user_active 
    ON financial_accounts(user_id) 
    WHERE is_active = true;

-- 4. Tabel audit batch impor berkas mutasi
CREATE TABLE IF NOT EXISTS import_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES financial_accounts(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    file_hash_sha256 CHAR(64) NOT NULL,          -- Identifikasi file fisik identik
    total_rows INT NOT NULL DEFAULT 0,
    imported_rows INT NOT NULL DEFAULT 0,
    skipped_duplicates INT NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'completed', -- 'processing', 'completed', 'failed', 'rolled_back'
    error_message TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_import_batches_user_account 
    ON import_batches(user_id, account_id, created_at DESC);

-- 5. Penambahan kolom pada tabel transactions
ALTER TABLE transactions 
    ADD COLUMN IF NOT EXISTS account_id UUID REFERENCES financial_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS import_batch_id UUID REFERENCES import_batches(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS fingerprint_hash CHAR(64),
    ADD COLUMN IF NOT EXISTS clean_merchant VARCHAR(150),
    ADD COLUMN IF NOT EXISTS transfer_pair_id UUID REFERENCES transactions(id) ON DELETE SET NULL;

-- Indeks unik fingerprint untuk memastikan idempoten dan bebas duplikasi
CREATE UNIQUE INDEX IF NOT EXISTS uq_tx_fingerprint 
    ON transactions(user_id, fingerprint_hash) 
    WHERE is_deleted = false AND fingerprint_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transactions_account_date 
    ON transactions(account_id, transaction_date DESC) 
    WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_transactions_transfer_pair 
    ON transactions(transfer_pair_id) 
    WHERE transfer_pair_id IS NOT NULL;

-- 6. Trigger update timestamp
CREATE OR REPLACE TRIGGER on_financial_accounts_updated
    BEFORE UPDATE ON financial_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

---

## 3. Invariant Data & Saldo Berjalan (Ledger Consistency)

Untuk mencegah ketidaksinkronan data keuangan (*data drift*), sistem menetapkan aturan ketat:

### 3.1 Rumus Saldo Berjalan (Current Balance)
Untuk setiap `account_id`:
$$\text{CurrentBalance} = \text{InitialBalance} + \sum \text{Income} - \sum \text{Expense} + \sum \text{TransferIn} - \sum \text{TransferOut}$$

- Pada mutasi bertipe `transfer`:
  - Baris sisi pengirim: saldo akun pengirim berkurang senilai `amount_base`.
  - Baris sisi penerima: saldo akun penerima bertambah senilai `amount_base`.
  - **Net Total Pengguna:** Tidak ada perubahan pada total kekayaan bersih (*Net Worth*).
- Pada mutasi bertipe `income`: saldo akun bertambah.
- Pada mutasi bertipe `expense`: saldo akun berkurang.

### 3.2 Lock Concurrency saat Batch Ingestion
Ketika file mutasi diimpor, pembaruan saldo akun wajib menggunakan kunci baris eksplisit:
```sql
-- Dijalankan di dalam transaksi DB sebelum mutasi dimasukkan
SELECT id, current_balance 
FROM financial_accounts 
WHERE id = $1 AND user_id = $2 
FOR UPDATE;
```
Hal ini mencegah *lost update* jika dua file berbeda diunggah pada saat yang bersamaan untuk akun yang sama.

### 3.3 Mekanisme Pembatalan (Batch Rollback Invariant)
Jika pengguna memicu aksi rollback pada `import_batch_id`:
1. Sistem mengambil seluruh transaksi aktif yang terafiliasi dengan `import_batch_id`.
2. Menghitung total pembalikan saldo per akun:
   $$\Delta \text{Balance} = -\sum \text{Income} + \sum \text{Expense} + \sum \text{TransferOut} - \sum \text{TransferIn}$$
3. Mengupdate `financial_accounts.current_balance = current_balance + \Delta \text{Balance}`.
4. Menandai seluruh transaksi tersebut sebagai `is_deleted = true`.
5. Melepaskan kaitan `transfer_pair_id` jika pasangan transfer berada di luar batch tersebut.
6. Menghapus vector embedding terkait di `transaction_embeddings`.
7. Mengubah status batch menjadi `status = 'rolled_back'`.

---

## 4. Representasi Model Domain di Go Backend

Definisi struct Go (`backend/internal/domain/financial_account.go`):

```go
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type FinancialAccountType string

const (
	AccountTypeBank       FinancialAccountType = "bank"
	AccountTypeEwallet    FinancialAccountType = "ewallet"
	AccountTypeCash       FinancialAccountType = "cash"
	AccountTypeInvestment FinancialAccountType = "investment"
)

type FinancialAccount struct {
	ID                uuid.UUID            `json:"id" db:"id"`
	UserID            uuid.UUID            `json:"user_id" db:"user_id"`
	Name              string               `json:"name" db:"name"`
	AccountType       FinancialAccountType `json:"account_type" db:"account_type"`
	InstitutionCode   string               `json:"institution_code" db:"institution_code"`
	AccountNumberMask *string              `json:"account_number_mask,omitempty" db:"account_number_mask"`
	CurrencyCode      string               `json:"currency_code" db:"currency_code"`
	CurrentBalance    float64              `json:"current_balance" db:"current_balance"`
	InitialBalance    float64              `json:"initial_balance" db:"initial_balance"`
	ColorHex          string               `json:"color_hex" db:"color_hex"`
	IconKey           string               `json:"icon_key" db:"icon_key"`
	IsActive          bool                 `json:"is_active" db:"is_active"`
	CreatedAt         time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at" db:"updated_at"`
}

type ImportBatchStatus string

const (
	BatchStatusProcessing ImportBatchStatus = "processing"
	BatchStatusCompleted  ImportBatchStatus = "completed"
	BatchStatusFailed      ImportBatchStatus = "failed"
	BatchStatusRolledBack  ImportBatchStatus = "rolled_back"
)

type ImportBatch struct {
	ID                uuid.UUID         `json:"id" db:"id"`
	UserID            uuid.UUID         `json:"user_id" db:"user_id"`
	AccountID         uuid.UUID         `json:"account_id" db:"account_id"`
	FileName          string            `json:"file_name" db:"file_name"`
	FileSizeBytes     int64             `json:"file_size_bytes" db:"file_size_bytes"`
	FileHashSHA256    string            `json:"file_hash_sha256" db:"file_hash_sha256"`
	TotalRows         int               `json:"total_rows" db:"total_rows"`
	ImportedRows      int               `json:"imported_rows" db:"imported_rows"`
	SkippedDuplicates int               `json:"skipped_duplicates" db:"skipped_duplicates"`
	Status            ImportBatchStatus `json:"status" db:"status"`
	ErrorMessage      *string           `json:"error_message,omitempty" db:"error_message"`
	Metadata          map[string]any    `json:"metadata" db:"metadata"`
	CreatedAt         time.Time         `json:"created_at" db:"created_at"`
}

var (
	ErrAccountNotFound      = errors.New("akun keuangan tidak ditemukan")
	ErrAccountDuplicateName = errors.New("nama akun keuangan sudah digunakan")
	ErrAccountInactive      = errors.New("akun keuangan berstatus nonaktif")
	ErrBatchAlreadyRolledBack = errors.New("batch impor sudah dibatalkan sebelumnya")
)
```
