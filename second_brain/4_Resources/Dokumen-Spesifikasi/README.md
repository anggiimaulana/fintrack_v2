# Dokumen-Spesifikasi — FinTrack v2

Referensi spesifikasi produk & bisnis FinTrack v2 (platform berlangganan + AI gateway terpusat).
Dokumen di sini adalah **source of truth** yang sudah disetujui; jangan sunting tanpa keputusan baru
bertanggal.

## Struktur

```
Dokumen-Spesifikasi/
├── README.md                 # File ini
├── BRD/
│   └── BRD - FinTrack v2.md  # Business Requirements Document
└── PRD/
    ├── PRD - FinTrack v2.md  # Product Requirements Document
    └── Diagram/
        ├── Diagram - Use Case FinTrack v2.md
        ├── Diagram - Flowchart Sistem FinTrack v2.md
        ├── Diagram - Sequence FinTrack v2.md
        ├── Diagram - Activity FinTrack v2.md
        └── Diagram - ERD FinTrack v2.md
```

## Ringkasan isi

| Dokumen | Isi |
|---|---|
| **BRD** | Masalah bisnis, tujuan, metrik, ruang lingkup, kebutuhan bisnis (BR-01..25), aturan bisnis, analisis pasar, risiko, kepatuhan |
| **PRD** | Persona, arsitektur, kebutuhan fungsional (FR), alur utama, NFR, model data, kontrak API, standar UI/UX, kriteria penerimaan |
| **Diagram** | Use case, flowchart (metering, pembayaran, siklus, entitlement), sequence (metering, webhook, upgrade, failover), activity, ERD |

## Sumber ide & analisis

- `../../0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/` — ringkasan ide, analisis mendalam, kritik &
  saran best-practice, keputusan terkunci, pertanyaan terbuka, peta jalan.

## Dokumen terkait

- Brief & keputusan migrasi P01: `../../0_Inbox/brief_awal.md`
- Roadmap proyek: `../../2_Projects/`
- Area operasional & billing: `../../3_Areas/`
- Dokumentasi DB: `../../../database/README.md`

## Aturan

1. Dokumen yang sudah disetujui **tidak boleh disunting**; buat versi/tanggal baru bila berubah.
2. Setiap perubahan signifikan dicatat di `../../1_Digest/Tugas-Rutin/log_aktivitas.md`.
3. Jangan menyimpan secret mentah di sini.
