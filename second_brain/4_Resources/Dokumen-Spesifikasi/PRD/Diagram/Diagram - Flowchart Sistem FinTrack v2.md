# Diagram — Flowchart Sistem FinTrack v2

## 1. Alur Pemakaian AI (metering token)

```mermaid
flowchart TD
    A[User kirim request AI] --> B{Entitlement plan izinkan fitur?}
    B -- Tidak --> B1[403 + tawaran upgrade]
    B -- Ya --> C[Reserve token di ledger]
    C --> D{Kuota cukup?}
    D -- Tidak --> D1[402 Payment Required]
    D -- Ya --> E[AI Gateway: pilih endpoint sehat]
    E --> F{Panggilan sukses?}
    F -- Tidak --> G[Circuit breaker + pindah endpoint]
    G --> F
    F -- Semua gagal --> H[503 pesan ramah + release reserve]
    F -- Ya --> I[Ambil usage provider / estimasi]
    I --> J[Commit token + release sisa]
    J --> K[Balas ke user + catat metrik]
```

## 2. Alur Berlangganan & Pembayaran

```mermaid
flowchart TD
    A[User pilih plan] --> B{Gateway aktif?}
    B -- Midtrans --> C[Snap createTransaction]
    C --> D[User bayar di halaman Midtrans]
    D --> E[Webhook Midtrans]
    E --> F{Signature valid?}
    F -- Tidak --> F1[Tolak + audit]
    F -- Ya --> G{Idempotent key sudah ada?}
    G -- Ya --> G1[Abaikan - replay]
    G -- Tidak --> H[Verifikasi via Status API]
    H --> I[Pembayaran paid]
    B -- Manual --> J[Instruksi transfer + unggah bukti]
    J --> K[Antrean review super admin]
    K --> L{Approve?}
    L -- Tidak --> L1[Tolak + notifikasi]
    L -- Ya --> I
    I --> M[Aktifkan/perpanjang subscription]
    M --> N[Reset kuota token + event realtime]
```

## 3. Alur Siklus Langganan (per-hari, hybrid renewal)

```mermaid
flowchart TD
    A[Siklus aktif] --> B{Mendekati akhir?}
    B -- H-3/H-1/H-0 --> C[Kirim reminder]
    C --> D{Metode auto?}
    D -- Ya --> E[Charge otomatis]
    D -- Tidak --> F[User bayar manual]
    E --> G{Sukses?}
    F --> G
    G -- Ya --> H[Perpanjang + reset kuota]
    G -- Tidak --> I[Grace period]
    I --> J{Masih gagal?}
    J -- Ya --> K[Turun ke free - soft-limit]
    J -- Tidak --> H
```

## 4. Alur Perubahan Entitlement (realtime)

```mermaid
flowchart TD
    A[Super admin ubah centang/limit plan] --> B[Simpan + bump entitlement_version]
    B --> C[Cache lama otomatis gugur]
    C --> D[Request user berikutnya pakai aturan baru]
    B --> E[Publish event realtime]
    E --> F[User terkait terima SSE - UI diperbarui]
```
