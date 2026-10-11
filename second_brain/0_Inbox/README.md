# 📥 0_Inbox — Quick Capture Zone

Zona tangkap-cepat untuk FinTrack v2. Taruh apapun di sini dulu, rapikan nanti.

## Aturan
1. **Tangkap cepat** — Taruh apapun di sini: ide mentah, catatan meeting, pertanyaan teknis, screenshot error, errata spec.
2. **Format bebas** — Tidak perlu struktur. File `.md`, `.txt`, gambar, log, apapun.
3. **Proses rutin** — Triage isi folder ini secara berkala (akhir sprint / sesi review):
   - Actionable → pindah ke `2_Projects/P01_FinTrack_v2/` sebagai task
   - Referensi terdistilasi → pindah ke `4_Resources/`
   - Standar/SOP jangka panjang → pindah ke `3_Areas/`
   - Selesai / tidak relevan → pindah ke `5_Archives/` atau hapus
4. **Jangan biarkan menumpuk** — Inbox yang membengkak = tanda triage mandek.

## Struktur
```
0_Inbox/
├── README.md      # File ini
├── ide/           # Ide mentah & eksperimen yang belum jadi task
└── *.md           # Catatan lepas, brief, errata
```

## Konvensi Penamaan
```
YYYY-MM-DD_deskripsi-singkat.md
```
Contoh: `2026-10-15_ide-optimasi-cache-per-user.md`

## Isi Saat Ini
- `brief_awal.md` — brief awal & 7 keputusan terkunci migrasi FinTrack v2 (belum dipindah, masih jadi acuan).
- `ide/2026-10-11_Subscription-AI-Gateway/` — hasil analisis mendalam permintaan platform berlangganan &
  AI gateway terpusat (6 file: ringkasan, analisis, kritik & saran, keputusan, pertanyaan terbuka, peta jalan).
  Acuan untuk BRD/PRD di `../4_Resources/Dokumen-Spesifikasi/` dan fase `../2_Projects/P02..P08/`.

## Status
Folder ini harus **kosong atau hampir kosong** setelah setiap sesi triage. File aktif yang bernilai tinggi segera dipindah ke zona yang tepat.
