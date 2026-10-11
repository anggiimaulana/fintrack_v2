# 🧹 3_Areas — Kualitas Kode

Area tanggung jawab berkelanjutan untuk **kualitas kode** FinTrack v2.

## Scope

- Standar penulisan kode (Go & TypeScript).
- Linting, formatting, type-safety.
- Strategi testing (unit, integrasi, kontrak, E2E, load).
- Review guideline & definition of done.
- Batas modul & arsitektur bersih.

## Dokumen

| Dokumen | Isi |
|---|---|
| `STANDAR_KODE.md` | Konvensi Go & TypeScript |
| `TESTING_STRATEGY.md` | Piramida testing & alat |
| `REVIEW_GUIDELINE.md` | Kriteria review & DoD |
| `MODULE_BOUNDARIES.md` | Batas modul & prinsip isolasi |

## Perintah Kanonik

```bash
# Backend
cd backend && go vet ./... && go test ./... && go build ./...

# Frontend
cd frontend && pnpm build && pnpm lint && pnpm test
```

## Prinsip

1. **Setiap klaim arsitektur punya test** yang membuktikannya (ledger, failover, idempotensi).
2. **Modul kecil ber-interface** — dapat dipahami & diuji independen.
3. **Tidak ada `: any`** di TypeScript; tidak ada `context.Background()` di `internal/` Go.
4. **Backend otoritatif** — jangan logika bisnis di frontend.

## Referensi

- Aturan workspace: `../../CLAUDE.md`
- Batas modul fase: `../../2_Projects/P0*/teknis/`
