# 📋 1_Digest — Log Aktivitas & Tugas Rutin

## Tujuan
Mencatat semua aktivitas pengembangan **FinTrack v2** secara kronologis dan mengelola checklist tugas rutin yang berulang.

## Struktur

```
1_Digest/
├── README.md                    # File ini
└── Tugas-Rutin/
    ├── log_aktivitas.md         # Audit trail aktivitas development (append-only, terbaru di atas)
    └── README.md                # Template & aturan tugas rutin
```

## Aturan Pengisian

### Log Aktivitas (`Tugas-Rutin/log_aktivitas.md`)
- Diisi **setiap aksi teknis signifikan** oleh developer/PIC yang aktif
- Format: tanggal → ringkasan per fase → catatan/deviasi kecil
- **Append-only / jangan edit entri lama** — koreksi dibuat sebagai entri baru
- **Urutan terbaru di atas** (reverse chronological)

### Tugas Rutin (`Tugas-Rutin/`)
- Berisi checklist recurring (code review, `go vet`/`go test`, `pnpm build`/`lint`/`test`, regenerate dump, backup drill)
- Setiap periode buat file baru: `checklist-YYYY-MM-DD.md`
- Template ada di `Tugas-Rutin/README.md`

## Siapa yang Mengisi
| Role | Tanggung Jawab |
|------|---------------|
| Backend Dev | Log aktivitas backend, migrasi, auth/JWT, SPK queue, RAG |
| Frontend Dev | Log aktivitas frontend, komponen, halaman, audit UI/UX |
| Database | Log perubahan migrasi bernomor, seed, dump/restore |
| DevOps | Log aktivitas Docker Compose, env, deploy |
| PIC/Lead | Review log mingguan, update HUD `2_Projects/P01_FinTrack_v2/roadmap.md` |

## Referensi
- Perintah kanonik: lihat `CLAUDE.md` / `.agents/rules/workspace_rules.md`
- Progress HUD: `../2_Projects/P01_FinTrack_v2/roadmap.md`
