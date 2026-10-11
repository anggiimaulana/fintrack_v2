# 📚 3_Areas — Dokumentasi

Area tanggung jawab berkelanjutan untuk **dokumentasi & knowledge base** FinTrack v2.

## Scope

- Standar dokumentasi (PARA di `second_brain/`).
- Decision log / ADR (Architecture Decision Record).
- Pemeliharaan sumber kebenaran (source of truth).
- Konvensi penulisan.

## Dokumen

| Dokumen | Isi |
|---|---|
| `STANDAR_DOKUMENTASI.md` | Struktur & konvensi dokumentasi |
| `DECISION_LOG.md` | Catatan keputusan arsitektur (ADR) |

## Prinsip

1. **Satu sumber kebenaran** per topik; yang lain merujuk, bukan menyalin.
2. **Dokumen disetujui tidak disunting** — buat versi baru bertanggal.
3. **Setiap keputusan teknis signifikan** dicatat (format `K-PXX-NN`).
4. **Audit trail**: setiap aksi teknis dicatat di `../1_Digest/Tugas-Rutin/log_aktivitas.md`.

## Struktur PARA

| Folder | Isi |
|---|---|
| `0_Inbox/` | Ide mentah, brief, analisis awal |
| `1_Digest/` | Log aktivitas & ringkasan harian |
| `2_Projects/` | Proyek aktif (fase, planning, tracking) |
| `3_Areas/` | Standar & SOP berkelanjutan (folder ini) |
| `4_Resources/` | Referensi & spesifikasi (BRD/PRD) |
| `5_Archives/` | Arsip (tidak dipindah tanpa konfirmasi) |

## Referensi

- `../../README.md`, `../../2_Projects/README.md`, `../../4_Resources/README.md`
- `../../CLAUDE.md`
