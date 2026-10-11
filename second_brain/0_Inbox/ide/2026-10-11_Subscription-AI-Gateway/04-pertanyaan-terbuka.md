# 04 — Pertanyaan Terbuka

> ✅ **Sudah dijawab sebagian besar** di `06-jawaban-rekomendasi-pertanyaan-terbuka.md`
> (rekomendasi konkret + alasan + rujukan) dan dikonkretkan di `07-konfigurasi-seed-awal.md`.
> Dokumen ini tetap disimpan sebagai daftar pertanyaan asli.

Pertanyaan berikut **belum diputuskan** dan perlu dijawab sebelum implementasi fase terkait. Tidak
semuanya harus dijawab sekarang; beberapa dapat dijawab saat fase itu dimulai.

## Bisnis & Pricing

1. **Harga & nama plan awal** — berapa harga IDR dan berapa kuota token untuk Free/Pro/Business?
   (Matriks di analisis masih contoh.)
2. **Base price per 1.000 token** — berapa angka dasar sebelum markup? Apakah markup berbeda per plan?
3. **Free plan** — apakah ada (freemium) atau semua berbayar? Apakah ada masa trial (mis. 14 hari)?
4. **Pengali modality** — berapa rasio vision/voice/embedding terhadap chat? (mis. vision 3×, voice 2×?)
5. **Kebijakan overage** — bila user habis kuota, apakah boleh "beli token tambahan" (top-up) atau hanya upgrade?

## Pembayaran

6. **Metode manual** — rekening/e-wallet tujuan transfer? Siapa yang mereview bukti & berapa SLA-nya?
7. **Midtrans environment** — sandbox dulu, kapan produksi? Siapa pemegang akun Midtrans?
8. **Refund** — kebijakan refund (tidak ada / prorata / case-by-case)? Bagaimana teknisnya (manual?)
9. **Grace period** — berapa hari (usul 3)? Berapa kali reminder & kapan?
10. **Pajak** — kapan PPN diaktifkan? Apakah wajib NPWP untuk invoice?

## AI Gateway

11. **Detail 9router** — apakah 9router menyediakan endpoint daftar model/health? Bagaimana format
    `usage` yang dikembalikan (OpenAI-compatible `usage.prompt_tokens`)? Combo apa saja yang tersedia?
12. **Endpoint fallback langsung** — apakah perlu route cadangan ke provider langsung (Gemini/OpenAI)?
    Dengan kunci darurat terpisah?
13. **Model embedding** — dipastikan tetap `text-embedding-004` (768d)? Jika ingin model lain, siap
    migrasi kolom + backfill?
14. **Batas kuota per kunci** — apakah 9router punya kuota per key/combo yang perlu dipantau?

## RBAC & Entitlement

15. **Limit per plan final** — daftar `feature_key` & `limit_key` final (nama & arti)?
16. **Limit harian vs per-siklus** — `max_ai_requests_per_day` dihitung per hari kalender atau per 24 jam?
17. **Perilaku saat melewati limit** — blokir keras, atau izinkan dengan peringatan + catat overage?

## Operasional

18. **Target skala awal** — berapa user aktif yang diantisipasi 6 bulan pertama? (menentukan ukuran Redis/VPS)
19. **Observability** — apakah cukup log + metrik internal, atau perlu Prometheus/Grafana?
20. **Backup & DR** — frekuensi backup billing (data uang wajib paling ketat)? RPO/RTO?
21. **Zona waktu** — konfirmasi TZ `Asia/Jakarta` untuk semua perhitungan siklus & invoice.
22. **Retensi data** — berapa lama menyimpan invoice & audit log? Kebijakan PDP (anonimisasi vs hapus)?

## UI/UX

23. **Posisi menu super admin** — apakah dipisah dari `/admin` (mis. `/super-admin/*`) atau menggantikan?
24. **Onboarding plan** — apakah ada halaman pricing publik? Halaman upgrade in-app?
25. **Notifikasi** — selain in-app (SSE), apakah perlu email (reminder, invoice, pembayaran berhasil)?

## Teknis

26. **Redis** — versi & mode (single / sentinel / cluster)? Deploy lokal via compose dulu?
27. **Feature flag** — cukup tabel `feature_flags` di DB, atau perlu layanan khusus (mis. Unleash)?
28. **Migrasi kunci lama** — apakah kunci per-user lama diimpor ke pool terpusat, atau dibiarkan arsip?

> Catatan: pertanyaan 1–5, 11, 15 perlu dijawab **sebelum** fase terkait dimulai (P03/P04). Sisanya
> dapat menyusul.
