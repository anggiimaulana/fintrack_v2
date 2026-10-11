# Standar Dokumentasi — FinTrack v2

## 1. Struktur & Penempatan

| Jenis dokumen | Lokasi |
|---|---|
| Ide mentah, brief, analisis awal | `0_Inbox/` |
| Log aktivitas | `1_Digest/Tugas-Rutin/log_aktivitas.md` |
| Perencanaan & tracking proyek | `2_Projects/PXX_*/` |
| Standar & SOP berkelanjutan | `3_Areas/` |
| Spesifikasi disetujui (BRD/PRD/Diagram) | `4_Resources/Dokumen-Spesifikasi/` |
| Arsip | `5_Archives/` |

## 2. Konvensi Penamaan

- Dokumen spesifikasi: `BRD - <Produk>.md`, `PRD - <Produk>.md`, `Diagram - <Jenis> <Produk>.md`.
- Keputusan teknis: `K-PXX-NN` (fase XX, nomor NN).
- Task: `TASK-PXX-NNN`.
- Migrasi: `000N_deskripsi.sql`.
- Dokumen bertanggal: `YYYY-MM-DD_<topik>`.

## 3. Aturan Penulisan

1. **Ringkas & jelas** — hindari basa-basi; gunakan tabel & daftar.
2. **Bahasa konsisten** — Indonesia untuk narasi; istilah teknis boleh Inggris.
3. **Setiap klaim teknis** merujuk file/keputusan (dapat diverifikasi).
4. **Tidak ada placeholder** (`TBD`/`TODO`) di dokumen yang disetujui.
5. **Tidak ada secret** di dokumen.

## 4. Aturan Perubahan

| Situasi | Tindakan |
|---|---|
| Dokumen disetujui perlu berubah | Buat dokumen baru bertanggal; jangan sunting asli |
| Errata/koreksi | Catat di `0_Inbox/` lalu eskalasi |
| Konflik antar dokumen | Prioritas: lebih spesifik > lebih umum; DB → `database/README.md` |
| Keputusan baru | Catat di `3_Areas/dokumentasi/DECISION_LOG.md` |

## 5. Audit Trail

- Setiap aksi teknis signifikan → `1_Digest/Tugas-Rutin/log_aktivitas.md`.
- Perubahan schema → migrasi bernomor + regenerate dump.
- Progress proyek → `2_Projects/PXX_*/tracking/tracking.md`.

## 6. Review

- Dokumen baru/berubah di-review sebelum dianggap final.
- Dokumen yang sudah disetujui bersifat **read-only**.

## Referensi

- `DECISION_LOG.md`
- `../../README.md`
