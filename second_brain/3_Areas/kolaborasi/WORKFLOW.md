# Workflow & Git — FinTrack v2

## 1. Alur Kerja Fase

```
1. Baca README fase + analisis + rules
2. Ambil task dari planning/tasks.md
3. Implementasi + test
4. Verifikasi (go vet/test/build; pnpm build/lint/test)
5. Update tracking/tracking.md
6. Catat di log_aktivitas.md
7. Commit (atomik)
```

## 2. Git Workflow

### Konvensi Commit

```
feat(<scope>): deskripsi singkat
fix(<scope>): deskripsi singkat
chore(<scope>): deskripsi
docs(<scope>): deskripsi
refactor(<scope>): deskripsi
```

- Scope contoh: `aigateway`, `billing`, `payment`, `rbac`, `datatable`, `admin`, `db`.
- Satu commit = satu perubahan logis.
- Jangan commit `.env`, secret, atau file `tester/`.

### Cabang

| Cabang | Fungsi |
|---|---|
| `main` | Selalu hijau & dapat di-deploy |
| `feat/<topik>` | Fitur dalam pengembangan |
| `fix/<topik>` | Perbaikan |

## 3. Aturan Zona

| Zona | Commit? |
|---|---|
| `frontend/`, `backend/`, `database/` | ✅ |
| `second_brain/` | ✅ |
| `tester/` | ❌ (gitignored) |

## 4. Migrasi

- Schema berubah → migrasi bernomor baru (`database/migrations/000N_*.sql`) → regenerate dump.
- Migrasi wajib **aditif** (expand/contract) untuk zero downtime.

## 5. Ceremonies

| Kegiatan | Frekuensi | Keluaran |
|---|---|---|
| Perencanaan sprint | Awal sprint | `sprint-plan.md`, `tasks.md` |
| Review | Akhir sprint | Verifikasi DoD |
| Retrospektif | Akhir fase | Perbaikan proses |

## 6. Eskalasi

- Blocker teknis → catat di `0_Inbox/` + tanyakan ke pemilik produk.
- Perubahan scope → keputusan baru bertanggal; jangan ubah dokumen disetujui.

## Referensi

- `../../CLAUDE.md`, `CODE_REVIEW.md`
- `../../2_Projects/README.md`
