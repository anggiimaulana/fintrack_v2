# 04 — Arsitektur Backend & Ingestion Pipeline

**Topik:** Desain Ingestion Engine di Go, Adapter Pattern Parser, Two-Stage Normalizer, Deduplikasi, & Rekonsiliasi Transfer  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Arsitektur Ingestion Engine (Go Backend)

Ingestion Engine dibangun di layer backend Go (`backend/internal/ingestion/`) dengan menerapkan prinsip **Adapter Pattern**, **Pipeline Processing**, dan **Strict Transaction Isolation**.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          INGESTION PIPELINE STEPS                           │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                         [File Upload / Stream]
                                    │
                                    ▼
                     ┌─────────────────────────────┐
                     │ 1. StatementParser Factory  │
                     │    (Deteksi Format & Mime)  │
                     └──────────────┬──────────────┘
                                    │ []RawStatementRecord
                                    ▼
                     ┌─────────────────────────────┐
                     │ 2. Deduplication Filter     │
                     │    (SHA-256 Fingerprint)    │
                     └──────────────┬──────────────┘
                                    │ Filtered Uniques
                                    ▼
                     ┌─────────────────────────────┐
                     │ 3. Two-Stage Normalizer     │
                     │    Stage 1: Regex & Pattern │
                     │    Stage 2: AI Gateway Fall │
                     └──────────────┬──────────────┘
                                    │ Enriched Transaction Items
                                    ▼
                     ┌─────────────────────────────┐
                     │ 4. Transfer Pairing Engine  │
                     │    (Detect Internal Trf)    │
                     └──────────────┬──────────────┘
                                    │ Paired Transfer Items
                                    ▼
                     ┌─────────────────────────────┐
                     │ 5. Atomic DB Batch Committer│
                     │    (Lock Account + Ledger)  │
                     └──────────────┬──────────────┘
                                    │
                                    ▼
                     ┌─────────────────────────────┐
                     │ 6. Async Embedding Enqueuer │
                     │    (pgvector RAG Sync)      │
                     └─────────────────────────────┘
```

---

## 2. Adapter Pattern: `StatementParser` Interface

Setiap format mutasi bank/e-wallet memiliki parser terisolasi yang mengimplementasikan satu kontrak Go standar:

```go
package ingestion

import (
	"context"
	"io"
	"time"

	"github.com/fintrack/backend/internal/domain"
)

// RawStatementRecord adalah bentuk data mentah hasil pembacaan baris mutasi.
type RawStatementRecord struct {
	Date            time.Time              `json:"date"`
	RawDescription  string                 `json:"raw_description"`
	Amount          float64                `json:"amount"`
	Type            domain.TransactionType `json:"type"` // income / expense
	ReferenceNumber string                 `json:"reference_number,omitempty"`
	EndingBalance   *float64               `json:"ending_balance,omitempty"`
	ExtraMetadata   map[string]any         `json:"extra_metadata,omitempty"`
}

// StatementParser adalah kontrak adapter parser perbankan/e-wallet.
type StatementParser interface {
	InstitutionCode() string
	SupportedMimeTypes() []string
	Detect(header []byte) bool
	Parse(ctx context.Context, r io.Reader) ([]RawStatementRecord, error)
}
```

### Daftar Adapter yang Diimplementasikan:
1. `bca_parser.go`:
   - Parsing CSV dari KlikBCA: Menangani separator koma/titik dua, format tanggal `DD/MM/YYYY`, membedakan tanda `DB` (keluar/expense) dan `CR` (masuk/income).
   - Parsing PDF dari myBCA: Ekstraksi teks tabel dengan pustaka PDF Go native (`ledongthuc/pdf`), memilah nomor referensi dan deskripsi mutasi berbaris banyak.
2. `mandiri_parser.go`:
   - Parsing PDF e-statement Livin' by Mandiri: Menangkap kolom Tanggal, Uraian Transaksi, Jumlah Mutasi, dan Status D/K.
3. `jago_parser.go`:
   - Parsing CSV & PDF Bank Jago: Pemetaan kolom `Transaction Type`, `Amount`, `Recipient/Sender Name`, dan nama `Pocket`.
4. `seabank_parser.go`:
   - Parsing CSV & PDF SeaBank: Menangani format timestamp transaksi ISO/lokal, nomor transaksi, dan detail bunga/biaya.
5. `gopay_parser.go`:
   - Parsing e-statement GoPay bulanan: Memfilter transaksi Top Up, GoFood, GoSend, Gojek Transport, dan QRIS.
6. `dana_parser.go`:
   - Parsing PDF mutasi DANA: Mengekstrak riwayat transaksi Merchant, Saldo Masuk, dan Biaya Layanan.
7. `shopeepay_parser.go`:
   - Parsing rekap mutasi ShopeePay: Ekstraksi belanja Shopee, transfer bank, dan QRIS.
8. `generic_csv_parser.go`:
   - Parser fallback universal dengan auto-detection header kolom (`date`, `description`, `amount`, `type`/`direction`).

---

## 3. Two-Stage Normalizer Pipeline

Deskripsi mutasi bank di Indonesia sangat kotor dan penuh singkatan mesin:
- Contoh 1: `TRSF E-BANKING DB 2010/WS95011 QRIS SHOPEEPAY JAKARTA`
- Contoh 2: `BI-FAST TRF DR 014 ANGGA TO 542 KANTONG JAGO`
- Contoh 3: `M-BCA 20/10 WSID:88123 TOKO KOPI TUKU JAKARTA SELATAN`

### 3.1 Stage 1: Regex & Pattern Engine (Lokal, Cepat, Gratis)
Tahap pertama dijalankan secara in-memory di Go tanpa pemanggilan AI:
1. **Pembersihan Noise:** Menghapus token perbankan seperti `TRSF`, `E-BANKING`, `DB`, `CR`, `BI-FAST`, `WSID:\d+`, `WS\d+`, `M-BCA`, `IBANK`, `SWITCHING`.
2. **Ekstraksi Rekanan/Merchant:**
   - Deteksi pola QRIS: `QRIS\s+([A-Z0-9\s\.\-]+)`.
   - Deteksi pola Transfer Nama: `TRF (?:TO|DR)\s+(?:[0-9A-Z]+\s+)?([A-Z\s]+)`.
3. **Pencocokan Riwayat Pengguna:**
   - Membandingkan nama merchant bersih dengan riwayat transaksi pengguna sebelumnya di database. Jika pengguna pernah mengkategorikan *"TOKO KOPI TUKU"* sebagai kategori *"Makanan & Minuman"*, kategori tersebut langsung digunakan (Confidence = 1.0).

### 3.2 Stage 2: AI Gateway Fallback (Kuantitas Terkendali, Shared Token Pool)
Jika Stage 1 menghasilkan confidence score $< 0.70$ (pola baru belum pernah ada):
- Ingestion memanggil AI Gateway FinTrack v2 dengan prompt JSON terstruktur:
```json
{
  "raw_text": "TRSF E-BANKING DB 2010/WS95011 TOKO KUE NYONYA MEI",
  "task": "Extract clean merchant name and suggest one primary expense category from user's active categories."
}
```
- Response tervalidasi:
```json
{
  "clean_merchant": "Toko Kue Nyonya Mei",
  "category_name": "Makanan & Minuman",
  "is_transfer": false
}
```
- **Shared Token Pool Invariant:** Pemanggilan ini memotong saldo token pengguna (`token_wallets`) melalui middleware metering yang sudah ada. Jika saldo token pengguna habis, sistem tidak gagal, melainkan beralih ke kategori default *"Uncategorized"* dan meminta pengguna melengkapinya di UI.

---

## 4. Deduplikasi Transaksi (SHA-256 Fingerprint)

Untuk mencegah data ganda saat pengguna mengunggah file mutasi bertanggal sama:
1. Setiap baris dihitung fingerprint-nya:
   ```go
   func GenerateFingerprint(userID, accountID uuid.UUID, date time.Time, amount float64, cleanDesc string) string {
       normalizedDate := date.Format("2006-01-02")
       normalizedAmount := fmt.Sprintf("%.2f", amount)
       clean := strings.ToLower(strings.TrimSpace(cleanDesc))
       
       h := sha256.New()
       h.Write([]byte(userID.String() + "|" + accountID.String() + "|" + normalizedDate + "|" + normalizedAmount + "|" + clean))
       return hex.EncodeToString(h.Sum(nil))
   }
   ```
2. Sebelum insert, sistem melakukan bulk query:
   ```sql
   SELECT fingerprint_hash FROM transactions 
   WHERE user_id = $1 AND fingerprint_hash = ANY($2) AND is_deleted = false;
   ```
3. Seluruh baris yang fingerprint-nya cocok dikeluarkan dari batch insert dan dicatat dalam counter `skipped_duplicates`.
4. Indeks unik `uq_tx_fingerprint` di database bertindak sebagai benteng pertahanan terakhir terhadap race condition.

---

## 5. Smart Transfer Pairing Engine (Rekonsiliasi Transfer Internal)

Menghilangkan bias laporan keuangan akibat pemindahan dana antar-akun pengguna (contoh: Top Up e-wallet dari Bank):

1. **Kriteria Pencocokan:**
   - Pengguna yang sama (`user_id`).
   - Akun berbeda (`account_id_A != account_id_B`).
   - Nominal sama: `amount_base_A == amount_base_B`.
   - Tanggal berdekatan: $| \text{date}_A - \text{date}_B | \le 36\text{ jam}$.
   - Arah transaksi berlawanan: Satu mutasi adalah `expense` (dana keluar dari rekening pengirim) dan satu mutasi adalah `income` (dana masuk ke rekening penerima).
   - Kata kunci transfer pada deskripsi (contoh: "Top Up GoPay" di BCA vs "Top Up BCA" di GoPay).
2. **Aksi Rekonsiliasi:**
   ```sql
   UPDATE transactions SET 
       type = 'transfer', 
       transfer_pair_id = $1, 
       updated_at = now() 
   WHERE id = $2;
   ```
3. **Efek Akuntansi:**
   - Transaksi bertipe `transfer` otomatis dikecualikan dari agregasi `income` dan `expense` pada dashboard utama, namun mutasi saldo pada masing-masing akun tetap tercatat akurat.

---

## 6. Atomic DB Committer & Batch Rollback

Seluruh proses penyimpanan berkas mutasi dibungkus dalam transaksi database atomik:
```go
err := r.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
    // 1. Kunci baris akun untuk konsistensi saldo
    account, err := r.accountRepo.GetForUpdate(txCtx, req.AccountID)
    if err != nil { return err }

    // 2. Simpan record batch impor
    batchID, err := r.batchRepo.Create(txCtx, batchRecord)
    if err != nil { return err }

    // 3. Simpan transaksi unik
    err = r.txRepo.BatchInsert(txCtx, uniqueTransactions)
    if err != nil { return err }

    // 4. Update saldo berjalan akun
    newBalance := account.CurrentBalance + netDelta
    err = r.accountRepo.UpdateBalance(txCtx, req.AccountID, newBalance)
    if err != nil { return err }

    return nil
})
```

Jika terjadi error di tengah proses, seluruh perubahan di-rollback dan database kembali ke keadaan semula tanpa inkonsistensi saldo.
