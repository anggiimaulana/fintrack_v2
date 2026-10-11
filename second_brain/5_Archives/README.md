# 🗄️ 5_Archives — Arsip Proyek & Dokumen Selesai

## Aturan
1. **Hanya pindahkan item yang benar-benar selesai** — semua acceptance criteria / DoD terpenuhi.
2. **Jangan hapus** — arsip adalah catatan sejarah proyek.
3. **⚠️ Butuh konfirmasi eksplisit user** — jangan memindahkan proyek ke sini tanpa persetujuan (lihat `../../.agents/rules/workspace_rules.md`).
4. **Struktur folder mengikuti sumber aslinya:**
   ```
   5_Archives/
   ├── phases/          # Proyek/fase yang sudah complete (dari 2_Projects/)
   ├── sprint-logs/     # Checklist & log lama (dari 1_Digest/Tugas-Rutin/)
   └── misc/            # Item lain-lain dari 0_Inbox yang sudah diproses
   ```
5. **Beri prefix tanggal** pada folder/file yang dipindahkan:
   ```
   2026-10-11_P01-fintrack-v2-migrasi/
   ```
6. **Jangan reference arsip di planning aktif** — jika perlu rujuk, copy ke `4_Resources/`.

## Kapan Mengarsipkan
- Proyek/fase selesai (semua task ✅ di `roadmap.md`).
- Checklist rutin selesai dan sudah di-review.
- Dokumen di `0_Inbox/` sudah diproses dan tidak actionable lagi.

## Referensi
- Proyek aktif: `../2_Projects/P01_FinTrack_v2/`
- Log & checklist: `../1_Digest/Tugas-Rutin/`
