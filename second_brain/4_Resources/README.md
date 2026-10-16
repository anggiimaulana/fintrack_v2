# 📦 4_Resources — Referensi & Aset Terdistilasi

## Tujuan
Menyimpan referensi yang **tidak berubah selama pengembangan** dan aset terdistilasi: dokumen spesifikasi
yang sudah disetujui (BRD/PRD/Diagram), kredensial (tanpa secret mentah), dan dokumentasi API.

> ⚠️ **JANGAN EDIT** dokumen yang sudah disetujui di sini. Jika ada perubahan, buat dokumen baru dengan versi/tanggal baru.

## Struktur

```
4_Resources/
├── README.md                # File ini
├── Dokumen-Spesifikasi/     # Spesifikasi disetujui (source of truth)
│   ├── README.md
│   ├── BRD/                 # Business Requirements Document
│   ├── PRD/                 # Product Requirements Document
│   │   └── Diagram/         # Use Case, Flowchart, Sequence, Activity, ERD
├── Dokumen-Plan/            # Plan & keputusan migrasi P01 yang disetujui
└── docs/                    # Referensi tambahan (API docs eksternal, catatan, dll)
```

## Aset Referensi

| Aset | Lokasi | Status | Keterangan |
|------|--------|--------|------------|
| **BRD FinTrack v2** | `Dokumen-Spesifikasi/BRD/BRD - FinTrack v2.md` | ✅ Aktif | Kebutuhan bisnis platform berlangganan + AI gateway |
| **PRD FinTrack v2** | `Dokumen-Spesifikasi/PRD/PRD - FinTrack v2.md` | ✅ Aktif | Kebutuhan produk, alur, API, NFR |
| **Diagram** | `Dokumen-Spesifikasi/PRD/Diagram/` | ✅ Aktif | Use Case, Flowchart, Sequence, Activity, ERD |
| Ide & analisis awal | `../5_Archives/ide/2026-10-11_Subscription-AI-Gateway/` | 📦 Arsip | Ringkasan, analisis, kritik, keputusan, peta jalan |
| Brief awal & 7 keputusan terkunci P01 | `../5_Archives/ide/brief_awal.md` | 📦 Arsip | Acuan keputusan migrasi P01 |
| Plan migrasi lengkap P01 | `Dokumen-Plan/` (durable copy) | ✅ Aktif | Path durable: `local://fintrack-v2-migration-plan.md` |
| Dokumentasi DB | `../../database/README.md` | ✅ Aktif | Skema, migrasi, scripts |
| Roadmap & HUD P01 | `../5_Archives/phases/P01_FinTrack_v2/roadmap.md` | 📦 Arsip | Milestones FASE 0–6 |
| Rencana fase P02–P08 | `../2_Projects/` & `../5_Archives/phases/` | ✅ Aktif | Perencanaan & tracking platform berlangganan |
| Standar & SOP | `../3_Areas/` | ✅ Aktif | Keamanan, operasional, billing, AI gateway, dll |

## Aturan
1. **Tidak boleh mengedit** dokumen spesifikasi/plan yang sudah disetujui.
2. **Referensikan** dari `2_Projects/` dan `3_Areas/` menggunakan relative link, jangan duplikasi.
3. Jika ada konflik antar dokumen, prioritas: dokumen lebih spesifik > dokumen umum;
   `database/README.md` untuk hal skema.
4. Jika ada errata/koreksi, catat di `../0_Inbox/` lalu eskalasi — jangan edit dokumen asli.
5. **Jangan simpan secret mentah** di sini — hanya `.env.example` atau referensi nama kredensial.
