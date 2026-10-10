# Database — FinTrack v2

PostgreSQL 17 + `pgvector`. Tanpa RLS, tanpa schema `auth.*` Supabase. Otorisasi ada di Go; role aplikasi (`fintrack_app`) least-privilege tanpa DDL.

## Struktur

```
database/
├── migrations/                  # DDL replayable, bernomor (urut)
│   ├── 0001_extensions.sql      # CREATE EXTENSION vector
│   ├── 0002_tables.sql          # 13 tabel + fungsi update_updated_at + trigger
│   ├── 0003_indexes.sql         # index valid (v1 011 tidak replayable)
│   └── 0004_roles.sql           # role fintrack_app (least-privilege)
├── seeds/                       # data demo (BUKAN migrasi)
│   └── 0001_demo.sql            # 4 user + kategori + transaksi + SPK + chat
├── scripts/
│   ├── migrate.sh               # apply migrasi (+ --seed opsional)
│   ├── dump.sh                  # regenerate dump final (commit)
│   ├── restore.sh               # restore dump ke DB target
│   ├── backup.sh                # dump bertimestamp (gitignored)
│   └── verify.sql               # count + sum per tabel
├── dump/                        # dump final di-commit
│   ├── fintrack_v2.dump         # custom format (untuk restore)
│   └── fintrack_v2.sql          # plain (untuk inspeksi)
└── migration_from_supabase/     # migrasi data dari DB lama (Fase 3)
```

## Cara Pakai

### 1. Start DB (compose)

```bash
docker compose up -d db        # port 5435 -> 5432, db "fintrack"
```

### 2. Apply migrasi

```bash
# Dari root repo:
APP_DB_PASSWORD=<password> database/scripts/migrate.sh

# Dengan seed demo:
APP_DB_PASSWORD=<password> database/scripts/migrate.sh --seed
```

Variabel env (default): `PGHOST=localhost PGPORT=5435 PGDATABASE=fintrack PGUSER=postgres PGPASSWORD=postgres APP_DB_PASSWORD=postgres`.

Migrasi idempoten (aman dijalankan ulang).

### 3. Regenerate dump final

```bash
database/scripts/dump.sh       # -> database/dump/fintrack_v2.{dump,sql}
```

Dump di-commit ke git (pilihan user). `.env` dan `database/backups/` **tidak** di-commit.

### 4. Restore / backup

```bash
# Restore ke DB target (harus sudah ada)
database/scripts/restore.sh --db fintrack_restore_test

# Backup bertimestamp (gitignored)
database/scripts/backup.sh
```

### 5. Verifikasi

```bash
psql -h localhost -p 5435 -U postgres -d fintrack -f database/scripts/verify.sql
```

## Database Pendamping untuk Test

Integration test backend memakai database terpisah `fintrack_test` di cluster yang sama (tanpa seed):

```bash
psql -h localhost -p 5435 -U postgres -c "CREATE DATABASE fintrack_test OWNER postgres;"
PGDATABASE=fintrack_test APP_DB_PASSWORD=postgres database/scripts/migrate.sh
```

Lalu jalankan test integrasi:

```bash
cd backend && TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/fintrack_test go test ./internal/repository/postgres/ -run Integration -v
```

## Aturan

1. **Schema berubah = migrasi bernomor baru** (`migrations/000N_*.sql`). Jangan edit migrasi lama yang sudah di-commit.
2. Setelah perubahan schema: jalankan `dump.sh` untuk regenerate dump final.
3. `seeds/` bukan migrasi — hanya data demo; boleh dijalankan berulang (idempoten).
4. `fintrack_app` tidak punya DDL. Semua DDL dijalankan sebagai `postgres` (owner).

## Kredensial Demo (seed)

| Email | Password | Role |
|-------|----------|------|
| `admin@fintrack.id` | `Admin@123456` | admin |
| `budi@example.com` | `User@123456` | user |
| `siti@example.com` | `User@123456` | user |
| `andi@example.com` | `User@123456` | user |
