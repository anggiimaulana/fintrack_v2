# Diagram — Activity FinTrack v2

## 1. Activity: User Memakai Fitur AI sampai Kuota Habis

```mermaid
flowchart TD
    S([Mulai]) --> A[Buka fitur AI]
    A --> B[Sistem cek entitlement]
    B --> C{Fitur diizinkan?}
    C -- Tidak --> C1[Tampilkan tawaran upgrade] --> E([Selesai])
    C -- Ya --> D[Sistem reserve token]
    D --> F{Kuota cukup?}
    F -- Tidak --> F1[Pesan kuota habis + upgrade] --> E
    F -- Ya --> G[Panggil AI Gateway]
    G --> H{Sukses?}
    H -- Tidak --> H1[Failover endpoint]
    H1 --> G
    H -- Ya --> I[Commit token + tampilkan hasil]
    I --> E
```

## 2. Activity: Super Admin Mengubah Plan

```mermaid
flowchart TD
    S([Mulai]) --> A[Buka menu Plan]
    A --> B[Pilih plan]
    B --> C[Ubah centang fitur / limit / harga]
    C --> D[Simpan]
    D --> E[Validasi]
    E --> F{Valid?}
    F -- Tidak --> F1[Tampilkan error] --> C
    F -- Ya --> G[Bump entitlement_version]
    G --> H[Catat audit log]
    H --> I[Publish event realtime]
    I --> J[User terkait melihat perubahan]
    J --> K([Selesai])
```

## 3. Activity: Review Pembayaran Manual

```mermaid
flowchart TD
    S([Mulai]) --> A[User unggah bukti transfer]
    A --> B[Masuk antrean review]
    B --> C[Super admin buka antrean]
    C --> D[Periksa bukti vs invoice]
    D --> E{Valid?}
    E -- Tidak --> F[Tolak + alasan] --> G[Notifikasi user]
    E -- Ya --> H[Approve]
    H --> I[Tandai payment paid]
    I --> J[Aktifkan/perpanjang langganan]
    J --> K[Catat audit + event realtime]
    K --> G
    G --> L([Selesai])
```

## 4. Activity: Downgrade di Akhir Siklus

```mermaid
flowchart TD
    S([Mulai]) --> A[User pilih downgrade]
    A --> B[Set cancel_at_period_end]
    B --> C[User tetap akses sampai akhir siklus]
    C --> D[Siklus berakhir]
    D --> E[Plan baru aktif]
    E --> F{Pemakaian melebihi kuota baru?}
    F -- Ya --> G[Soft-limit: blokir fitur baru]
    F -- Tidak --> H[Normal]
    G --> I[Data user tetap utuh]
    H --> I
    I --> J([Selesai])
```
