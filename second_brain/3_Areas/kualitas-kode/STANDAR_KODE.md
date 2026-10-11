# Standar Kode — FinTrack v2

## 1. Go (Backend)

### Struktur

```
backend/
├── cmd/            # entrypoint (server, backfill)
├── internal/
│   ├── domain/         # entitas & interface repository
│   ├── usecase/        # logika bisnis
│   ├── repository/     # implementasi DB
│   ├── delivery/http/  # handler, middleware, router
│   ├── infrastructure/ # auth, crypto, db, embedding
│   └── aigateway/ billing/ entitlement/ payment/ realtime/ audit/   # [BARU]
└── pkg/            # utilitas bersama (logger, response, validator)
```

### Aturan

- **Context propagation wajib** — tidak ada `context.Background()` di `internal/`.
- **Error wrapping** dengan `%w`; jangan menelan error.
- **Repository = interface di `domain/`**, implementasi di `repository/postgres/`.
- **Uang = integer** (`int64` rupiah); **token = integer**.
- **Idempotensi** untuk semua mutasi uang/token.
- **Tidak ada logika bisnis di handler** — handler hanya binding & delegasi ke usecase.
- Logging structured (zap), tanpa secret.

### Penamaan

- File: `snake_case.go`. Tipe: `PascalCase`. Variabel: `camelCase`.
- Migrasi: `000N_deskripsi.sql` (bernomor, replayable).

## 2. TypeScript / Next.js (Frontend)

- **App Router**; server component default, `"use client"` bila perlu.
- **TanStack Query** untuk server-state; hindari state server di `useState`.
- **Tidak ada `: any`**; gunakan tipe eksplisit.
- Komponen kecil & fokus; file besar = tanda tanggung jawab terlalu banyak.
- Semua tabel pakai `<DataTable>`; semua elemen interaktif `cursor-pointer`.
- `queryKeys` terpusat di `lib/cache-keys.ts`; invalidasi granular (bukan seluruh grup).

## 3. Umum

| Aturan | Alasan |
|---|---|
| Backend otoritatif | Klien tidak dipercaya |
| Data uang integer | Hindari galat pembulatan float |
| Ledger append-only | Audit & anti-race |
| Feature flag untuk fitur baru | Zero downtime & rollback cepat |
| Migrasi aditif | Kompatibel dengan kode berjalan |

## Referensi

- `TESTING_STRATEGY.md`, `MODULE_BOUNDARIES.md`
- `../../CLAUDE.md`
