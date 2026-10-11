# Diagram — Use Case FinTrack v2

```mermaid
graph LR
    Guest((Guest))
    User((User / Tenant))
    SuperAdmin((Super Admin))
    Midtrans[(Midtrans)]
    Router9[(9router / LLM)]

    subgraph FinTrack v2
        UC1[Daftar / Login]
        UC2[Catat Transaksi]
        UC3[Chat AI]
        UC4[Scan Struk / Vision]
        UC5[Voice Input]
        UC6[Lihat Laporan & Chart]
        UC7[Kelola Langganan]
        UC8[Upgrade / Downgrade]
        UC9[Bayar - Midtrans]
        UC10[Bayar - Manual / Bukti]
        UC11[Lihat Kuota Token]
        UC12[Kelola Plan & Entitlement]
        UC13[Kelola User]
        UC14[Review Pembayaran Manual]
        UC15[Kelola AI Gateway]
        UC16[Lihat Audit Log]
        UC17[Atur Gateway & Settings]
    end

    Guest --> UC1
    User --> UC1
    User --> UC2
    User --> UC3
    User --> UC4
    User --> UC5
    User --> UC6
    User --> UC7
    User --> UC8
    User --> UC11
    UC8 --> UC9
    UC8 --> UC10
    UC3 --> Router9
    UC4 --> Router9
    UC5 --> Router9
    UC9 --> Midtrans
    SuperAdmin --> UC12
    SuperAdmin --> UC13
    SuperAdmin --> UC14
    SuperAdmin --> UC15
    SuperAdmin --> UC16
    SuperAdmin --> UC17
    Midtrans -.webhook.-> UC14
```

**Catatan:**
- `User` hanya dapat memakai fitur yang diizinkan oleh **entitlement plan**-nya (dicek backend).
- `SuperAdmin` mengelola platform; tidak ada role lain (keputusan K-02).
- Pembayaran manual & Midtrans bermuara ke aktivasi langganan yang sama.
