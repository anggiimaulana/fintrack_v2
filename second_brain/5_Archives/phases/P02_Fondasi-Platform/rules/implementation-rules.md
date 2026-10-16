# Implementation Rules — P02 Fondasi Platform

Dokumen aturan implementasi ini bersifat **BINDING (MENGIKAT)** bagi setiap engineer, sub-agent, dan script yang bekerja pada codebase FinTrack v2 selama Fase P02.

---

## 1. Aturan Infrastruktur & Redis
1. **Dilarang Menjalankan Command Mematikan di Redis Production**:
   - `KEYS *`, `FLUSHDB`, `FLUSHALL`, `CONFIG SET` dilarang digunakan dalam kode aplikasi Go.
   - Pengecekan atau pencarian key wajib melalui deterministic key generation atau versi.
2. **Koneksi Pool Redis Wajib Dikonfigurasi**:
   - `PoolSize`: Minimal 20, Maksimal 100.
   - `MinIdleConns`: Minimal 5.
   - `DialTimeout`: 5 detik.
   - `ReadTimeout`: 3 detik.
   - `WriteTimeout`: 3 detik.
3. **Setiap Cache Key Wajib Punya TTL**:
   - Dilarang membuat key cache tanpa TTL, kecuali key versi (`cache:u:{user_id}:ver`) yang dikontrol siklus hidupnya.
   - Default TTL untuk HTTP response cache adalah 60 menit dengan jitter random +/- 5 menit untuk mencegah *cache stampede*.
4. **Isolasi Namespace Cache**:
   - Seluruh cache yang berhubungan dengan data user wajib menyertakan user ID dalam key: `cache:u:{user_id}:...`. Dilarang mencampur payload antar-user.

---

## 2. Aturan Modul Audit Log (UU PDP No. 27/2022)
1. **Append-Only Immutability**:
   - Dilarang menyediakan fungsi atau query `UPDATE` atau `DELETE` pada tabel `audit_logs`.
   - Modul `internal/audit` hanya mengekspos method `Log(ctx, event AuditEvent) error`.
2. **Sanitasi Data Wajib (Masking)**:
   - Sebelum payload `before_state` atau `after_state` dikirim ke database, seluruh field sensitif wajib di-masking menjadi `"[REDACTED]"`.
   - Daftar field sensitif: `password`, `token`, `secret`, `api_key`, `refresh_token`, `pin`, `cvv`, `card_number`.
3. **Non-Blocking Operation**:
   - Pemanggilan audit log dari HTTP handler tidak boleh memblokir eksekusi response utama. Wajib disalurkan ke worker queue channel.
   - Kegagalan pengiriman log audit tidak boleh menyebabkan request bisnis user gagal (fail-safe logging dengan panic recovery).

---

## 3. Aturan Feature Flags
1. **Strategi Evaluasi Fail-Closed**:
   - Jika service evaluator mengalami kegagalan (koneksi Redis down, database error), flag baru wajib bernilai `false`.
2. **Penamaan Flag**:
   - Nama feature flag wajib menggunakan format `kebab-case` atau `snake_case` konsisten, diawali domain fungsional:
     Contoh: `ai_gateway_v2`, `subscription_stripe_checkout`, `realtime_sse_v2`.
3. **Idempoten Evaluasi Rollout**:
   - Rollout percentage wajib deterministik berdasarkan hashing integer user ID: `hash(flag_key + user_id) % 100 < percentage`.
   - User yang sama pada hari yang sama harus mendapatkan status flag yang konsisten.

---

## 4. Aturan Arsitektur Data Table & Frontend
1. **URL State sebagai Single Source of Truth**:
   - Komponen `<DataTable>` dilarang menyimpan pagination/filter/search hanya di lokal state React (`useState`) tanpa melakukan cermin ke searchParams URL.
   - Setiap navigasi tabel wajib memperbarui searchParams browser menggunakan router shallow update / replace state agar tidak merusak riwayat back button.
2. **Standar Pagination**:
   - Halaman dimulai dari index 1 (`page=1`).
   - Ukuran default adalah 20 item per halaman (`per_page=20`).
   - Pilihan ukuran halaman terbatas: `10, 20, 50, 100`.
3. **Debounce pada Search Input**:
   - Input teks pencarian global wajib memiliki debounce minimal 300ms sebelum memicu pembaruan URL atau query network TanStack Query.
4. **Visual Loading State**:
   - Pengambilan data tabel wajib menampilkan `<Skeleton>` baris tabel, bukan spinner layar penuh (no full-screen blocker), menjaga context layout tetap stabil.

---

## 5. Aturan Database & Migrasi (Expand/Contract)
1. **Penomoran Migrasi Berurut**:
   - Migrasi P02 menggunakan file `0005_*.sql` dan `0006_*.sql`. Dilarang menimpa file migrasi lama `0001` - `0004`.
2. **Kesesuaian Tipe Data**:
   - Timestamp wajib bertipe `TIMESTAMPTZ` dengan default `CURRENT_TIMESTAMP`.
   - ID entitas audit menggunakan `BIGSERIAL` atau `UUIDv7` berindeks waktu untuk efisiensi write IOPS.
3. **Indeks Wajib**:
   - Setiap kolom yang sering difilter pada tabel audit (`actor_id`, `entity_type`, `created_at`) wajib memiliki B-Tree index terpisah atau composite index yang terukur.
