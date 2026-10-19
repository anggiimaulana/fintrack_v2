# Analisis Risiko, Mitigasi Kegagalan & Zero Downtime Strategy

Dokumen ini memetakan seluruh potensi risiko kegagalan, degradasi performa, kebocoran memori, celah otorisasi, dan strategi mitigasi teknis untuk paritas modul `admin` dan `super_admin` di FinTrack v2.

---

## 1. Matriks Analisis Risiko & Mitigasi

| ID Risiko | Area | Deskripsi Risiko | Tingkat Keparahan | Probabilitas | Solusi Mitigasi Terapan |
|---|---|---|---|---|---|
| **R-01** | Realtime | *Cache Storm* (Banjir query berulang akibat broadcast event SSE). | Tinggi | Sedang | Terapkan **Hybrid Caching**: Gunakan *In-Memory Cache Patch* untuk pembaruan status individual (0 HTTP hit). Batasi *Surgical Invalidation* hanya untuk queryKey halaman aktif ber-parameter sama. |
| **R-02** | Jaringan | Koneksi SSE terputus saat perpindahan jaringan atau browser background. | Sedang | Tinggi | Mekanisme *Auto-Reconnect* dengan backoff eksponensial (1s s/d 16s + jitter). Kirim `last_event_id` saat reconnect untuk memutar ulang event tertinggal dari buffer server. |
| **R-03** | Performa | Lag pengetikan pada input pencarian dan dropdown filter. | Sedang | Tinggi | Terapkan *Debounce 300ms* pada sinkronisasi URL dan request API. Tampilkan *immediate draft state* pada input kontrol agar rendering UI tetap 60fps. |
| **R-04** | Mutasi Data | *Double Click* atau eksekusi ganda pada aksi berbahaya (adjust saldo, suspend). | Tinggi | Sedang | Kunci tombol aksi dengan guard `disabled={isSubmitting || !isValid}` seketika saat klik pertama. Backend mewajibkan header `Idempotency-Key` pada mutasi saldo token. |
| **R-05** | Keamanan | *RBAC Bypass*: Manipulasi URL atau injeksi state klien untuk membuka halaman admin. | Kritis | Rendah | *Defense-in-depth*: Guard frontend hanya berfungsi untuk navigasi UX. Middleware Go Gin (`RequireRole("admin")` dan `RequireRole("super_admin")`) menjadi gerbang mutlak penegakan HTTP 403. |
| **R-06** | Database | Kueri pencarian lambat (*full-table scan*) saat jumlah pengguna membesar. | Tinggi | Rendah | Kueri backend wajib mengombinasikan `LIMIT 20 OFFSET X` dengan indeks B-Tree pada kolom `created_at` dan indeks Trigram / GIN pada `full_name` dan `email`. |
| **R-07** | Antarmuka | *Cumulative Layout Shift (CLS)* saat memuat data tabel atau grafik. | Rendah | Sedang | Gunakan komponen `DataTableSkeleton` dengan tinggi dan jumlah baris tetap yang identik dengan tabel sebenarnya sebelum data selesai dimuat. |
| **R-08** | Browser | Kebocoran memori (*Memory Leak*) akibat event listener SSE tidak dilepas. | Sedang | Rendah | Seluruh langganan pada `realtime-bus.ts` wajib mengembalikan fungsi pembersih (*cleanup function*) pada siklus hidup React `useEffect`. |

---

## 2. Strategi Deployment Zero-Downtime

Untuk memastikan pembaruan antarmuka dan backend dapat dideploy tanpa memutus sesi pengguna yang sedang aktif:

### 2.1. Kompatibilitas Mundur Endpoint API (Backward Compatibility)
- Handler backend `admin_handler.go` yang baru tetap menerima request tanpa parameter pagination (menggunakan default fallback `page=1`, `per_page=20`).
- Frontend lama yang belum di-refresh tetap menerima respon yang valid, sementara frontend baru langsung memanfaatkan metadata pagination.

### 2.2. Skema Event SSE yang Aman (Additive Event Contracts)
- Payload event SSE hanya menambahkan field baru tanpa mengubah nama field yang sudah ada.
- Klien yang menerima event dengan field yang belum dikenali akan mengabaikan field tersebut secara aman tanpa memicu crash parser JSON.

### 2.3. Zero Cache Eviction Outage
- Pembaruan skema state URL (`table-params.ts`) menggunakan nilai default yang aman jika URL parameter yang diterima kosong atau tidak lengkap.

---

## 3. Rencana Pemulihan Bencana (*Rollback & Fallback Plan*)

Jika terjadi anomali kritis pasca-deployment:

1. **Fallback Realtime ke Manual Refresh**:
   - Jika koneksi SSE mengalami gangguan pada level infrastruktur reverse proxy (misal proxy memblokir streaming HTTP), bus realtime menandai status `degraded`.
   - Tombol manual *"Segarkan Data"* (ikon SVG `RotateCw`) muncul di toolbar tabel, memungkinkan admin menyegarkan data tanpa bergantung pada SSE.
2. **Feature Flag Kill Switch**:
   - Fitur-fitur baru (seperti modul shortcut pencarian dinamis) dapat dinonaktifkan secara terpusat melalui flag konfigurasi jika ditemukan anomali browser tertentu.
3. **Rollback Cepat Kontainer**:
   - Image Docker frontend dan backend sebelumnya siap di-deploy kembali dalam hitungan detik jika ditemukan regresi fungsional yang tidak terduga.
