# 05 — RAG Embedding & Analisis Keuangan

**Topik:** Pengayaan Vector Embedding pgvector, Hybrid Retrieval, Use-Case AI Advisor, & Visualisasi Analitik Multi-Akun  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Evolusi Sistem RAG FinTrack v2

Dengan adanya integrasi akun perbankan dan dompet digital, asisten AI FinTrack v2 berkembang dari asisten pencatat pasif menjadi **Personal Financial Advisor & Cash Flow Analyst** yang memahami konteks rekening, transfer internal, dan profil merchant.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       ENRICHED RAG RETRIEVAL PIPELINE                       │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                       [User Prompt / Pertanyaan AI]
         "Berapa total jajan kopi saya di ShopeePay vs GoPay bulan ini?"
                                    │
                                    ▼
                     ┌─────────────────────────────┐
                     │ 1. Intent & Filter Analyzer │
                     │    - Institution: shopeepay,│
                     │      gopay                  │
                     │    - Date: Current Month    │
                     │    - Semantic Query: "kopi" │
                     └──────────────┬──────────────┘
                                    │
                                    ▼
                     ┌─────────────────────────────┐
                     │ 2. Hybrid SQL + Vector Query│
                     │    - SQL Filter: metadata   │
                     │    - pgvector: Cosine Sim   │
                     └──────────────┬──────────────┘
                                    │ Top-K Chunks
                                    ▼
                     ┌─────────────────────────────┐
                     │ 3. LLM Synthesis (AI Gate)  │
                     │    - Rekonsiliasi Angka     │
                     │    - Insight & Rekomendasi  │
                     └─────────────────────────────┘
```

---

## 2. Format Embedding & Metadata Terstruktur

Setiap transaksi yang berasal dari mutasi bank/e-wallet diindeks ke tabel `transaction_embeddings` dengan format representasi teks yang dirancang khusus untuk meminimalkan halusinasi model.

### 2.1 Format Teks Dokumen Embedding
```
[Akun: {account_name} | Institusi: {institution_code} | Tipe: {account_type}]
Transaksi: {type} senilai {currency_code} {amount} (Nilai Dasar: IDR {amount_base})
Tanggal: {transaction_date}
Kategori: {category_name} (Subkategori: {subcategory_name})
Merchant/Pihak Terkait: {clean_merchant}
Deskripsi Asli: {description}
Status Transfer Internal: {is_transfer} (Pasangan Transfer: {transfer_pair_id})
```

### 2.2 Payload Metadata JSONB
Metadata terstruktur disimpan di kolom `metadata` pada `transaction_embeddings` untuk mendukung filter deterministik:
```json
{
  "account_id": "a5e8c112-...",
  "account_name": "BCA Tahapan Utama",
  "institution_code": "bca",
  "account_type": "bank",
  "currency_code": "IDR",
  "amount_base": 125000.00,
  "transaction_type": "expense",
  "clean_merchant": "Kopi Kenangan",
  "is_transfer": false,
  "year_month": "2026-10",
  "transaction_date": "2026-10-18T14:32:00Z"
}
```

---

## 3. Hybrid Retrieval Engine

Pencarian semantik murni (*pure vector search*) sering kali gagal saat pengguna memberikan instruksi matematika atau filter rentang tanggal yang ketat. FinTrack v2 menggunakan strategi **Hybrid Retrieval**:

```sql
-- Contoh Hybrid Query di PostgreSQL dengan pgvector
SELECT 
    t.id,
    t.amount_base,
    t.currency_code,
    t.transaction_date,
    t.clean_merchant,
    te.content,
    1 - (te.embedding <=> $1) AS similarity_score
FROM transaction_embeddings te
JOIN transactions t ON t.id = te.transaction_id
WHERE te.user_id = $2
  AND t.is_deleted = false
  -- Deterministic SQL Metadata Filter:
  AND ($3::uuid IS NULL OR t.account_id = $3)
  AND ($4::varchar IS NULL OR te.metadata->>'institution_code' = $4)
  AND ($5::timestamptz IS NULL OR t.transaction_date >= $5)
  AND ($6::timestamptz IS NULL OR t.transaction_date <= $6)
ORDER BY te.embedding <=> $1 ASC
LIMIT $7;
```

---

## 4. Skenario & Use-Case AI Financial Advisor

### Skenario 1: Analisis Pengeluaran Lintas Dompet Digital
- **Pertanyaan Pengguna:** *"Bandingkan pengeluaran jajan makanan saya di GoFood (GoPay) vs ShopeeFood (ShopeePay) selama 3 bulan terakhir."*
- **Aksi RAG:** Hybrid filter mengambil seluruh transaksi dengan metadata `institution_code IN ('gopay', 'shopeepay')` dan kategori makanan, mengagregasikan nominal dasar IDR, lalu memberikan ringkasan perbandingan beserta proporsinya.

### Skenario 2: Deteksi Kebocoran Kas & Biaya Tersembunyi
- **Pertanyaan Pengguna:** *"Berapa total biaya admin bank, transfer antar-bank, dan biaya top-up yang saya bayar bulan ini?"*
- **Aksi RAG:** Pencarian semantik mencari kata kunci *"biaya transfer"*, *"admin bulanan"*, *"topup fee"*, mengelompokkannya per rekening bank/e-wallet, dan menyarankan cara menghemat (misal: merekomendasikan penggunaan BI-FAST atau SeaBank yang bebas biaya transfer).

### Skenario 3: Audit Saldo & Pelacakan Arus Kas
- **Pertanyaan Pengguna:** *"Kenapa saldo rekening Jago saya tinggal sedikit, ke mana saja uangnya pindah?"*
- **Aksi RAG:** AI mengidentifikasi transaksi transfer keluar terbesar dari Bank Jago ke akun investasi atau e-wallet pengguna, menjelaskan bahwa uang tersebut bukan hilang sebagai konsumsi, melainkan berpindah ke rekening lain.

---

## 5. Visualisasi Analitik Multi-Akun (Frontend Dashboard)

Untuk melengkapi kemampuan AI, dashboard FinTrack v2 menghadirkan 4 komponen visualisasi utama:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     NET WORTH HERO WIDGET (Rp 45.250.000)                   │
├──────────────────────────┬──────────────────────────┬───────────────────────┤
│ Bank BCA: Rp 28.500.000  │ Bank Jago: Rp 12.000.000 │ GoPay: Rp 1.250.000   │
│ SeaBank:  Rp  3.000.000  │ ShopeePay: Rp   500.000  │ Tunai/Cash: Rp 0      │
└──────────────────────────┴──────────────────────────┴───────────────────────┘

┌──────────────────────────────────────┬──────────────────────────────────────┐
│  SANKEY CASH FLOW DIAGRAM            │  TOP MERCHANTS (PENGELUARAN TERBESAR)│
│                                      │                                      │
│  [Gaji Masuk] ──► [BCA]              │  1. Tokopedia      ████████  Rp 3.5M │
│                     ├──► [Jago]      │  2. Superindo      ██████    Rp 2.1M │
│                     ├──► [GoPay]     │  3. Kopi Tuku      ███       Rp 450K │
│                     └──► [Belanja]   │  4. PLN Postpaid   ██        Rp 350K │
└──────────────────────────────────────┴──────────────────────────────────────┘
```

1. **Net Worth Hero Card:** Menampilkan total kekayaan bersih likuid dari seluruh rekening aktif, disertai indikator kenaikan/penurunan bulanan (*MoM growth*).
2. **Breakdown Saldo per Institusi:** Visualisasi donat/progress bar pembagian aset likuid (Berapa % di Bank Digital vs E-Wallet).
3. **Sankey Diagram Arus Kas:** Memetakan perjalanan uang dari sumber pendapatan, transfer internal antar-rekening, hingga keluar ke kategori belanja.
4. **Top Merchants Breakdown:** Bar chart horizontal interaktif yang menampilkan tempat-tempat belanja paling sering dan bernominal tertinggi, dapat difilter per akun.
