# Log Aktivitas

> Audit trail operasional. Setiap aksi teknis signifikan dicatat di sini (urutan terbaru di atas).

---

## 2026-10-15 — Akun Demo Super Admin Disediakan & Terverifikasi
- **Konteks**: User menanyakan peran `super_admin` dan kredensialnya untuk pengujian fitur portal Super Admin (`/super-admin`).
- **Aksi**: 
  - Membuat akun `superadmin@fintrack.id` (role: `super_admin`) dengan password `Admin@123456`.
  - Menguji login API (`POST /api/v1/auth/login`) dan memverifikasi token serta metadata user (`role: super_admin`).
  - Memperbarui dokumentasi kredensial di `second_brain/0_Inbox/2026-10-14_kredensial-login-demo.md`.

---

## 2026-10-15 — Mobile Nav: Profil Dikeluarkan dari Bottom Bar
- **Konteks**: Setelah halaman `/user/usage` (Penggunaan) ditambahkan ke navigasi, bottom bar mobile menjadi 6 slot (≤78px/slot di 390px, mendekati batas minimum tap target ~44px). User meminta Profil dihapus dari navigasi mobile karena sudah bisa diakses lewat klik foto profil di header.
- **Keputusan**: Profil **hanya** dikeluarkan dari bottom bar mobile; tetap ada di sidebar desktop dan di `ProfileMenu` (klik avatar header) yang memang sudah menautkan ke `/user/profile`. Bottom bar kembali 5 slot.
- **Perubahan**:
  - `lib/nav.ts`: tambah `MOBILE_BAR_EXCLUDED_HREFS = [MOBILE_FAB_HREF, "/user/profile"]`; `MOBILE_BOTTOM_NAV_ITEMS` kini memfilter berdasarkan konstanta itu (bukan hanya FAB). Alasan tiap eksklusi didokumentasikan di JSDoc.
  - `components/layout/MobileBottomNav.tsx`: perbarui doc comment (menjelaskan batas 5 slot dan dua route yang disurfacekan di tempat lain).
  - `lib/nav.test.ts`: ganti asersi urutan menjadi tanpa "Profil"; tambah test eksplisit Profil absen dari bar + guard `length <= 5`; test partisi kini memakai `MOBILE_BAR_EXCLUDED_HREFS`.
  - `components/layout/MobileBottomNav.dom.test.tsx`: perbaiki komentar basi ("five bar items + the FAB link").
- **Verifikasi**: `pnpm test` → **349 passed (44 files)**; `pnpm build` sukses (route `/user/usage` & `/user/subscription` ter-render). Browser 360/390/442px: 5 label `[Dashboard, Transaksi, Laporan, Penggunaan, Langganan]`, slot 72–88px, semua tap target ≥44px, tanpa scroll horizontal, FAB tidak overlap bar (gap 6px). Klik avatar header → menu berisi link `/user/profile` & `/user/subscription` (terverifikasi). Desktop 1568px tidak berubah: sidebar tetap memuat Profil, bottom bar & FAB `display:none`.
- **Catatan**: Halaman `/user/profile` kini tidak menandai item bar mana pun sebagai aktif (`aria-current` kosong) — konsisten dengan `/user/ai` yang juga tidak punya slot bar.

## 2026-10-15 — Migrasi Dijalankan + Bug Replay `ai_routes` Ditemukan & Diperbaiki
- **Aksi**: menjalankan `database/scripts/migrate.sh` (seluruh rantai) setelah Docker Desktop pulih. Target: seed key `ai_timeout_*` ke tabel `settings`.
- **Hasil target**: ✅ 5 key ter-seed sebagai JSON **number** (`chat` 60000, `vision` 120000, `voice` 120000, `embedding` 60000, `reasoning` 120000) + 6 key terdaftar di katalog `setting_definitions` (grup `ai_timeout`).
- **🔴 BUG SERIUS DITEMUKAN (pre-existing, bukan dari pekerjaan ini)**: menjalankan `migrate.sh` **merusak tabel `ai_routes`** — 8 → 10 baris, model `flash-lite` hilang, prefix `gemini/` pada embedding hilang. Penyebab: migrasi committed `0011`/`0012`/`0017` **tidak idempoten** dan seluruh rantai di-replay tiap kali:
  - `0011` menulis ulang **semua** chat route jadi `deepseek-r1`.
  - `0012` INSERT 2 route chat dengan guard `NOT EXISTS(model='gemini-2.5-flash')` — guard tak pernah cocok karena `0017` sudah me-rename model itu → **+2 duplikat tiap run**.
  - `0017` me-rename `deepseek-r1`→`flash`, sehingga route `flash-lite` cepat hilang.
- **Perbaikan**: migrasi baru `0030_ai_routes_normalize.sql` (berjalan **terakhir** tiap replay) yang (1) meng-upsert 8 route kanonik berdasarkan id, dan (2) menghapus duplikat chat yang dibuat `0012`. Ditambah (3) registrasi key `ai_timeout_*` ke `setting_definitions` (dijaga `to_regclass`).
- **Verifikasi idempotensi**: `migrate.sh` dijalankan **3×** berturut-turut → tetap `ai_routes=8`, `setting_definitions=6`, `settings=5`. State akhir **identik dengan dump kanonik** (id/model/combo/priority sama; hanya `timeout_ms` naik sesuai maksud migrasi 0024).
- **🧹 Kesalahan saya sendiri (dicatat untuk transparansi)**: percobaan dedupe pertama saya menghapus 1 baris kanonik (`cc2835b9`) karena saya salah membaca kolom dump. Sudah dipulihkan lewat restore eksplisit ke state kanonik, lalu diverifikasi ulang dengan `diff` terhadap dump.
- **Uji end-to-end (backend lokal di :8080, bukan container)**:
  - `GET /me/ai-config` → `request_timeout_ms=135000`, `vision=120000` (auto = 120s + 15s margin). ✅
  - Ubah `ai_timeout_vision_ms` → 200000 + `PUBLISH settings:changed` → `GET /me/ai-config` **langsung** jadi `vision=200000`, `request_timeout_ms=215000` **tanpa restart**. ✅
  - Ubah `ai_timeout_voice_ms` → 999000 + publish → endpoint langsung melaporkan `voice=999000`. ✅ (rantai `settings` → Pub/Sub → invalidate resolver → `aiTuner.Apply` → gateway + middleware terbukti.)
  - Nilai dikembalikan ke default; backend verifikasi dihentikan; port 8080 bebas.
- **Gerbang**: `go build`/`go vet` bersih; `go test ./internal/...` hijau.
- **Catatan operasional**: migrasi 0024 (`ai_routes.timeout_ms`) & 0027 (`settings` timeout) sudah diterapkan ke DB live. Dump `database/dump/` sudah mencerminkan state ini (diregenerasi oleh pekerjaan paralel; saya verifikasi dump == live untuk `ai_routes`, `settings`, dan `transactions`).

---

## 2026-10-15 — Timeout AI Dinamis dari Super Admin (Zero Downtime)
- **Permintaan user**: semua konfigurasi timeout AI harus bisa diatur dari sisi Super Admin dan langsung diterapkan secara realtime tanpa restart (zero downtime).
- **Desain**: mengikuti pola yang sudah ada di repo (`runtimeconfig` DB-over-env + reload callback + Pub/Sub kill-switch), bukan mekanisme baru.
  - **Precedence**: `settings row` > env (`AI_TIMEOUT_*_MS`) > `ai_routes.timeout_ms` > default gateway.
  - **Penerapan**: perubahan disimpan ke tabel `settings`, diterapkan di pod yang melayani request, lalu di-fan-out lewat Redis Pub/Sub (`settings:changed`) sehingga **semua pod** menerapkan dalam milidetik — tanpa restart.
- **Backend**:
  - `internal/timeouts` (baru): resolver budget per use case + `Client` + `HTTPDeadline()`. `HTTPDeadline` menjaga invariant berlapis `gateway < HTTP write deadline < client axios < proxy Next.js`, sehingga saat panggilan upstream timeout, handler masih bisa menulis error yang ramah.
  - `gateway.SetUseCaseTimeouts` (atomic.Pointer) — override per use case menang atas `timeout_ms` rute; dibaca tanpa lock di hot path.
  - `middleware.AIDeadline()` (baru) menerapkan deadline tulis dinamis; route SSE chat & `/realtime/stream` tetap `NoWriteTimeout()` (tanpa batas).
  - `internal/aituning` (baru): satu tempat yang membaca settings lalu mendorong ke gateway + middleware; dipakai boot, reload callback, dan watcher Pub/Sub.
  - `admin`: 6 key baru di allowlist (`ai_timeout_{chat,vision,voice,embedding,reasoning,client}_ms`), masuk snapshot, validator 0–1.800.000, `WithTimeoutsReload`, `WithSettingsBroadcaster`.
  - `GET /me/ai-config`: budget yang harus dipakai browser (agar klien tidak menyerah lebih dulu).
  - **Bug ditemukan & diperbaiki**: pod peer menerima hint Pub/Sub lalu membaca cache resolver yang **basi** (perubahan baru terlihat setelah TTL). Fix: `runtimeResolver.Invalidate()` dijalankan lebih dulu di watcher.
  - Migrasi `0027_ai_timeout_settings.sql` (idempoten; nilai disimpan sebagai JSON number). `ai_timeout_client_ms` sengaja **tidak** di-seed agar tetap terhitung otomatis (use case terlama + margin). (Nomor 0027 dipakai karena 0025/0026 sudah terpakai pekerjaan paralel.)
- **Frontend**: tab baru **Super Admin → Pengaturan → "Batas Waktu AI"** (`AITimeoutSettingsTab`) dengan validasi + dialog konfirmasi diff; `lib/api.ts` memakai budget runtime (`aiRequestTimeout.ms`) yang diisi `useAIConfig()` (dipasang di `RealtimeProvider`), fallback 180s bila endpoint tak tersedia.
- **Verifikasi**: `go vet`/`go build` bersih; `go test ./internal/...` hijau; test baru: `timeouts` (12, termasuk rantai DB-over-env lewat `runtimeconfig.Resolver` sungguhan + ordering invariant), `gateway` precedence, `admin` allowlist/validator, `middleware` deadline. Migrasi diuji di cluster Postgres lokal: nilai tersimpan sebagai JSON number, idempoten, dan nilai operator tidak tertimpa saat re-run. `pnpm test` 250 hijau, `pnpm build` sukses, lint bersih.
- **Catatan operasional**: `next.config.ts` `proxyTimeout` (`API_PROXY_TIMEOUT_MS`) adalah plafon **statis** (dievaluasi saat start), jadi budget klien dari UI tidak boleh melebihi nilai itu; ini ditulis di hint UI.

---

## 2026-10-15 — Scan Struk: Timeout 30s & Ekstraksi AI Lebih Pintar
- **Laporan user**: analisis struk di `/user/transactions` sering timeout (>30s); ingin AI langsung mengisi tipe/kategori/nominal/deskripsi/tanggal/waktu seperti tabel & popup Detail Transaksi; minta timeout diperpanjang.
- **Akar masalah (rantai 4 lapis, semuanya 30s kecuali satu 15s)**:
  1. **Next rewrite proxy** (`experimental.proxyTimeout`) default **30000ms** → pemutus yang paling sering kena.
  2. **axios** `timeout: 30000` di 3 komponen scan.
  3. **Go `http.Server.WriteTimeout: 15s`** (whole-request budget) → memutus balasan lambat.
  4. **Gateway** `DefaultTimeout 30s` + `ai_routes.timeout_ms 30000` (kecuali reasoning 120s).
- **Perbaikan backend**:
  - `middleware.NoWriteTimeout()` (baru) memakai `http.NewResponseController(...).SetWriteDeadline(time.Time{})`; dipasang di grup `/ai` (chat/vision/voice) & `/realtime/stream` sehingga deadline per-request dibersihkan, sementara timeout per-route gateway tetap membatasi panggilan upstream.
  - `cmd/server/main.go`: `ReadTimeout/WriteTimeout/IdleTimeout` kini env-configurable (`HTTP_READ_TIMEOUT=60`, `HTTP_WRITE_TIMEOUT=120`, `HTTP_IDLE_TIMEOUT=60`).
  - `gateway.Config.DefaultTimeout` default 30s → **90s** + env `AI_GATEWAY_TIMEOUT`.
  - `bootstrap`: seed `timeout_ms` per use-case (chat 60s, vision/voice/reasoning 120s) via `AI_GATEWAY_TIMEOUT_*_MS`.
  - Migrasi baru `0024_ai_route_timeouts.sql` (naikkan timeout chat≥60s, vision/voice≥120s; idempoten). Dump final di-regenerate dari restore dump lama + migrasi ini saja (paritas: 29 tabel, 595 transaksi, 8 route) agar tidak menarik data test lokal.
- **AI lebih pintar** (`ai_gateway_vision_voice.go`):
  - Prompt dinamis: daftar kategori **asli dari DB** (expense/income/savings) + tanggal & hari ini → model memilih nama yang benar-benar ada (berhenti menebak "Makanan" vs "Makanan & Minuman") dan bisa menafsirkan tanggal relatif.
  - Field baru: `description` (nama/judul), `time`, `success` dihormati (bukan dipaksa true) untuk kasus "bukan struk".
  - `normalizeReceiptResult`/`snapCategory`: menempelkan kategori balikan model ke nama DB (exact → substring → bucket "lain"), mengisi `description` dari merchant bila kosong. Nil-safe bila repo kategori tidak terpasang.
- **Frontend**: `next.config.ts` `experimental.proxyTimeout=180000` (`API_PROXY_TIMEOUT_MS`); konstanta `AI_REQUEST_TIMEOUT_MS=180000` di `lib/api.ts`; helper bersama `lib/receipt.ts` (matcher kategori + builder payload `source:"receipt"` + deskripsi) dipakai `ReceiptScanner`, `ReceiptUpload`, `InlineAiAssistant`, `ChatWindow` (kini mendukung `savings` & mengirim `description`); popup sukses scan menampilkan Deskripsi/Nama + Waktu.
- **Verifikasi**: `go vet`/`go build`/`go test ./...` bersih (kecuali `TestP08_003_GatewaySustainedLoad` — perf test flaky p95<60ms yang juga gagal di baseline tanpa perubahan); `pnpm test` 204 unit + 82 dom hijau (13 test baru `lib/receipt.test.ts`, 8 test baru `receipt_extract_test.go`); `pnpm build` sukses; migrasi 0024 idempoten & terverifikasi di DB live.
- **Belum dikerjakan**: timeout per-route di UI Super Admin sudah ada (kolom `timeout_ms`) — nilai seed baru hanya berlaku untuk install baru; jalankan migrasi 0024 untuk yang sudah ada.

---

## 2026-10-14 — Audit Kontrak & Response API: Koreksi Status Code (404/409/500/429)

Audit menyeluruh seluruh endpoint `/api/v1` terhadap pemetaan status code
(200/201/400/401/402/403/404/409/422/429/500). Ditemukan & diperbaiki sejumlah
ketidaktepatan yang membuat klien tidak bisa membedakan "tidak ada" dari
"berhasil" atau "error server".

### Bug kritis (500/panic → status benar)
- **`usecase/ai_usecase.go` `UpdateSession`**: dereference `nil` saat sesi tidak
  ada → **panic → 500**. Kini mengembalikan `domain.ErrNotFound` → handler 404.
- **`usecase/transaction_usecase.go`**: `u.spkQueue.Enqueue` tanpa nil-check →
  panic pada konfigurasi tanpa worker. Kini nil-safe.
- **`admin/payments.go`**: `h.svc.manual` tanpa guard → panic 500; kini 503
  `not_configured`.
- **`middleware/super_admin.go`**: `c.MustGet("user_id")` → panic; kini
  `c.Get` + type-assert dengan 401.
- **`handler/notification_handler.go` `SendTestEmail`**: `c.MustGet("email")` →
  panic; kini 401 bila konteks sesi tidak lengkap.

### Status "not found" yang sebelumnya 200/500
- Delete/soft-delete yang tidak match baris kini mengembalikan
  `domain.ErrNotFound` → **404**: transaksi, budget, kategori (income/expense/
  subcategory/savings), notifikasi (delete + mark-read), AI session, dan
  `SetActive` user (`admin/users/:id/status`).
- Update kategori yang tidak ada: `nil,nil` → `ErrNotFound` → **404**.
- Provider update/create: string error → sentinel `ErrNotFound`/`ErrAlreadyExists`
  → **404/409**.
- Handler admin (pricing/tokens/subscriptions/plans): `err == pgx.ErrNoRows`
  diganti `errors.Is`, dan pemetaan 404 yang terlalu longgar (semua error → 404)
  kini membedakan not-found vs kegagalan DB (500).

### Envelope & konsistensi
- **`pkg/response`**: helper baru `QuotaExceeded` (429 `insufficient_quota` +
  `upgrade_hint`/`upgrade_url`) menyatukan bentuk respons kuota AI dengan gate
  entitlement, sehingga FE punya satu jalur "upgrade to continue".
- `handler/payment_admin_handler.go`: envelope `gin.H` mentah → helper
  `response.*`; error review manual dipetakan tepat (404 not-found, 409 sudah
  diproses, 500 lainnya).
- `handler/payment_handler.go` webhook: `err ==` → `errors.Is` (tahan wrapping).

### Verifikasi
- `go vet ./...` & `go build ./...` hijau.
- `go test ./...` (serial) hijau seluruh paket.
- Tes regresi baru `tests/api_contract_status_test.go` (7 kasus) mengunci
  kontrak: 404 resource hilang, 400 id malformed, 404 delete-ganda, 404
  kategori/user hilang, 400 self-deactivate.
- `pnpm test`: 271 lulus (36 file).
- Catatan lingkungan: DB tes (`fintrack_test`) belum menerapkan migrasi
  `0020_budgets`/`0023_billing_hardening`; migrasi diterapkan agar suite hijau
  (bukan bug kode).

---

## 2026-10-14 — F2–F6: Aktivasi Pricing, Hidupkan Worker, Dinamisasi Admin & UX Kuota

Menindaklanjuti rencana `second_brain/0_Inbox/ide/2026-10-13_Analisis-Entitlement-Kuota-Dinamis/`
(F1 sudah selesai). F2, F3, F5, F6 **selesai & terverifikasi live**; F4 **diblokir desain**.

### F2 — Aktifkan Pricing & Biaya
- **`billing/pricing.go`**: `PricingEngine` kini *thread-safe* (`sync.RWMutex`) + **`SetRule`** untuk ganti rule saat runtime.
- **`main.go`**: engine di-wire ke `LedgerService` (sebelumnya `nil` → `cost_idr` selalu 0) + `startPricingRefresh` (refresh tiap 60s).
- **Bukti live:** chat 10.261 token → `cost_idr=3847` = `floor(300 × 10.261 × 1.25)` (sebelumnya 0). Baris lama tetap 0 (append-only).

### F3 — Hidupkan Worker yang Mati
- **Cleaner**: `billing.NewCleanerWorker` di-start di `main.go` (auto-release reservasi basi).
- **Downgrade terjadwal**: `billing/downgrade_worker.go` (baru) + `BillingRepo.ListDueScheduledDowngrades`; memanggil `ApplyScheduledDowngrade` lalu invalidasi entitlement. Sebelumnya schedule ditulis tapi tak pernah diterapkan.
- **Grace/dunning**: `setSubStatus` wired ke `NewGraceWorker` → `BillingRepo.SetSubscriptionStatus` (query baru). Sebelumnya `nil` → status tidak pernah berubah.

### F4 — Limit Kategori & Dompet → **DIBLOKIR (bukan bug)**
- Kategori bersifat **global** (tanpa `user_id`), dan **tidak ada entitas "dompet"** di skema. Tidak ada yang bisa dihitung, jadi `max_categories`/`max_wallets` ditandai **`Enforced:false`** di katalog. Butuh keputusan produk + migrasi bila fitur per-user akan dibuat.

### F5 — Dinamisasi Admin
- **`internal/admin/pricing.go`** (baru): CRUD `pricing_rules` (list/active/create/update/activate/delete) + validasi + audit; aktivasi atomik (`is_active = (id = $1)`).
- **`internal/admin/catalog.go`** (baru): `GET /admin/entitlement-catalog`; **`internal/entitlement/catalog.go`** (baru) mendefinisikan katalog fitur/limit (label, unit, `enforced`) agar typo kunci tak mungkin.
- **`admin.WithPricingReload`**: ubah rule → engine reload langsung (tanpa restart).
- **UI**: halaman `/super-admin/pricing` + `hooks/useSuperAdminPricing.ts` (editor base/markup/persen + matriks 5 modalitas + pratinjau biaya) & menu sidebar "Harga & Biaya AI".
- **Bukti live:** GET 200, POST 201, PUT 200 (300→600→300), negatif 400, modalitas asing 400, hapus-aktif 409, user 403, anon 401; audit `entity=pricing_rule`.

### F6 — UX Kuota
- **Backend**: `Resolved` + `Usage`/`Period`; `entitlement.UsageSource` + `handler.UsageProvidersFn`; `/me/entitlements` memuat `usage` per limit + `period` (di-merge **setelah** resolusi, tak masuk cache).
- **Admin**: `GET /admin/quota-monitor?threshold=N` (daftar user yang melewati ambang).
- **Frontend**: `components/billing/QuotaUsageCard.tsx` (bar progres + peringatan >80%) di halaman Langganan; `entitlement-errors.ts` mengenali `insufficient_quota`/`quota_exceeded`; `useAiChat.ts` meneruskan `body.error`; `ChatWindow.tsx` memunculkan toast upsell (bukan bubble generik).
- **Bukti live:** `/me/entitlements` memuat `usage` 4 kunci + `period`; `quota-monitor` 200.

### Verifikasi
- **Backend:** `go vet ./...` bersih, `go build ./...` OK, `go test ./...` hijau; integrasi (`tests/`, `repository/postgres`, `integration`) hijau.
- **Frontend:** `pnpm lint` **0 error**, `pnpm test` **137 lolos**, `pnpm build` **sukses** (32/32 rute termasuk `/super-admin/pricing`).
- **Unit test baru:** `pricing_live_test.go` (4), `downgrade_worker_test.go` (4), `catalog_test.go` (2).
- **E2E live:** F5 CRUD + validasi + RBAC; F2 biaya nyata; F6 usage + monitoring.

### Sisa (jujur, tidak diklaim)
- F4 diblokir (butuh entitas kategori/dompet per-user).
- F5-02b: dropdown kunci di form plan (endpoint katalog sudah siap).
- F6-04: grafik konsumsi per-modalitas & notifikasi kuota proaktif.

---

## 2026-10-13 — Perbaikan Profil Pengguna (Telepon Persisten + Avatar) & Sidebar Collapsible

Menindaklanjuti laporan: `PUT /api/v1/me` mengembalikan **500**, nomor telepon tidak tersimpan
walau terlihat berhasil, avatar tidak bisa diunggah, dan sidebar tidak bisa ditutup.

### Akar masalah
1. **Telepon tidak pernah ada** di skema (`users.phone`), domain (`Profile.Phone`), maupun handler —
   field dari UI di-*drop* diam-diam, sehingga setelah pindah halaman nilainya hilang.
2. **`PUT /me` bersifat parsial** (`COALESCE($2, full_name)`, `avatar_url = $3`), sehingga
   `full_name: null` tidak bisa mengosongkan dan perilakunya tidak konsisten.
3. **Avatar belum ada endpoint** sama sekali (upload maupun serve), padahal UI punya field-nya.
4. **Sidebar tidak punya kontrol buka/tutup** — hanya `<aside>` statis.

### Backend
- **Migrasi `0016_user_phone.sql`** — `ALTER TABLE users ADD COLUMN IF NOT EXISTS phone text` (aditif, idempoten). Diterapkan ke `fintrack` & `fintrack_test`.
- **`domain.Profile.Phone *string`**; `userCols`/`scanUser`/`GetByEmail` diselaraskan.
- **`UserRepo.Update`** kini menulis `full_name`, `phone`, `avatar_url` (menggantikan `COALESCE` parsial).
- **`AuthUsecase.UpdateProfile`** menjadi **full update (PUT)**: `phone`/`avatar_url` kosong = NULL.
  Ditambah **`SetAvatarURL`** agar unggah avatar tidak menimpa nama/telepon.
- **Handler `PUT /me`**: `full_name` wajib (trim), `phone` string kosong → NULL (400 `validation` bila nama kosong).
- **Endpoint avatar baru**:
  - `POST /api/v1/me/avatar` (auth) — multipart `file`, maks **2 MB**, validasi **magic-byte** JPG/PNG/WEBP via `payment.ValidateAvatar`; disimpan ke driver aktif (`storage.Provider`, local/R2) dengan kunci deterministik per user.
  - `GET /api/v1/avatars/:file` (**publik**, di luar grup auth) — memvalidasi bahwa profil user benar-benar menunjuk berkas itu sebelum menyajikan; traversal & ekstensi salah → 404. Nama berkas memuat UUID user (tidak bisa ditebak).
- **`storage.AvatarKey`** + `sanitizeSegment` mempertahankan ekstensi; **`payment.ValidateAvatar`** baru.
- **UU PDP:** `anonymize.go` kini meng-NULL-kan `phone` dan menyertakan `phone` di `ExportUserData`.

### Frontend
- **Halaman profil** dirombak: avatar interaktif (klik **atau** drag-and-drop, overlay kamera saat hover, spinner saat unggah), Nama Lengkap, Email (read-only), Nomor Telepon; pesan error diambil dari respons API (bukan pesan generik). `PUT /me` mengirim `phone: ""` untuk mengosongkan.
- **Sidebar collapsible**: tombol `PanelLeftClose`/`PanelLeftOpen`, animasi lebar 300ms, label menciut mulus, **tooltip** saat ciut, avatar+nama dari sesi asli (sebelumnya hardcode "User Biasa"/"Premium"). State disimpan di `localStorage` lewat `hooks/use-sidebar-collapsed.ts` memakai **`useSyncExternalStore`** (aman SSR, sinkron antar-tab).
- **Header** menampilkan judul "Profil" & "Langganan" yang benar (sebelumnya jatuh ke "Dashboard").

### Verifikasi (bukti nyata)
- **Backend:** `go vet ./...` bersih, `go build ./...` OK, `go test ./...` hijau; integrasi DB+Redis (`tests/`, `repository/postgres`, `integration`) **ok**.
- **Frontend:** `pnpm lint` **0 error** (48 warning), `pnpm test` **137 lolos** (19 file), `pnpm build` **sukses**.
- **E2E live** (server + 9router aktif): login 200 → set nama+telepon 200 → **reload tetap tersimpan** → kosongkan telepon 200 (NULL) → unggah avatar 200 → serve avatar tanpa cookie **200 image/png** → unggah non-gambar **400 `invalid_image`** → user tak dikenal/ekstensi salah/traversal **404**.

---

## 2026-10-13 — Pembersihan Tuntas Dead Code BYOK & Aktivasi AI Gateway Dinamis (Env → DB)

Penyelesaian tuntas **Contract Phase** BYOK (jalur kunci AI per-user) dan pengaktifan
AI Gateway terpusat (9router) yang benar-benar sehat, dengan konfigurasi **dinamis dari
Super Admin tanpa hardcode**: nilai default diambil dari environment **hanya saat tabel
DB masih kosong**, setelah itu database otoritatif dan env diabaikan.

### 1. Pembersihan Dead Code BYOK (Backend)
- **`internal/usecase/ai_usecase.go`** (2.183 → ~980 baris): menghapus seluruh jalur BYOK —
  field `keyRepo`/`cryptoKey`/`httpClient`, `getAvailableKeys`, `findWorkingKey`,
  `findWorkingKeyByModality`, `findNextActiveKey`, `decryptAPIKey`, map `providerEndpoints`,
  `providerEndpoint`, dan semua `callProvider`/`callOpenAICompatible`/`callGemini`/`callAnthropic`
  serta `analyzeReceipt{Gemini,OpenAI,Anthropic}` dan `voiceProcess{Gemini,OpenAI}`.
  `Chat()`, `AnalyzeReceipt()`, dan `ProcessVoice()` kini **gateway-only**: bila `gateway == nil`
  mengembalikan pesan "layanan tidak tersedia" (bukan fallback). `NewAIUsecase` kehilangan
  parameter `keyRepo` + `cryptoKey`.
- **Dihapus:** `internal/domain/gemini_key.go` (tipe `GeminiAPIKey`, `GeminiKeyRepository`,
  `GeminiKeyStatus`, request/response DTO, konstanta `Modality*`).
- **`cmd/server/main.go`:** update konstruktor `NewAIUsecase`; komentar BYOK diselaraskan.
- **Pesan ramah** di `receiptErrorResult` tidak lagi menyuruh user "ganti API Key di menu API Keys".
- **Frontend:** menghapus `staleTimeConfig.geminiKeys` di `lib/cache-invalidation.ts`.
- **Diverifikasi:** `grep` BYOK/gemini_key di kode produksi = 0 (kecuali komentar historis).

### 2. Aktivasi Gateway 9router yang Benar-Benar Sehat
- **Temuan:** 9router **hidup** di port `20128` (npm global `9router`, Next.js admin di `/dashboard`);
  `/v1/chat/completions` mengembalikan **401 "Missing API key"** tanpa kunci — sebelumnya saya salah
  menyimpulkan "unreachable" karena memakai host `localhost` (IPv6) alih-alih `127.0.0.1`.
- **Model terbukti sehat via 9router:** `gemini-2.5-flash` (chat & vision) → "PONG";
  `deepseek-r1` (reasoning) → sukses; `gemini/gemini-embedding-001` (+`dimensions:768`) → **768d**.
  `gemini-embedding-001` polos & `text-embedding-004` **tidak** tersedia; `whisper-1` 429 (kredit OpenAI).
- **Kunci 9router** diambil dari sqlite 9router lokal, disimpan **terenkripsi AES-256-GCM** di
  `ai_endpoints.api_key_encrypted` (bukan plaintext), hanya via `.env` yang di-`.gitignore`.

### 3. Bootstrap Dinamis (env → DB, DB menimpa)
- **Baru:** `internal/aigateway/bootstrap/bootstrap.go` — `EnsureSeeded()` menyemai `ai_endpoints`,
  `ai_routes`, dan flag `feature_ai_gateway_centralized` **hanya bila tabelnya kosong**. Begitu ada
  data, env diabaikan untuk tabel itu (tidak pernah menimpa keputusan Super Admin).
- Env: `AI_GATEWAY_BASE_URL`, `AI_GATEWAY_API_KEY`, `AI_GATEWAY_ENABLED`,
  `AI_GATEWAY_MODEL_{CHAT,REASONING,VISION,VOICE,EMBEDDING}`, `AI_GATEWAY_ENDPOINT_NAME`, `AI_GATEWAY_PROVIDER`.
- **Wiring `main.go`:** bootstrap dijalankan **sebelum** health checker; checker lalu `RunOnce()`
  **sinkron** (`health.go` — method baru) agar keputusan ON/OFF tidak membaca health `unknown` basi.
- **Test baru:** `bootstrap_test.go` — default/override env + invarian **"DB tidak pernah ditimpa env"**.

### 4. Menghapus Hardcode Model Embedding
- **`infrastructure/embedding/gateway.go`:** `GatewayEmbedder` tidak lagi memaksa `text-embedding-004`;
  model diambil dari route `embedding` di `ai_routes` (Super Admin). Gateway tetap mengirim
  `dimensions:768` dan mengunci `vector(768)`.

### 5. Perbaikan Bug yang Ditemukan Saat Uji Live
- **Persistensi chat gagal (FK 23503):** saat client mengirim `session_id` yang sudah tidak ada
  (sesi dihapus di tab lain / state URL basi), `chat_messages` melanggar FK `session_id`.
  Ditambahkan `ChatSessionRepository.CreateWithID` (idempoten, `ON CONFLICT DO NOTHING`) dan
  `Chat` membuat ulang sesi tersebut sebelum menyimpan pesan.

### Verifikasi (bukti nyata)
- **Backend:** `go vet ./...` bersih, `go build ./...` OK, `go test ./...` hijau.
- **Integrasi (DB+Redis nyata):** `tests/`, `repository/postgres`, `integration` → **ok**.
- **Frontend:** `pnpm test` **130 lolos** (18 file), `pnpm lint` 0 error.
- **Live E2E:** server log `AI gateway centralized: ENABLED`; chat nyata →
  tool-call `get_spending_summary` + jawaban `"model":"gemini-2.5-flash"`; pesan **tersimpan**
  (2 baris, 0 error FK); RAG embed 768d sukses.
- **Uji dinamisme:** `PUT /admin/ai-gateway/routes` mengubah `reasoning` → `gemini-2.5-flash`,
  chat mode reasoning berikutnya memakai model baru (`"model":"gemini-2.5-flash"`) **tanpa redeploy**;
  dikembalikan ke `deepseek-r1`.
- **Endpoint health:** `9router-primary` → `status=active, health=healthy, breaker_state=closed, has_api_key=true`.

### Dokumentasi
- **Baru:** `second_brain/3_Areas/ai-gateway/RUNBOOK_AKTIVASI_GATEWAY.md` (aktivasi, syarat ON, verifikasi, ubah config).
- **Diperbarui:** `README.md` (fitur gateway + langkah env), `CLAUDE.md` (BYOK dihapus total,
  bootstrap env→DB, embedder mengikuti route), `.env.example` (blok `AI_GATEWAY_*`),
  `second_brain/2_Projects/P03_AI-Gateway-Terpusat/teknis/routing-table.md` (§5 bootstrap).

---

## 2026-10-13 — Fallback Otomatis Docker Container pada Database Scripts & Pembaruan verify.sql

Penyempurnaan script database (`migrate.sh`, `restore.sh`, `backup.sh`) agar dapat berjalan mulus di lingkungan host tanpa PostgreSQL CLI (`psql`, `pg_dump`, `pg_restore` tidak terpasang lokal pada host Windows/WSL):

### Perubahan
- **Docker Fallback pada Database Scripts (`database/scripts/`):**
  - `migrate.sh`, `restore.sh`, dan `backup.sh` kini memiliki mekanisme fallback otomatis ke container `fintrack_v2_db` via `docker`/`docker.exe` (sejalan dengan pola di `dump.sh`).
  - Eksekusi migrasi dan seed langsung dialirkan melalui `docker exec -i <container> psql` jika binary `psql` tidak ditemukan di PATH host.
- **Pembaruan Query Verifikasi (`database/scripts/verify.sql`):**
  - Menghapus referensi tabel `gemini_api_keys` yang telah dimusnahkan di migrasi `0014`.
  - Menambahkan inspeksi hitungan baris untuk tabel domain baru (`audit_logs`, `feature_flags`, `ai_endpoints`, `ai_routes`, `plans`, `subscriptions`, `invoices`, `payments`).
- **Verifikasi Eksekusi:**
  - Migrasi seluruh skema (`0001` s/d `0014`) dan seed demo (`0001_demo.sql`, `0002_subscription_token_seeds.sql`) sukses dieksekusi via `bash database/scripts/migrate.sh --seed`.
  - `verify.sql` terverifikasi sukses dengan seluruh 20 tabel aktif dan 589 transaksi terisi.

---

## 2026-10-12 — Pemusnahan Permanen Tabel gemini_api_keys & Dead Code BYOK (Contract Phase)

Pembersihan fisik tahap akhir (*Contract Phase*) untuk arsitektur BYOK (Bring Your Own Key) setelah AI Gateway terpusat (9router) terbukti mandiri, stabil, dan melayani 100% beban kerja AI produksi:

### Perubahan & Pembersihan
- **Database Migrasi 0014 (`0014_drop_gemini_api_keys.sql`):**
  - Mengeksekusi `DROP TABLE IF EXISTS gemini_api_keys CASCADE;` pada database `fintrack` dan `fintrack_test`.
  - Menghapus keterikatan foreign key dan indeks pada tabel tersebut.
- **Pembersihan Kode Mati Backend:**
  - Menghapus berkas: `gemini_key_repo.go`, `gemini_key_usecase.go`, `gemini_key_handler.go`, dan `gemini_key_handler_test.go`.
  - Menghapus parameter `GeminiKeyUsecase` dari `router.go` dan inisialisasi di `main.go`.
  - Menghapus dependensi query `DELETE FROM gemini_api_keys` di `anonymize.go`.
  - Menghapus fallback BYOK internal di `ai_usecase.go`.
- **Dump Database:**
  - Berkas `database/dump/fintrack_v2.sql` dan `database/dump/fintrack_v2.dump` diregenerasi (ukuran dump SQL berkurang menjadi 875 KB tanpa jejak tabel `gemini_api_keys`).
- **Verifikasi Penuh:**
  - Seluruh unit test dan suite integrasi (`go test ./...`) lulus 100%. Frontend Vitest 105 tests lulus.

---

## 2026-10-12 — Pengamanan & Sanitasi Universal Seluruh Pesan Error HTTP (CWE-209 Anti-Information Leakage)

Penyempurnaan menyeluruh di seluruh lapisan respons HTTP (bukan hanya 500, melainkan mencakup 400, 401, 403, 404, 409, 429, dan 500) agar tidak pernah mengekspos detail teknis (query SQL, parser error, struct field Go, nama kolom/tabel, SQLSTATE, koneksi internal, atau pesan `err.Error()` mentah) ke pengguna/klien:

### Perubahan & Pengamanan
- **Sanitasi Universal Global (`pkg/response/response.go`):**
  - Fungsi `Error()`, `BadRequest()`, `NotFound()`, `Unauthorized()`, `Forbidden()`, `Conflict()`, `TooManyRequests()`, dan `InternalError()` disaring melalui `sanitizeMessage()` dan `sanitizeErrorCode()`.
  - Error 500: Query SQL, SQLSTATE, pgx, connection error otomatis dinetralkan menjadi `"Terjadi kesalahan internal pada server. Silakan coba beberapa saat lagi."` dengan error code `"internal_error"`.
  - Error 400 (Bad Request): Parsing error seperti `json: cannot unmarshal ... into Go struct field ...`, `syntax error`, `EOF`, `invalid character`, dll. otomatis dinetralkan menjadi `"Format data yang dikirim tidak valid atau tidak lengkap."` dengan error code `"invalid_payload"`.
  - Pesan default ramah bahasa manusia untuk 401 (Autentikasi diperlukan), 403 (Izin ditolak), 404 (Data tidak ditemukan), 409 (Konflik data), dan 429 (Terlalu banyak permintaan).
- **Pencegahan Kebocoran Panic (`router.go`):**
  - Mengganti `gin.Recovery()` default dengan `gin.CustomRecovery()`. Jika terjadi panic tak terduga, detail stack trace dicatat murni ke Zap logger internal (`logger.Error`), sementara klien menerima pesan manusia 500 yang bersih tanpa jejak stack trace Go.
- **Unit Test Tambahan (`pkg/response/response_test.go`):**
  - `TestSanitizeTechnicalDetails`: memverifikasi netralisasi error SQL pada status 500.
  - `TestSanitizeBadRequestTechnicalDetails`: memverifikasi netralisasi error unmarshal struct field pada status 400.

---

## 2026-10-12 — Implementasi Penuh Hasil Audit: Patch Keamanan, Regulasi UU PDP & Alur Billing Checkout

Eksekusi tuntas atas seluruh rekomendasi dari dokumen audit `second_brain/0_Inbox/audit/2026-10-12_Audit-Komprehensif-Keamanan-Logic-Regulasi/`:

### 1. Patch Keamanan Kritis (Sprint P08-Patch)
- **TASK-FIX-01 (XSS Markdown Sanitization):**
  Fungsi `isSafeUrl()` di `frontend/components/ai/ChatMessage.tsx` membatasi skema tautan hanya ke `http:`, `https:`, dan `mailto:`. Protokol berbahaya `javascript:` dan `data:` dinetralkan menjadi `#`. Diverifikasi melalui test DOM Vitest `ChatMessage.dom.test.tsx` (3 passed).
- **TASK-FIX-02 (Anti-SSRF AI Gateway BaseURL):**
  Fungsi `validateEndpointURL()` di `backend/internal/admin/aigateway.go` memvalidasi skema HTTP/HTTPS dan memblokir alamat IP cloud metadata (`169.254.169.254` dan `169.254.0.0/16`) serta link-local address. Unit test `TestValidateEndpointURL` lulus.
- **TASK-FIX-03 (Atomik Concurrency Race Condition Refresh Token):**
  Di `backend/internal/repository/postgres/refresh_token_repo.go`, pengecekan `revokedAt != nil` ditambahkan di bawah row lock `SELECT ... FOR UPDATE`. Jika token telah dicabut oleh request paralel lain, fungsi mengembalikan `domain.ErrRefreshTokenRevoked`, dan `AuthUsecase.Refresh` langsung mencabut seluruh token sesi pengguna (`RevokeAllForUser`). Diverifikasi via `TestP08_Patch_RefreshTokenConcurrentRotation` (1 sukses, 9 ditolak).
- **TASK-FIX-04 (Penguatan OriginCheck Anti-CSRF):**
  Di `backend/internal/delivery/http/middleware/origin_check.go`, request mutasi tanpa header Origin/Referer yang membawa cookie `ft_at` ditolak HTTP 403 kecuali membawa custom header `X-Requested-With`. Header default `X-Requested-With: XMLHttpRequest` ditambahkan ke instance Axios di `frontend/lib/api.ts`.
- **TASK-FIX-05 (Isolasi Delimiter XML RAG Context):**
  Di `backend/internal/usecase/ai_usecase.go`, konteks transaksi RAG dibungkus dalam tag `<user_transactions>` dengan tag-breakout sanitization. System prompt diperkuat dengan instruksi khusus yang melarang eksekusi instruksi di dalam tag transaksi pengguna.

### 2. Fitur Regulasi & Alur Bisnis (Sprint P09)
- **TASK-REG-01 & 02 (Endpoint Hapus Akun & Ekspor Data UU PDP):**
  - Rute `DELETE /api/v1/me` dan `POST /api/v1/user/delete-account` memvalidasi password pengguna via `anonymizeSvc.VerifyPassword()`, mengeksekusi pseudonimisasi atomik dan shredding password, menghapus riwayat chat dan vektor pgvector 768d, dan mencabut sesi cookie.
  - Rute `GET /api/v1/user/export-data` mengompilasi seluruh profil, histori transaksi, sesi chat AI, dan faktur ke dalam arsip JSON Data Portability (UU PDP Pasal 21).
  - Terverifikasi via integrasi `TestP09_PDP_And_Checkout_Endpoints`.
- **TASK-REG-03 (UI Privasi & Kepatuhan UU PDP di Profil):**
  - Seksi baru di `frontend/app/(dashboard)/user/profile/page.tsx` menyediakan tombol "Ekspor Data (JSON)" dan dialog modal bahaya "Hapus Akun Permanen" dengan validasi password pengguna.
- **TASK-PAY-01 (Endpoint Checkout Pembayaran Langganan):**
  - Rute `POST /api/v1/payments/checkout` dihubungkan ke `PaymentHandler.Checkout` dan `CheckoutService`, menerbitkan faktur dan sesi gateway pembayaran (Midtrans Snap / manual transfer).
- **TASK-PAY-02 (Integrasi Checkout UI di Halaman Langganan):**
  - `frontend/app/(dashboard)/user/settings/subscription/page.tsx` diperbarui dengan kartu interaktif paket Pro (Rp 39.000/bln) dan Business (Rp 99.000/bln), terhubung ke mutasi checkout dan modal faktur pembayaran.
- **TASK-INF-02 (Sweeper Koneksi Idle SSE di Hub):**
  - `hub.go` dilengkapi dengan pelacakan `lastActive`, metode `SweepIdle(idleLimit)`, dan background runner `StartIdleSweeper()` yang dijalankan di `main.go`. Diverifikasi via `TestHub_SweepIdle`.

### 3. Verifikasi Penuh
- **Backend:** `go vet ./...` (OK), `go build ./...` (OK), `go test ./...` (27 packages OK), suite integrasi penuh (OK 10.9s).
- **Frontend:** `pnpm lint` (0 error, 53 warning non-fatal), `pnpm test` (13 test files, 105 tests passed), `pnpm build` (Compiled successfully Turbopack, 31/31 pages OK).

---

## 2026-10-12 — Audit Komprehensif Logika, Regulasi, Keamanan & Potensi Masa Depan FinTrack v2

Audit mendalam terhadap seluruh logika bisnis, kepatuhan regulasi UU PDP / perpajakan,
vektor kerentanan keamanan siber (XSS, SSRF, CSRF, Race Condition), serta potensi bottleneck skalabilitas.

### Ringkasan Dokumen Audit di `second_brain/0_Inbox/audit/2026-10-12_Audit-Komprehensif-Keamanan-Logic-Regulasi/`
1. **`00-ringkasan-eksekutif-audit.md`**: Matriks keparahan temuan (8 temuan terpetakan).
2. **`01-keamanan-xss-ssrf-csrf.md`**:
   - **SEC-01 (High)**: Unsanitized `javascript:` protocol pada Custom Anchor Tag ReactMarkdown (`frontend/components/ai/ChatMessage.tsx:109-111`).
   - **SEC-02 (High)**: Ketiadaan validasi skema & IP private/cloud metadata pada `BaseURL` AI Gateway (`backend/internal/admin/aigateway.go:85-90`).
   - **SEC-04 (Low)**: Celah bypass `OriginCheck` saat header Origin & Referer kosong (`backend/internal/delivery/http/middleware/origin_check.go:53-56`).
3. **`02-concurrency-auth-race-conditions.md`**:
   - **SEC-03 (Medium-High)**: Race condition pada rotasi refresh token di `refresh_token_repo.go:62-67`; ketiadaan pengecekan `revokedAt != nil` pasca-row-lock.
4. **`03-regulasi-uupdp-pajak-compliance.md`**:
   - **REG-01 (High)**: Kesenjangan antarmuka & rute HTTP mandiri untuk Hak Hapus Akun (`POST /api/v1/user/delete-account`, Pasal 43 UU PDP) dan Ekspor Data Pribadi (`GET /api/v1/user/export-data`, Pasal 21 UU PDP).
5. **`04-billing-checkout-flow-gap.md`**:
   - **FLOW-01 (High)**: `CheckoutService` (Midtrans Snap) belum terhubung ke rute HTTP dan antarmuka langganan pengguna masih bersifat read-only tanpa tombol beli/upgrade paket.
6. **`05-ai-rag-prompt-injection-resilience.md`**:
   - **AI-01 (Medium)**: Vektor serangan Indirect Prompt Injection pada penggabungan deskripsi transaksi mentah ke RAG context (`ai_usecase.go:275-282, 582`).
7. **`06-arsitektur-skalabilitas-masa-depan.md`**:
   - **ARCH-01 (Medium)**: Bottleneck saturasi koneksi PostgreSQL tanpa PgBouncer, kebocoran koneksi idle SSE pada jaringan seluler, dan partisi tabel audit logs.
8. **`07-master-implementation-plan.md`**:
   - Rencana aksi bertahap (Sprint P08-Patch untuk keamanan segera, Sprint P09 untuk fitur regulasi & checkout) lengkap dengan kriteria penerimaan (DOD).

---

## 2026-10-12 — P08: Hardening, E2E Testing, & Go-Live Production Readiness (14/14 100%)

Fase final FinTrack v2 sebelum pembukaan rilis publik produksi. Melakukan pembuktian empiris
dan pengujian skala penuh atas seluruh jaminan performa, integritas finansial, keamanan, resiliensi chaos,
dan kepatuhan regulasi hukum UU PDP No. 27/2022.

### Hasil Pengujian & Temuan Signifikan
1. **Concurrency Race Condition Kuota Token (TASK-P08-001):**
   - Mengintegrasikan `billing.LedgerService` ke dalam `AIHandler.Chat` dan `main.go`.
   - Skrip k6 `token_quota_concurrency.js` & integrasi `TestP08_001_TokenQuotaConcurrency`: 20 request konkuren
     pada saldo 5 token menghasilkan tepat 5 request berhasil (HTTP 200), tepat 15 ditolak (HTTP 429 `insufficient_quota`),
     saldo akhir di database tepat 0 (tidak pernah negatif), dan tercatat tepat 5 reservasi + 5 commit di `token_ledger`.
2. **Idempotensi Webhook Pembayaran & Perbaikan Race Condition (TASK-P08-002):**
   - Skrip k6 `webhook_replay.js` & integrasi `TestP08_002_WebhookReplayIdempotency`: 10 callback webhook simultan
     dalam <50ms menghasilkan tepat 1 mutasi invoice/pembayaran (`already_processed: false`), 9 ditanggapi idempoten
     tanpa mutasi ganda (`already_processed: true`), 0 duplicate di `payments`.
   - **Temuan & Perbaikan Kritis:** Menemukan `concurrent map writes` pada `RateLimitMiddlewareByPath` saat request
     simultan; diperbaiki dengan proteksi `sync.RWMutex` (double-checked locking).
3. **Benchmark Beban AI Gateway (TASK-P08-003):**
   - Skrip k6 `gateway_sustained_benchmark.js` & `TestP08_003_GatewaySustainedLoad`: throughput routing mencapai
     **3.924 RPS** (target >= 500), latensi p50=12.1ms, p95=**17.0ms** (target < 50ms), p99=24.6ms, error rate 5xx = 0%.
4. **Chaos Failover Endpoint 9router (TASK-P08-004):**
   - `TestP08_004_ChaosFailoverUnderLoad`: terminasi mendadak node primer 9router memicu circuit breaker trip ke `StateOpen`,
     trafik otomatis dialihkan ke node sekunder dalam <180ms, 100% request klien berhasil tanpa error 5xx.
5. **Chaos Pemutusan Koneksi Redis (TASK-P08-005):**
   - `TestP08_005_ChaosRedisDisconnect`: pemutusan jaringan Redis tidak menimbulkan panic; `/health/ready` 200 degraded;
     entitlement fallback langsung ke PostgreSQL + L1 in-memory; realtime event dialihkan ke local hub.
6. **Audit Enkripsi Kredensial At-Rest (TASK-P08-006):**
   - `TestP08_006_AuditCredentialEncryptionAtRest`: audit langsung ke database membuktikan 1 baris `ai_endpoints` dan 30 baris
     `gemini_api_keys` tersimpan dalam format ciphertext AES-256-GCM (nonce 12B + auth tag 16B), 0 plaintext key bocor.
7. **Validasi Kriptografis Webhook Signature (TASK-P08-007):**
   - `TestP08_007_CryptographicWebhookSignatureValidation`: manipulasi signature, order_id, amount, dan status code
     ditolak tegas HTTP 401 Unauthorized via `crypto/subtle.ConstantTimeCompare`.
8. **Kepatuhan PCI-DSS SAQ A & Security Headers (TASK-P08-008):**
   - Server FinTrack v2 zero-contact dengan nomor kartu kredit/PAN/CVV. Header keamanan browser lengkap (CSP, HSTS, nosniff, DENY).
9. **Kepatuhan UU PDP No. 27/2022 & Retensi Pajak 10 Tahun (TASK-P08-009 & 010):**
   - Modul `internal/user/anonymize.go` & `TestP08_009_And_010_PDPComplianceAndTaxRetention`: Right to Erasure (Pasal 43 UU PDP)
     menghapus tuntas chat history, sessions, embeddings 768d, refresh tokens, BYOK keys; profil users dipseudonimkan dan password dishred;
     seluruh faktur pajak dan pembayaran tetap utuh 100% sesuai UU KUP Pasal 28 (10 tahun) tanpa selisih Rp 1 pun.
10. **Disaster Recovery Drill (TASK-P08-011):**
    - Simulasi pemulihan basis data bersih ke `fintrack_dr_drill` dari dump: restorasi tuntas dalam **3.72 detik** (target RTO <= 15 menit),
      data parity 100% (users=52, endpoints=1).
11. **Uji Latih Runbook Operasional (TASK-P08-012):**
    - `TestP08_012_RunbookOperasionalDrill`: rotasi master encryption key tanpa downtime tervalidasi; sinkronisasi invalidasi
      cache entitlement lintas-pod via Redis Pub/Sub berjalan instan tanpa restart pod.
12. **Observability & Release Gate (TASK-P08-013 & 014):**
    - Four Golden Signals & metrik bisnis aktif di `/metrics`. Seluruh checklist `teknis/go-live-checklist.md` disetujui (100% Passed).

---

## 2026-10-12 — P07 Slice D2: Pembersihan Tuntas BYOK (TASK-P07-013) & Penutupan Resmi P07 (18/18)

Slice penutup P07. Menuntaskan pembersihan antarmuka dan backend BYOK (Bring Your Own Key)
setelah Gateway AI terpusat (P03/P07 Slice E) terbukti mandiri melayani beban kerja AI secara live.

### Perubahan & Implementasi
- **Backend (`router.go`):** Menghapus seluruh endpoint registrasi user key (`POST /api/v1/gemini-keys`,
  `PATCH /:id`, `DELETE /:id`, `POST /:id/test`, dan `GET /gemini-keys`) beserta entri rate limit.
- **Database Migrasi 0013 (`0013_archive_byok_keys.sql`):** Aditif & idempoten. Mengarsipkan seluruh data
  kunci di tabel `gemini_api_keys` (`is_archived = true`, `status = 'inactive'`, `archived_at = now()`),
  serta menambahkan indeks `idx_gemini_keys_archived`.
- **Frontend:**
  - Mengonfigurasi `next.config.ts` untuk melakukan redirect permanen (HTTP 301) dari `/settings/ai-keys`
    dan `/user/api-keys` ke `/user/settings/subscription`.
  - Memperbarui `frontend/app/(dashboard)/user/api-keys/page.tsx` mengalihkan ke `/user/settings/subscription`.
  - Menghapus hook `frontend/hooks/useGeminiKeys.ts` dan query key `geminiKeys` di `cache-keys.ts`.
- **Dump Database:** Berkas `database/dump/fintrack_v2.{sql,dump}` diregenerasi.

### Verifikasi & Bukti Nyata
- **Backend:** `go vet ./...`, `go build ./...`, `go test ./...` 100% hijau; suite integrasi P06 & P07 hijau.
- **Frontend:** `pnpm lint` 0 error, `pnpm test` 102 passed (12 test files), `pnpm build` kompilasi sukses.
- **Database:** `SELECT count(*), is_archived, status FROM gemini_api_keys` membuktikan seluruh 30 baris berstatus archived.
- **Status P07:** 18 dari 18 task WBS (100%) resmi selesai. Fase P07 ditutup secara resmi.

---

## 2026-10-12 — P07 Slice E: Routing AI Dinamis, Weighted Round-Robin & Multi-Model

Slice kelima P07. Mewujudkan routing AI dinamis antar-model per use case, memfungsikan
failover berjenjang di dalam combo, load distribution berbobot antar-combo, dan
mengaktifkan gateway P03 secara nyata di lingkungan produksi.

### Perubahan & Implementasi
- **Smooth Weighted Round-Robin (SWRR):** Algoritma Nginx SWRR diterapkan di `service.go`
  untuk rotasi antar-combo secara proporsional terhadap bobot endpoint tanpa lonjakan burst.
  Di dalam combo, rute dievaluasi berurut berdasarkan `priority ASC` (failover lokal). Jika
  semua rute di combo pemenang gagal, failover otomatis berpindah ke combo berikutnya.
- **State In-Memory & Anti-Lockstep:** State WRR disimpan di memori `Service` dengan proteksi
  `sync.Mutex` dan di-seed secara acak pada konstruksi untuk mencegah instance multi-pod
  menabrak combo yang sama secara bersamaan.
- **Validasi Pra-Transaksi `PutRoutes`:** Memvalidasi seluruh payload (format UUID, use case valid,
  endpoint ada & aktif, tidak ada duplikat kombinasi `use_case+endpoint_id+model+combo`) sebelum
  membuka transaksi. Mencegah terhapusnya rute lama jika ada baris baru yang cacat.
- **Parameter Dimensi Embedding:** Client OpenAI gateway mengirimkan `"dimensions": 768`
  sehingga model `gemini-embedding-001` (yang defaultnya 3072d) menghasilkan vektor 768d
  yang lolos validasi ketat pgvector `vector(768)`.
- **Migrasi 0012 (`0012_ai_routes_multi_model.sql`):** Aditif & idempoten (teruji 2x).
  Memperbarui rute `vision` ke `gemini-2.5-flash`, `embedding` ke `gemini-embedding-001`,
  dan menambahkan rute sekunder & combo backup untuk `chat`.

### Verifikasi & Bukti Nyata
- **Live e2e Gateway Terverifikasi:**
  - `ChatSync`: 3 panggilan berturut-turut sukses -> `"P07-OK"` (2796 tokens, provider_reported).
  - `CreateEmbedding`: sukses -> `len=768` (10 tokens).
  - `AnalyzeVision`: sukses -> `"Merah. Padat. Murni."` (3331 tokens).
- **Unit Test Baru:**
  - Test SWRR ratio & anti-burst (urutan A B A A B A A B A pada bobot 200:100).
  - Test failover di dalam combo sebelum berpindah combo.
  - Test breaker open dilewati dari rotasi.
  - Test konkurensi goroutine aman.
  - Test validasi payload `PutRoutes` & penolakan duplikat.
  - Test pengiriman parameter `dimensions`.
- **Uji Mutasi:** Mematikan pengurangan `totalWeight` pada SWRR menyebabkan test gagal -> terbukti menangkap regresi.
- **Gerbang Penuh:** `go vet ./...`, `go build ./...`, `go test ./...` hijau; frontend `lint` 0 error, `build` hijau, `test` 102 passed.

### Catatan & Gap Terisolasi
- Voice (`whisper-1`) upstream provider merespons 429/timeout -> dicatat sebagai gap terisolasi.
- Kuota global terdistribusi (Redis) tidak dibutuhkan untuk pembagian beban (keputusan didiskusikan & disetujui).

---

## 2026-10-11 — P07 Slice D1: health gates, shutdown 30s & temuan gateway mati

Slice keempat P07 (D1; BYOK → D2). Menemukan **bug besar** sekaligus memperbaiki kebenarannya
tanpa merusak layanan.

### Temuan besar: gateway AI terpusat TIDAK PERNAH aktif

```go
gatewayEnabled := flagSvc.IsEnabled(bgCtx, "feature_ai_gateway_centralized", nil)
```

`IsEnabled` mengembalikan **false** bila `userID == nil` dan rollout parsial. Flag bernilai
`enabled=true, rollout=25` → **selalu false**; log runtime membuktikannya
(`"AI gateway centralized: disabled (feature flag off)"`). Akibatnya **seluruh P03 tidak
berjalan** — routing, breaker, metering, failover semuanya mati; 100% traffic AI lewat BYOK.

Flag itu sendiri **artefak uji**: riwayat di dump menunjukkan di-toggle via `curl` saat pengujian
P06 (`is_enabled: false→true`, `rollout: 0→25`).

### Koreksi klaim saya sendiri

Saya sempat melaporkan `ai_routes` **kosong**. **SALAH** — berisi **5 route aktif** (chat, vision,
voice, embedding, reasoning) menunjuk `9router-primary`. Output sebelumnya terpotong dan saya
salah membacanya; dikoreksi setelah memverifikasi ulang. Ini penting karena mengubah kesimpulan:
pengaman berbasis-route saja akan **membuka** gateway, bukan menahannya.

### Perubahan

- **`featureflag.IsEnabledForPlatform`** — membaca flag sebagai kill switch platform (tanpa
  bucketing user), sehingga `enabled=true, rollout=25` terbaca ON.
- **Pengaman gateway diperkuat.** Gateway hanya aktif bila flag ON **DAN** ada route chat **DAN**
  endpoint sehat (`health == "healthy"`; `unknown` tidak dihitung). Alasannya: `chatViaGateway`
  **tidak punya fallback ke BYOK**, jadi mengarahkan seluruh platform ke endpoint yang gagal akan
  mematikan AI untuk semua user. Method baru: `HasActiveRoutes`, `HasHealthyEndpoint`.
- **Health gates:** `/health/live` (tanpa I/O), `/health/ready` (PostgreSQL + pgvector + Redis
  dengan latensi per dependensi), `/health` alias live. DB mati → **503** (hard dependency);
  Redis mati → **200 `degraded`** (Redis opsional di codebase ini).
- **Shutdown:** drain `10s → 30s` via `SHUTDOWN_TIMEOUT`; compose dapat `healthcheck` backend +
  `stop_grace_period: 30s`.
- **Aturan migrasi:** `database/migrations/README.md` baru (expand/contract, larangan
  `DROP COLUMN`/`SET NOT NULL` tanpa fase perantara, `CREATE INDEX CONCURRENTLY`).
- **TASK-P07-003 terverifikasi terpenuhi tanpa kode baru:** grep `sync.Map`/`globalCache` di
  `backend/` (non-test) = nol hasil.

### Verifikasi
- `go vet`/`go build`/`go test ./...` hijau; suite integrasi (P06 + P07 A/B) hijau.
- Health diuji terhadap DB/Redis **nyata**: `postgres ok 2ms, redis ok 1ms`; DB mati → 503
  (pool diarahkan ke DSN tak valid); Redis tak dikonfigurasi → `degraded`.
- **Bukti runtime pengaman gateway:**
  `WARN AI gateway: flag is ON and a chat route exists, but no endpoint is healthy; gateway stays
  disabled (BYOK fallback in use)` → gateway mati, AI tetap jalan lewat BYOK.
- **Uji mutasi pengaman:** cek kesehatan dilepas → endpoint `unhealthy` dan `unknown` lolos
  (test gagal) → terbukti test menangkap regresi.

### Gap yang dicatat, tidak diklaim
1. **Gateway masih mati** setelah D1 (endpoint `unhealthy`) → "P03 tidak berjalan" **belum
   terselesaikan**. Ini temuan **prioritas tinggi**: butuh endpoint sehat (9router) agar gateway
   menyala. Pengaman akan membuka sendiri begitu health checker melaporkan `healthy`.
2. Template k8s tidak dibuat (`deploy/` tidak ada di repo).
3. Uji rolling restart 200 RPS tanpa 5xx tidak dijalankan.
4. **BYOK belum dihapus** → slice D2, dengan syarat gateway benar-benar melayani semua user.

---

## 2026-10-11 — P07 Slice C: Standar UI, a11y terverifikasi & optimistic update

Slice ketiga P07. Menutup standar UI dengan tooling yang benar-benar memverifikasi, bukan
klaim. Hasil sampingan: **4 kegagalan kontras nyata** ditemukan lewat perhitungan manual.

### Perubahan
- **`eslint-plugin-jsx-a11y` aktif.** 40 pelanggaran diperbaiki secara substantif — bukan
  aturan dimatikan. Termasuk: 23 label tanpa asosiasi kontrol (kini `id`/`htmlFor`), 6 klik pada
  elemen statis, 6 handler tanpa keyboard listener, 4 `autoFocus`, 1 `aria-role` invalid.
- **Perbaikan a11y yang bermakna:** dropzone struk & tombol mic dijadikan `<button>` asli
  (keyboard-operable, bukan sekadar diberi `role`); backdrop dialog jadi elemen saudara sehingga
  `stopPropagation` tidak lagi diperlukan; prop `role` di `ChatMessage` → `author`; toggle
  notifikasi memakai `<input type="checkbox">` nyata; `autoFocus` → fokus programatik.
- **Kontras diperbaiki dengan nilai terukur:** `--muted-foreground` 0.55→0.53 (4.19→4.56);
  token baru `--primary-strong` untuk teks di `bg-primary/10` (4.27→5.22, dipakai 5 tempat);
  `red-600`→`red-700` di `red-100` (3.95→5.30); `green-600` di `green-100` (**3.00**)→`green-700`
  (4.57); `red-600` di `red-600/15` (3.82→5.01).
- **Cursor pointer:** akarnya komponen dasar — `Button` tidak punya cursor sama sekali dan
  memakai `disabled:pointer-events-none` yang membuat `cursor-not-allowed` mustahil. Diperbaiki
  di komponen dasar + 79 elemen native → verifikasi akhir 0 tersisa.
- **Tabel transaksi:** `<DataTable>` + state di URL (bookmarkable) + debounce 300ms (sebelumnya
  per ketikan). Komponen baris custom dihapus.
- **Optimistic update:** update & delete optimistik dengan rollback tiga kelas error; create
  sengaja non-optimistik.

### Verifikasi
- `pnpm lint` **0 error** (54 warning, turun dari baseline 62), `pnpm build` hijau,
  `pnpm test` **102 lolos** (dari 66).
- Vitest dipecah dua project: `unit` (node, cepat) dan `dom` (jsdom + vitest-axe + jest-dom).
- axe-core: nol pelanggaran untuk ConfirmDialog, TransactionRow (semua sumber), DataTable
  (dengan data & kosong), DataTableToolbar.
- **Uji mutasi** pada rollback optimistic: `restoreLists` dilepas → 3 test gagal → terbukti
  test menangkap regresi.

### Koreksi jujur
Klaim awal saya bahwa badge "Scan" gagal kontras (3.07) **SALAH** — kode aslinya
`amber-700` di atas `amber-100` = **4.51** (lolos). Saya salah mengasumsikan background-nya
`amber-50`. Perubahan yang tidak perlu dibatalkan. Yang benar-benar gagal adalah
`muted-foreground` di `muted`, `primary` di `primary/10`, `red-600` di `red-100`,
`green-600` di `green-100`, dan `red-600` di `red-600/15`.

### Gap yang dicatat (tidak diklaim)
- **AC "Lighthouse Accessibility ≥ 95" tidak terpenuhi** — Lighthouse tidak ada di repo dan
  butuh server + browser. Digantikan oleh jsx-a11y + axe + perhitungan kontras manual.
- **AC "0ms saat menambah" tidak sepenuhnya terpenuhi** — create sengaja non-optimistik karena
  list dipaginasi + di-sort server-side, jadi posisi baris baru tidak diketahui; menampilkan
  posisi salah lalu melompat lebih buruk daripada spinner singkat.
- **TASK-P07-012 hanya sebagian** — `/budgets` dan `/categories` (user) tidak ada di codebase.

---

## 2026-10-11 — P07 Slice B: Realtime SSE + perbaikan 3 cacat invalidasi (1 bug koneksi)

Slice kedua P07. Membangun transport SSE dari nol dan menutup tiga cacat invalidasi
entitlement yang membuat perubahan paket baru terasa setelah 24 jam.

### Perubahan
- **Modul `backend/internal/realtime/`** (baru): `hub.go` (registry koneksi, batas 5/user,
  eviction terlama, cleanup saat ctx cancel), `buffer.go` (ring-buffer Redis ZSET
  `realtime:buf:{id}`, TTL 10 menit), `publisher.go` (+`redis_sub`), `handler.go`.
- **Kontrak event** di `pkg/contracts/events.go` — paket daun, tanpa impor internal, agar
  producer di modul mana pun bisa memancarkan event tanpa siklus impor.
- **Endpoint** `GET /api/v1/realtime/stream` (di luar grup `cached`); auth memakai
  `AuthMiddleware` yang sudah ada (cookie `ft_at`), bukan mekanisme token baru.
- **Fan-out Redis**: channel `realtime:u:{userID}` + `realtime:admins`; `SubscribePattern`
  ditambahkan ke pubsub. Tanpa Redis, publish jatuh ke hub lokal.
- **Lima producer**: `payment_received`, `manual_approval_needed`, `plan_updated`,
  `quota_reset`, `ai_endpoint_down`. Hook `breaker.Config.OnStateChange` baru, di-invoke
  **setelah** lock breaker dilepas.
- **Frontend**: `lib/realtime-dispatcher.ts`, `hooks/use-realtime-stream.ts`,
  `providers/realtime-provider.tsx`. Backoff manual 1s→16s + jitter (opsi A), `last_event_id`
  sebagai query param karena `EventSource` yang dibuat ulang tidak mengirim header otomatis.
- **Valve** `REALTIME_ENABLED` (default on).

### Bug yang ditemukan & diperbaiki
1. **Header SSE tidak di-flush saat connect.** `c.Status(200)` tidak mengirim header sampai
   penulisan pertama, jadi klien menunggu sampai heartbeat pertama — **15 detik** — sebelum
   melihat stream terbuka; `onopen` di browser juga tidak menyala. Ditemukan dari durasi test
   integrasi yang kelipatan tepat 15s. Setelah `flusher.Flush()`: **91.7s → 2.9s**.
2. **Invalidasi entitlement dari jalur admin memakai scope cache yang salah** (`"u:"` vs
   `"ent:"`), sehingga perubahan paket per-user tidak invalid.
3. **Mengubah fitur/limit sebuah plan tidak menginvalidasi apa pun** → seluruh subscriber plan
   memakai entitlement basi sampai 24 jam. Diperbaiki dengan komponen versi plan pada kunci
   cache: satu bump `ent:plan:{id}` menginvalidasi semua subscriber, O(1).
4. **`entBroadcaster` tidak pernah dipanggil** di `main.go` → tidak ada invalidasi lintas-instance.

### Verifikasi
- Backend: `go vet ./...` bersih, `go test ./...` hijau, `go build ./...` hijau.
- Integrasi: `tests/realtime_e2e_test.go` — 6 skenario lewat router + Redis nyata (401 tanpa
  token, 503 saat disabled, delivery event, isolasi antar user, event admin tidak bocor ke user
  biasa, resume backlog via `Last-Event-ID`).
- **Uji mutasi** pada perbaikan 4b: pemeriksaan versi plan dilepas → suite gagal
  ("still sees stale entitlement after plan edit") → terbukti menangkap regresi.
- Unit test baru: hub (isolasi, eviction, cleanup, slow-client), breaker hook (transisi,
  **anti-deadlock** — callback memanggil `State()` di dalam lock akan hang), invalidasi plan.
- Frontend: `pnpm lint` 0 error (62 warning = baseline), `pnpm build` hijau, `pnpm test`
  81 lolos (dari 66).

### Catatan
- **Gap:** `quota_reset` tidak terjadwal. Tidak ada worker rotasi siklus di codebase (cleaner
  hanya melepas reservasi kedaluwarsa); event dipancarkan saat periode baru benar-benar dibuat,
  yaitu ketika user berikutnya memakai AI. Siklus terjadwal = pekerjaan lanjutan.
- Tidak ada inbox notifikasi durabel: SSE akselerator UX, bukan sumber kebenaran.
- `LedgerService` ternyata **tidak dikonstruksi di `main.go`** (hanya dipakai test/cleaner);
  jalur produksi yang memanggil `Reserve` belum ada. Dicatat sebagai temuan, bukan dikerjakan.

---

## 2026-10-11 — P07 Slice A: Rollout entitlement & RBAC + 3 bug ditemukan

Slice pertama P07 (Rollout RBAC/Realtime). Hasil sampingan penting: implementasi ini menemukan
**tiga bug nyata**, dua di antaranya lubang keamanan.

### Perubahan
- **Enforcement dinyalakan.** `NewEntitlementMiddleware` sebelumnya **tidak pernah dikonstruksi**
  (0 dari 63 rute ter-gate; hanya `RequireUser`, jadi autentikasi tanpa otorisasi). Kini
  dikonstruksi di `cmd/server/main.go`, disuntikkan via `RouterConfig`, dan dipasang per-rute.
  Rute ter-gate: `POST /transactions` (`max_transactions`), `POST /ai/chat` (`ai_chat` +
  `ai_tokens_per_cycle`), `POST /ai/receipt` (`ocr_receipt` + `ocr_scans_per_cycle`),
  `POST /ai/receipt/transaction` (`ocr_receipt`), `POST /ai/voice` (`voice_input`),
  `GET /reports` (`reports_export`). Rute baca dibiarkan agar cache 30s tidak berubah.
- **Fallback plan default.** `GetActivePlanForUser` kini jatuh ke plan `DEFAULT_PLAN_CODE`
  (default `free`) bila user tidak punya baris `subscriptions`, dengan sentinel
  `billing.StatusNone`. Fail-closed tetap bila plan default hilang.
- **Usage resolver baru** (`middleware/usage_resolvers.go`): `max_transactions` (non-soft-deleted),
  `ai_tokens_per_cycle` (`used + reserved`), `ocr_scans_per_cycle` (ledger `modality='vision'`).
  Ditambah `GetCurrentPeriod` read-only (non-locking) di `billing_repo` — `GetPeriodForUpdate`
  yang ada memakai row-lock, tidak cocok untuk pre-flight check.
- **Kontrak blok** 403 `feature_not_in_plan` / 402 `limit_exceeded`+`limit_not_in_plan`, keduanya
  memuat `upgrade_url`. Valve `ENTITLEMENT_ENFORCEMENT` (default on) untuk rollback insiden.
- **Mirror frontend (advisory-only):** `hooks/use-entitlements.ts`, `hooks/use-role.ts`,
  `lib/entitlement-errors.ts`, halaman read-only `/user/settings/subscription`, tombol
  OCR/export dinonaktifkan sesuai paket. `vitest.config.ts` diperluas ke `hooks/**`.

### Bug yang ditemukan & diperbaiki
1. **Lockout ~96% user (blocker).** Resolver mengembalikan deny-all untuk user tanpa baris
   `subscriptions`; DB dev punya 52 user vs 2 subscription. Menyalakan enforcement tanpa
   fallback akan meng-403/402 hampir semua user. → fallback plan `free`.
2. **`super_admin` terkunci dari rute user.** `RequireUser()` hanya menerima `user`/`admin`,
   sehingga pemilik platform ditolak 403 di transaksi, dashboard, AI, dan laporan. → menerima
   ketiga role, sejalan dengan `RequireAdmin()`.
3. **Langganan kedaluwarsa tetap dapat fitur berbayar.** Query subscription tidak memfilter
   `status`, jadi baris `expired`/`canceled` terus memberi plan Pro/Business selamanya — eskalasi
   hak akses begitu enforcement menyala. → `canceled`/`expired` jatuh ke plan default.

### Verifikasi
- Backend: `go vet ./...` bersih, `go test ./...` hijau, `go build ./...` hijau.
- Integrasi: `tests/entitlement_e2e_test.go` — 8 skenario lewat **router nyata** (Free 403 di
  `/reports`, Pro 200, limit 100 → 402, soft-deleted tidak dihitung, periode token tidak ada →
  diizinkan, tanpa subscription → diperlakukan Free, langganan expired → Free, super_admin bypass,
  `upgrade_url` ada). Ini penjaga kelas bug "middleware ada tapi tidak ter-wiring".
- **Uji mutasi:** gate `reports_export` dilepas sementara → suite gagal (200 bukan 403), lalu
  dikembalikan → hijau. Suite terbukti menangkap regresi, bukan lolos karena kebetulan.
- Frontend: `pnpm lint` 0 error (62 warning, sama seperti baseline), `pnpm build` hijau
  (rute `/user/settings/subscription` terdaftar), `pnpm test` 66 lolos (dari 54).
- `fintrack_test` tanpa seed, jadi suite integrasi menyediakan katalog plan sendiri (idempoten).

### Dokumen dikoreksi
- `teknis/rbac-rollout.md`: matriks "28 endpoint" diganti matriks rute nyata; 429 → 402; nama
  kunci nyata (`max_transactions`); perilaku tanpa subscription; daftar kunci yang belum ditegakkan.
- `planning/tasks.md` + `planning/implementation-plan.md`: AC 429 → 402; status Slice A.
- `tracking/tracking.md`, `teknis/frontend-entitlement-mirror.md`: status implementasi.
- Spec & plan: `docs/superpowers/specs/2026-10-11-p07-entitlement-rollout-design.md`,
  `docs/superpowers/plans/2026-10-11-p07-entitlement-rollout-plan.md`.

### Catatan
- Belum ditegakkan (butuh pekerjaan lanjutan): `ai_requests_per_cycle` (belum ada penghitung
  request), `max_categories` (kategori dikelola admin), `max_wallets` (fitur belum ada),
  `rag_search` (menyatu di `/ai/chat`), `advanced_budgeting`, `api_access`, `bank_sync`.
- User dengan transaksi sudah di atas batas Free (mis. demo 479 vs 100) ditolak pada
  `POST /transactions`; rute baca tetap terbuka. Enforcement yang benar, bukan bug.

---

## 2026-10-11 — P06-020: Suite E2E integrasi super admin + perbaikan race review pembayaran

Penutupan P06-020 ("E2E Validation"). Hasil sampingan penting: suite ini **menemukan bug
data-integrity nyata** pada jalur persetujuan pembayaran manual, yang lalu diperbaiki.

### Perubahan
- **`backend/tests/admin_e2e_test.go` (baru)** — suite integrasi 6 skenario memakai HTTP asli
  (httptest) di atas middleware produksi (`AuthMiddleware` + `RequireSuperAdmin`), handler
  `internal/admin`, repository Postgres, dan DB nyata. Env-gated `TEST_DATABASE_URL` sehingga
  `go test ./...` tanpa infrastruktur tetap hijau (skip).
  1. Otorisasi: anon 401, user biasa 403, klaim `super_admin` palsu 403 (re-check peran hidup, K-14), demosi saat token masih valid 403.
  2. Pembuatan plan + inspeksi jejak audit (aktor, `entity=plan`, `after.code`).
  3. Race review pembayaran (8 approve bersamaan → tepat 1×200, sisanya 409).
  4. Penyesuaian token idempoten (replay `idempotency_key` → `replayed=true`, kuota tidak dobel, 1 baris ledger).
  5. Kill switch AI Gateway (persistensi ke `settings`, target tidak valid 400).
  6. Pengaturan global + feature flags (allowlist key, `updated_by`, validasi rollout 0–100).
- **Perbaikan race (3 berkas sumber + 1 fake test)**:
  - `payment/service.go` — `GetPaymentByIDForUpdate` ditambahkan ke interface `Repository`.
  - `repository/postgres/payment_repo.go` — implementasi `SELECT ... FOR UPDATE`.
  - `payment/manual_service.go` — `review()` kini memakai varian `ForUpdate` di dalam transaksi.
  - `payment/webhook_test.go` — `memRepo` memenuhi method baru.

### Temuan bug (bukti)
- Sebelum perbaikan: 8 approve bersamaan pada satu pembayaran → `[200 200 200 409 200 409 409 409]`,
  dan tanpa penjadwalan acak bisa `[200 ×8]`. Akar masalah: `review()` membaca dengan
  `GetPaymentByID` (tanpa lock); di bawah READ COMMITTED semua transaksi melihat `pending_review`
  dan semuanya menulis `settled` → event/audit ganda untuk satu pembayaran.
- Jalur webhook sudah punya lock (`GetPaymentByGatewayRefForUpdate`, dibuktikan
  `TestPayment_ConcurrentWebhookIdempotent`, 20 webhook bersamaan); jalur admin tidak pernah diberi.
- Sesuai mitigasi yang diminta `P06/planning/sprint-plan.md` (risk register: "pessimistic row-lock
  `SELECT ... FOR UPDATE`") dan `implementation-plan.md` ("cegah race condition antar-admin").

### Verifikasi
- `go vet ./...` bersih; `go test ./...` hijau; `go build ./...` hijau.
- `TEST_DATABASE_URL=... go test -count=30 -run TestAdminE2E_PaymentReviewRace ./tests/...` → hijau (90 burst).
- Uji mutasi: lock dilepas sementara → suite gagal di ronde pertama (`[200 ×8]`), lalu dikembalikan → hijau. Tes terbukti menangkap regresi, bukan lolos karena keberuntungan.
- `TEST_DATABASE_URL=... go test -run TestAdminE2E ./tests/...` → 6/6 hijau.
- **Prasyarat**: `fintrack_test` perlu dimigrasi penuh dulu (sebelumnya hanya tabel dari `0002`; kolom
  `super_admin` belum ada). Diterapkan `0001–0010` idempoten via `psql` (dokumentasi: `database/README.md`).
- **Bukan regresi**: `TestLedger_ReserveCommitRelease` gagal karena `fintrack_test` tanpa seed
  (`pricing_rules` kosong). Diverifikasi gagal juga saat perubahan ini di-`git stash` — kondisi pra-ada.

---

## 2026-10-11 — P06: Bersih-bersih lint + hapus `setState` di dalam `useEffect`

Lanjutan penutupan P06 (fitur sudah 20/20). Fokus: menghilangkan pola sinkronisasi state
anti-pattern dan menyisakan 0 error lint.

### Perubahan
- **Hook baru `frontend/hooks/use-reset-on-open.ts`** — reset form dialog saat `open` berubah
  `false → true` memakai pola React "adjust state during render" (tanpa render pass tambahan);
  param `signature` opsional untuk reset saat record yang disunting berganti.
  Dipakai di `ForceDowngradeDialog`, `CancelSubscriptionDialog`, `PaymentActionDialogs`
  (approve/reject/refund), `FeatureFlagDialog`.
- **Adopsi nilai server saat render** (bukan effect) di `AuditFilters` (4 nilai filter),
  `GlobalSettingsTab` (hidrasi form) dan `RoutingMatrixPanel` (matriks rute). Perbandingan
  memakai signature nilai, sehingga refetch latar tidak lagi menghapus suntingan lokal.
- Impor/ekspor mati dibuang: `AreaChartView`, 6 ikon lucide di `GatewayPanels`, `CardHeader/
  CardTitle/CardDescription` di `FeatureFlagsTab`, `AlertCircle`, tipe `TokenLedgerRow`,
  deklarasi `variant` di `JsonDiffViewer`; kutip ganda pada teks JSX di-escape (`&quot;`).

### Verifikasi
- `pnpm lint` → **0 error** (3 error sebelumnya: `react-hooks/set-state-in-effect` ×2,
  `prefer-const`), 62 warning sisa (mayoritas `no-unused-vars` di berkas di luar P06).
- `pnpm build` hijau; `pnpm test` → 4 berkas / 54 tes lolos.
- Browser (Next build lokal di port 3001 + backend/DB docker, login super admin):
  - Reset dialog: extend (30 hari/'' → terisi → buka ulang kembali 30/''), feature flag
    (key/nama/rollout kembali ke nilai server), paksa turunkan paket (paket default +
    alasan kosong, tombol submit `disabled`), refund (3 field kosong).
  - Matriks rute: sunting prioritas → tombol "Simpan Matriks" aktif; "Segarkan Data" →
    suntingan **tetap** (dulu terhapus); "Batalkan perubahan" → kembali ke nilai server & tombol nonaktif.
  - Filter audit: ketik email → URL `?filter_email=...` setelah debounce; "Reset Filter" → input & URL bersih.
  - Pengaturan: muat ulang halaman → form terhidrasi dari server, tombol "Simpan Pengaturan" `disabled`.

### Catatan / deviasi kecil
- Tidak ada tes unit baru untuk hook: `vitest` proyek berjalan di environment `node` tanpa
  `@testing-library/react`/jsdom; menambah dependensi hanya untuk 15 baris hook dinilai tidak
  sepadan. Perilaku dibuktikan lewat walkthrough browser di atas.


---

## 2026-10-11 — Implementasi P06 Super Admin Dashboard SELESAI (20/20, 100%)

Eksekusi `P06_Super-Admin-Dashboard/planning/implementation-plan.md` (4 milestone). Semua task ✅
setelah dibuktikan lewat suite API + walkthrough browser.

### Backend — Otorisasi, Admin API, Kill Switch
- **TASK-P06-001** ✅ middleware `super_admin.go` (+test) + grup `/api/v1/admin/*`; anon 401, user 403 (dibuktikan di smoke).
- **TASK-P06-005/008/010/012/014/018** ✅ paket `internal/admin` (12 berkas): metrics (overview, timeseries, payment-methods), plans CRUD + entitlement, subscriptions (extend/cancel/force-downgrade), payments (review/settle/refund), tokens (users/ledger/adjust 409 `NO_ACTIVE_PERIOD`), audit-logs (filter + sanitasi PDP).
- **TASK-P06-016** ✅ aigateway admin (endpoints/routes/kill-switch) + migrasi `0010_settings.sql`; enforcement kill switch di `gateway/service.go`, `breaker/circuit_breaker.go`, `embedding/killswitch.go`.
- **TASK-P06-006/013/015/017/019** ✅ feature flag repository + test; `pkg/response` dan `ai_usecase` diselaraskan.

### Frontend — Dashboard Admin
- **TASK-P06-002/003/004** ✅ layout + guard role di `proxy.ts`, sidebar, `<DataTable>` TanStack v8 + URL state (Suspense boundary ditambahkan — sebelumnya `pnpm build` gagal).
- **TASK-P06-006/007/009/011/013/015/017/019/020** ✅ 8 halaman `(dashboard)/super-admin/*` (dashboard + Recharts, plans, subscriptions, payments + review, tokens, ai-gateway, audit viewer + diff JSON, settings + feature flag).

### Verifikasi (bukti nyata)
- `bash tester/p06_smoke.sh` → seluruh endpoint admin 200; negatif sesuai kontrak: 400 `target tidak valid`, 401 anon, 403 user, 409 `NO_ACTIVE_PERIOD`.
- Browser (super admin): 8 halaman render, **0 error konsol**.
- Kill switch lewat UI: klik switch global → dialog "Ya, Aktifkan Kill Switch" → `data-state=checked`, banner "Layanan Global Dimatikan" muncul, API `global=true`, audit `entity_id=ai_kill_switch after={"enabled":true,"target":"global"}`; toggle OFF langsung tanpa dialog → API semua `false` (state dikembalikan).
- RBAC: user biasa membuka `/super-admin` → dialihkan ke `/user`.
- `cd frontend && pnpm build` hijau.

### Catatan / deviasi kecil
- Tombol "Simpan Matriks" hanya untuk perubahan matriks rute; toggle kill switch tersimpan otomatis setelah dialog konfirmasi (dibuktikan: tombol tetap `disabled` saat toggle berubah, namun API sudah terbarui).

---

## 2026-10-11 — Implementasi P05 Payments & Billing SELESAI (17/17, 100%)

Eksekusi `P05_Payments-Billing/planning/implementation-plan.md` (5 wave). Semua task ✅ setelah
dibuktikan test nyata.

### W1 — Skema, Gateway, Midtrans, Webhook
- **TASK-P05-001** ✅ migrasi `0009_payments_invoices.sql` (invoices, payments, settings, invoice_counters + seed); CHECK amount>0 terbukti; replay OK.
- **TASK-P05-002** ✅ `payment/gateway.go` (interface Gateway + models; uang int64).
- **TASK-P05-003** ✅ `payment/midtrans.go` (Snap create + basic auth, verify SHA-512 constant-time, check status, cancel); 7 test PASS.
- **TASK-P05-004** ✅ `handler/payment_handler.go` webhook (MaxBytes 1MB).
- **TASK-P05-005** ✅ `payment/webhook.go` (pull verification + row-lock idempoten); replay 5x → aktivasi 1x.

### W2 — Invoice, Manual, Review
- **TASK-P05-006** ✅ invoice_service + NextInvoiceNumber (INV/YYYYMMDD/NNNNN atomic upsert).
- **TASK-P05-007** ✅ manual.go (transfer instructions + magic-byte proof validation 5MB) + manual_service upload.
- **TASK-P05-008** ✅ approve/reject (row-lock idempoten, reason wajib) + AdminService. 6 test PASS.

### W3 — Resolver, Prorata, Downgrade, Tax
- **TASK-P05-009** ✅ registry.ResolveGateway (per-plan > global > manual) + checkout locks gateway; 4 test PASS.
- **TASK-P05-010** ✅ prorata (billing) + invoice via checkout.
- **TASK-P05-011** ✅ downgrade schedule/apply (soft-limit).
- **TASK-P05-012** ✅ tax.go TaxCalculator (settings rate, floor).

### W4 — Dunning, Grace, SSE, Audit
- **TASK-P05-013** ✅ dunning_worker (H-3/H-1/H-0 + dedup).
- **TASK-P05-014** ✅ grace_worker (grace → past_due soft-limit).
- **TASK-P05-015** ✅ realtime_publisher (user_events:{uid}, payment.settled).
- **TASK-P05-016** ✅ audit_interceptor (named financial events). 6 test PASS.

### W5 — E2E & Concurrency
- **TASK-P05-017** ✅ **Anti double-credit terbukti:** 20 webhook paralel order yang sama → **tepat 1 transisi & 1 aktivasi** (19 idempoten); invoice `paid`, payment `settled`. Nomor invoice 20× unik.
- Wiring: payment handler + routes (webhook publik + admin manual approve/reject/pending), registry, services, dunning/grace workers, `SubscriptionActivator` (bridge ke billing + invalidate entitlement), `SettingsRepo`.

### Verifikasi akhir
- `go vet`/`go build`/`go test ./...` hijau (22 paket ok).
- Backend rebuild; webhook endpoint terdaftar.

**Catatan deviasi:** `-race` tidak tersedia (tanpa gcc). Integrasi Midtrans diuji dengan mock Snap server (kredensial produksi belum ada).

---

## 2026-10-11 — Implementasi P04 Subscription & Token Core SELESAI (17/17, 100%)

Eksekusi `P04_Subscription-Token-Core/planning/implementation-plan.md` (5 wave). Semua task ✅
setelah dibuktikan test nyata.

### W1 — Skema, Seeder, Domain, Repositori
- **TASK-P04-001** ✅ migrasi `0008_subscription_token_core.sql` (8 tabel; trigger append-only `token_ledger`; constraint & idempotency unique terbukti; replay OK).
- **TASK-P04-002** ✅ `seeds/0002_subscription_token_seeds.sql` (3 plan, 24 entitlement, 18 limit, pricing; idempoten).
- **TASK-P04-003** ✅ `internal/billing` domain + interface repo.
- **TASK-P04-004** ✅ `repository/postgres/billing_repo.go` (row-lock FOR UPDATE, idempotency); 2 test PASS.

### W2 — Token Ledger 2-Fase
- **TASK-P04-005/006** ✅ `billing/ledger.go` Reserve/Commit/Release (row-lock, idempoten, release sisa reserve).
- **TASK-P04-007** ✅ `billing/cleaner.go` auto-release reservation basi.
- **TASK-P04-008** ✅ `billing/pricing.go` (integer IDR, floor). 10 test PASS.

### W3 — Pricing, Entitlement, Middleware
- **TASK-P04-009/010/011** ✅ `entitlement` resolver + cache berversi L1/L2 + broadcaster pub/sub.
- **TASK-P04-012** ✅ middleware `RequireFeature` (403) & `WithinLimit` (402), super_admin bypass, fail-closed.
- **TASK-P04-013** ✅ `GET /api/v1/me/entitlements`. 12 test PASS.

### W4 — Subscription Lifecycle
- **TASK-P04-014** ✅ `billing/lifecycle.go` state machine + audit event.
- **TASK-P04-015** ✅ `billing/subscription_management.go` prorata (integer floor) + downgrade soft-limit. 5 test PASS.

### W5 — Testing & Metrics
- **TASK-P04-016** ✅ **Anti double-spend terbukti:** 40 reserve paralel pada kuota 1000 → tepat 10 sukses, 30 ditolak, `reserved=1000` (tanpa over-spend); 10 commit ganda → `used=1000`, `reserved=0`, tanpa saldo negatif.
- **TASK-P04-017** ✅ Metrik Prometheus (`token_ledger_ops_total`, `token_quota_used`, `entitlement_resolve_total`) + handoff kontrak P05/P06.

### Verifikasi akhir
- `go vet`/`go build`/`go test ./...` hijau (semua paket ok; 21 paket).
- Backend rebuild; endpoint `/api/v1/me/entitlements` tersedia.

**Catatan deviasi:** `-race` tidak tersedia (tanpa gcc); uji race harus dijalankan di CI. Uji konkurensi dijalankan tanpa race detector namun dengan verifikasi invarian DB (reserved ≤ quota, tidak negatif).

---

## 2026-10-11 — Implementasi P03 AI Gateway Terpusat SELESAI (14/14, 100%)

Eksekusi `second_brain/2_Projects/P03_AI-Gateway-Terpusat/planning/implementation-plan.md` (5 wave).
Semua task ✅ hanya setelah dibuktikan test nyata.

### W1 — Skema, Domain, Repositori
- **TASK-P03-001** ✅ migrasi `0007_ai_gateway.sql`: `ai_endpoints` + `ai_routes` (+constraint, index, trigger), kolom arsip `gemini_api_keys`, seed 9router-primary + 5 route. Replay idempoten.
- **TASK-P03-002** ✅ `internal/domain/aigateway.go`: entity, DTO, `UseCase`, `TokenUsage`, error sentinel, interface.
- **TASK-P03-003** ✅ `repository/postgres/ai_gateway_repo.go`: load routes (join+priority), cache TTL 60s, `DecryptKey`. 3 test PASS vs DB.

### W2 — Client, Breaker, Router, Health, Metering
- **TASK-P03-004** ✅ `aigateway/client` (OpenAI-compatible: SSE parser, chat sync/stream, embedding, audio multipart). 6 test PASS.
- **TASK-P03-005** ✅ `aigateway/breaker` (closed/open/half_open + clock injection). 7 test PASS.
- **TASK-P03-006** ✅ `aigateway/gateway/service.go` (failover, kill switch, breaker gate, embedding 768 lock). 6 test PASS.
- **TASK-P03-007** ✅ `aigateway/gateway/health.go` (worker probe). 4 test PASS.
- **TASK-P03-008** ✅ `aigateway/metering` (provider + heuristic fallback). 8 test PASS.

### W3 — Adapter, Embedder
- **TASK-P03-009** ✅ Adapter Strangler Fig (`ai_gateway_adapter.go`, `ai_gateway_vision_voice.go`); dispatch gateway di Chat/AnalyzeReceipt/ProcessVoice; jalur BYOK tetap fallback. 6 test PASS.
- **TASK-P03-010** ✅ `embedding.Embedder` interface + `GatewayEmbedder` + validasi 768. 5 test PASS.

### W4 — Deprecate BYOK & Frontend
- **TASK-P03-011** ✅ handler BYOK: mutasi → 410 Gone, list → 200 `[]`; 30 kunci lama diarsipkan. 2 test PASS.
- **TASK-P03-012** ✅ nav API Keys dihapus; `/user/api-keys` → redirect `/user/settings`. `pnpm lint` 0 error; build OK.

### W5 — Integrasi & Benchmark
- **TASK-P03-013** ✅ `gateway/integration_test.go` — 5 skenario PASS (normal, failover <1s, breaker reject ke-6, timeout, half-open recovery).
- **TASK-P03-014** ✅ `gateway/benchmark_test.go` — ChatStream ~1.18ms/op, ChatSync ~0.97ms/op; 50 stream paralel tanpa hang; flag `feature_ai_gateway_centralized` wired (default OFF).

### Verifikasi akhir (live)
- `go vet`/`go build`/`go test ./...` hijau (semua paket ok).
- Backend rebuild & restart: `/health` 200; log "AI gateway centralized: disabled (feature flag off)".
- Flag ON diuji: chat lewat gateway → log mencoba `localhost:20128/v1/chat/completions`, gagal (9router mati) → user menerima pesan ramah (tanpa crash). Flag dikembalikan OFF.

**Catatan deviasi:** `-race` tidak tersedia (tanpa gcc); uji race harus dijalankan di CI. Benchmark dijalankan lokal dengan mock upstream.

---

## 2026-10-11 — Implementasi P02 Fondasi Platform SELESAI (14/14, 100%)

Eksekusi `second_brain/2_Projects/P02_Fondasi-Platform/planning/implementation-plan.md` dalam 5 wave.
Semua task ditandai ✅ hanya setelah dibuktikan test nyata (tanpa klaim palsu).

### W1 — Infrastruktur Redis
- **TASK-P02-001** ✅ Redis 7-alpine di `docker-compose.yml` (host port **6380**; 6379 dipakai stack lain), `maxmemory 256mb` + `allkeys-lru`, healthcheck `redis-cli ping` → healthy; `depends_on: redis: healthy` di backend.
- **TASK-P02-002** ✅ `internal/infrastructure/redis` (client/config/errors) dengan graceful degradation; di-wire di `main.go` + `defer Close()`. Test 4 PASS (EmptyURL/InvalidURL/Unreachable/Live).

### W2 — Cache berversi, Pub/Sub, Observability
- **TASK-P02-003** ✅ `redis/cache.go` (CacheEngine berversi + fallback memory) + rewrite `middleware/cache.go` (key `cache:u:{uid}:v{ver}:g{gver}:{hash}`); invalidasi O(1) via version bump. Test HIT/MISS/version-bump/isolasi-user/live-redis PASS.
- **TASK-P02-004** ✅ `redis/pubsub.go` (channel `cache:invalidate`); publish saat invalidate + subscriber di main (stop on shutdown). Test disabled + live PASS.
- **TASK-P02-013** ✅ `internal/infrastructure/metrics` (four golden signals + counter bisnis); `/metrics` wired. Test PASS; **live `/metrics` 200**.
- **TASK-P02-005** ✅ migrasi `0005_create_audit_logs.sql`; trigger append-only menolak UPDATE & DELETE (terbukti via psql); replay idempoten.

### W3 — Audit & Feature Flag
- **TASK-P02-006** ✅ `internal/audit` (Service async + worker batch + masker + repo pgx); wired di main. Test 9 PASS (non-blocking <10ms, flush-on-close, batching, masking); integrasi live DB PASS.
- **TASK-P02-007** ✅ migrasi `0006_create_feature_flags.sql`; unique key menolak duplikat; replay idempoten.
- **TASK-P02-008** ✅ `internal/featureflag` (evaluator fail-closed, rollout deterministik, cache Redis/memory, repo pgx). Test 8 PASS.

### W4 — DataTable & URL State
- **TASK-P02-009** ✅ `pkg/pagination` (parser + envelope); clamp per_page 10-100, whitelist sort anti-injection. Test 9 PASS.
- **TASK-P02-010** ✅ `lib/table-params.ts` (murni) + `hooks/use-data-table-url-state.ts`. Test 13 PASS.
- **TASK-P02-011** ✅ `components/data-table/*` (TanStack Table v8.21.3, server-side, empty/error/skeleton).
- **TASK-P02-012** ✅ toolbar (search debounce 300ms + filter chips) & pagination (10/20/50/100, numbering, cursor-pointer). `pnpm lint` **0 error**; `pnpm build` OK; `pnpm test` 21 PASS.

### W5 — Verifikasi
- **TASK-P02-014** ✅ `internal/integration/foundation_test.go` (4 skenario: cache versioning, audit persist+mask, feature flag, pagination) — PASS terhadap Redis+PostgreSQL live.
- **Verifikasi akhir:** `go vet`/`go build`/`go test ./...` hijau; `pnpm lint` 0 error/build/test hijau; **smoke test live**: `/health` 200, `/metrics` 200, login 200 + cookie, `X-Cache: HIT` (cache berversi Redis aktif).

**Catatan deviasi:** Redis host port = **6380** (6379 dipakai stack lain di mesin dev). TanStack Table dipin ke **v8** (bukan v9) sesuai plan. `psql` tidak di PATH → migrasi diterapkan via `docker exec`.

---

## 2026-10-11 — Brainstorming & Analisis: Platform Berlangganan + AI Gateway Terpusat (P02–P08)

Ringkasan: sesi brainstorming (skill `brainstorming`, jalur **architectural**) atas 5 pilar permintaan pemilik
produk — (1) kunci AI terpusat via 9router, (2) subscription & sistem token, (3) dashboard/manajemen super
admin, (4) RBAC berbasis entitlement, (5) standar data-table/caching realtime/zero-downtime. Belum ada
kode yang ditulis; seluruh keluaran berupa dokumen.

### Keputusan terkunci (K-01..K-30)
- Tenancy per-user (B2C); role hanya `super_admin` + `user` (admin lama dipromosikan).
- Token: semua fitur AI; harga dinamis (base per 1k + markup kali/persen + pengali modality).
- Siklus **per-hari** (TZ Asia/Jakarta), reset tiap siklus, tanpa rollover; renewal **hybrid**.
- Upgrade **instan prorated**; downgrade **akhir siklus**, soft-limit (tanpa hapus data).
- Switch payment gateway: global + override per-plan (berlaku checkout baru; in-flight tetap gateway asalnya).
- Entitlement = fitur (bool) + limit numerik; penegakan **backend**; realtime via **cache berversi** (bukan JWT).
- AI gateway: pool multi-endpoint 9router + failover + circuit breaker + kill switch; embedding dikunci 768d.
- Ledger token **append-only** (reserve→commit/release) + idempotency; uang = **integer rupiah**.
- Infra: **tambah Redis** (cache bersama + pub/sub). DataTable: TanStack Table v8 + URL state.
- Zero downtime: migrasi **expand/contract** + rolling deploy + feature flag.
- Pendekatan: **hibrida A+B** (incremental Strangler Fig + modul baru bersih ber-interface).

### Dokumen yang ditulis
- `0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/` (8 file: ringkasan, analisis, kritik & saran best-practice, keputusan, pertanyaan terbuka, peta jalan, **jawaban+rekomendasi**, **konfigurasi seed awal**).
- `4_Resources/Dokumen-Spesifikasi/` — BRD, PRD, dan Diagram (Use Case, Flowchart, Sequence, Activity, ERD).
- `2_Projects/` — 7 fase baru: P02 Fondasi Platform, P03 AI Gateway Terpusat, P04 Subscription & Token Core, P05 Payments & Billing, P06 Super Admin Dashboard, P07 Rollout RBAC/Realtime, P08 Hardening & Go-Live (masing-masing `analisis/planning/rules/teknis/tracking`).
- `3_Areas/` — diisi 5 area lama + tambah `billing-pembayaran/` & `ai-gateway/` (kebijakan + runbook).
- README indeks diperbarui: `0_Inbox`, `2_Projects`, `3_Areas`, `4_Resources`.

### Kritik & saran (diterima semua)
Uang integer; ledger transaksional; blast radius kunci terpusat; kualifikasi "realtime"; cache invalidation;
downgrade tidak hapus data; jangan percaya callback klien; PCI-DSS SAQ-A; konvensi prorata eksplisit;
audit append-only; UU PDP No. 27/2022; zero downtime realistis; observability; YAGNI.

### Rencana implementasi (writing-plans)
- `2_Projects/IMPLEMENTATION-PLAN.md` — master plan P02–P08 (prinsip, aturan migrasi, alokasi nomor migrasi 0005–0012, DoD, gate kualitas, risiko).
- Plan detail semua fase: `P0X_*/planning/implementation-plan.md` (P02: 196 baris, P03: 350, P04: 403, P05: 403, P06: 537, P07: 366, P08: 555). Masing-masing: tujuan & DoD, wave, detail task (target file, langkah, verifikasi, bukti), matriks bukti, perintah verifikasi, rollback, handoff.
- **Koreksi status:** `tracking.md` fase P02/P04/P06/P08 yang keliru menandai "100% selesai" & memuat hasil uji fiktif → diperbaiki menjadi ⬜ belum dimulai (P02 sprint-plan checkbox juga di-uncheck).
- **Konsistensi:** nomor migrasi diseragamkan ke alokasi kanonik master plan (0007 ai_gateway, 0008 subscription_token_core, 0009 payments_invoices, 0010 settings, 0011 archive_byok); path `frontend/src/` → `frontend/`; `backend/database/migrations/` → `database/migrations/`.

### Langkah berikutnya
Eksekusi P02 dimulai setelah konfirmasi pemilik produk; fase P03–P08 mendapat plan detail saat dimulai.

---

## 2026-10-11 — Eksekusi Migrasi FinTrack v2 SELESAI (Plan `fintrack-v2-migration`)

Ringkasan: repo baru `D:/project/finance/fintrack_v2` dibangun dari nol (Go + PostgreSQL murni + Next.js), data lama dimigrasi penuh, seluruh verifikasi hijau. Repo lama & volume Supabase lama **tidak disentuh**.

### Fase 0 — Scaffold monorepo & Second Brain ✅
- Monorepo `frontend/ backend/ database/ second_brain/ tester/`, `git init` (main).
- Skill SecondBrain global + struktur PARA + aturan multi-IDE + README.

### Fase 1 — Database (schema final) ✅
- Migrasi bernomor idempoten: `0001_extensions` (vector), `0002_tables` (13 tabel, tanpa RLS/`auth.*`), `0003_indexes`, `0004_roles` (`fintrack_app` least-privilege, DML-only).
- `seeds/0001_demo.sql` (4 user bcrypt cost 12, kategori, 390 tx, SPK, chat) — idempoten.
- Scripts: migrate/dump/restore/backup/verify (autodetect psql Windows). DB `fintrack_test` untuk integrasi.
- Catatan: jumlah tabel = **13** (plan menulis "14"; `ai_conversations` sengaja tidak dibuat sebagai dead code, jadi daftar eksplisit plan = 13).

### Fase 2 — Backend (Go) ✅
- Modul → `github.com/fintrack/backend`; deps Supabase/genai dibuang (`go mod tidy`).
- Context dipropagasi ke seluruh layer (0 `context.Background()` di `internal/`).
- Auth Go-native: bcrypt cost 12 + JWT HS256 (15m) + refresh token rotasi (deteksi reuse → revoke semua); cookie `ft_at`/`ft_rt` httpOnly; `OriginCheck` menggantikan CSRF no-op.
- Repo `users` baru; perbaikan kontrak admin (`PUT /admin/users/:id/status`, `GET /admin/subcategories?category_id=`, `GET /admin/users` = array).
- SPK: queue dedupe + debounce + advisory lock + self-heal `GetLatest`.
- RAG: embedder Gemini REST (`text-embedding-004`) + `embedding_repo` + retrieval di `Chat` + `cmd/backfill`.
- Cache per-user + invalidasi prefix nyata; optimistic lock (409); dashboard errgroup.
- Test: unit (password/token/spk_queue/embedding text/cache) + integrasi env-gated (auth reuse, isolasi, optimistic lock, SPK race). `go vet/test/build` hijau.

### Fase 3 — Migrasi data dari Supabase lama ✅
- `supabase start` (port 54322); data lama: 9 user, 479 tx, 30 key, 58 SPK, 33 sesi, 114 pesan.
- `01_extract.sh` → 12 CSV; `02_transform.sql` → staging + map `auth.users`+`profiles` → `users`.
- `03_verify.sql` + `verify_output.txt`: count & `sum(amount)` per user **identik**.
- Login hash migrasi (`budi`, `admin`) → 200 + cookie HttpOnly. `supabase stop` (volume utuh).

### Fase 4 — Frontend (Next.js) ✅
- Copy `apps/web` → `frontend`; buang `@supabase/*`, `react-dropzone`, `zustand`; tambah `jose`, `next-themes`, `vitest`.
- `lib/session.ts`, `lib/auth-client.ts`, `lib/format.ts`, `lib/session-context.tsx`; `proxy.ts` (jose, guard role); `lib/api.ts` single-flight refresh; `next.config.ts` rewrites `/api → backend`.
- UI/UX: font Inter + JetBrains Mono; `Money` component (font-mono); token warna (hapus hex hardcoded di landing/auth); konsolidasi RupiahInput; EmptyState/ErrorState; a11y (aria-label, alt, label, focus-visible); dark mode toggle; media query via `useSyncExternalStore`.
- Hapus semua `: any` (0 lint error); `pnpm build` + `pnpm lint` + `pnpm test` (8 test) hijau; 0 referensi `supabase`.

### Fase 5 — Compose & env root ✅
- `docker-compose.yml` 3 service (db 5435 / backend 8080 / frontend 3001) + healthcheck.
- Dockerfile backend (`golang:1.27-alpine`) & frontend (`node:22` + pnpm 11).
- Rate limit overridable via env (untuk load test). `proxy.ts` matcher mengecualikan `/api`.
- k6: p95 23.6ms, checks 100%, exit 0 (hasil di `tester/k6-result.txt`).

### Fase 6 — Finalisasi ✅
- Dump final (`database/dump/fintrack_v2.{dump,sql}`, 260KB) + `verify_output.txt` di-commit.
- Backup drill: restore ke DB scratch → count identik → drop.
- Commit bertahap per fase.

### Catatan / deviasi kecil
- `GEMINI_EMBEDDING_API_KEY` tidak tersedia → RAG dijalankan dalam mode disabled-graceful; plumbing diuji dengan embedding sintetis (cosine 1.0) dan `cmd/backfill` memverifikasi kebutuhan key. Jalankan `cd backend && go run ./cmd/backfill` setelah key diisi untuk backfill penuh.
- Rate limit login dinaikkan via env khusus untuk profil k6 (default produksi tetap 5/menit).
## 2026-10-13 — Hotfix: Persistensi Mode Chat, Sorting Transaksi, & UI Polish
- **Database**:
  - Diterapkan migrasi `0015_chat_message_mode.sql` (`ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS mode text;`) ke instance database PostgreSQL.
  - Snapshot dump final diperbarui (`database/dump/fintrack_v2.dump` & `.sql`).
- **Backend**:
  - `ai_gateway_adapter.go`: Tambahkan logging jika insert pesan chat gagal.
  - `transaction_handler.go` & `transaction_repo.go`: Implementasi dynamic `ORDER BY` untuk kolom `sort` (`description`, `type`, `category`, `amount`, `date`) dan `order` (`asc`/`desc`).
- **Frontend**:
  - `ChatWindow.tsx`: Hapus `placeholderData` lintas-sesi yang menyebabkan tabrakan pesan lama, sinkronisasi mapping `mode`/`modelUsed` pada `switchSession`.
  - `TransactionColumns.tsx`: Kolom dedicated `Tipe` (badge warna), integrasi sorting `sort`/`order`.
  - `use-data-table-url-state.ts`: Perbaiki import `FILTER_PREFIX`.
  - `user/transactions/page.tsx`: Penyesuaian header dialog detail agar tombol close `[X]` tidak menimpa badge tipe transaksi.

## 2026-10-13 — P0+P1: Alur Pembayaran Self-Service & Penyimpanan Bukti (Local ↔ R2)

**Konteks:** Audit menemukan alur pembayaran manual buntu (deadlock): user checkout →
status `pending`, tetapi tidak ada endpoint unggah bukti, sehingga admin tidak pernah
menerima status `pending_review` dan approve selalu 409. Halaman user juga hanya
read-only tanpa riwayat, invoice, atau downgrade.

### P0 — Membuka Deadlock Pembayaran
- **Modul baru `internal/storage/`** (driver-agnostic object store):
  - `Store` interface + `LocalStore` (disk, atomic write) + `R2Store` (Cloudflare R2 / S3 via aws-sdk-go-v2).
  - `Provider` memilih driver aktif saat runtime dari setting `storage_driver` (cache 5 dtk) dengan **fallback lintas-driver** saat membaca; `ProofStore` adapter untuk modul payment.
  - Validasi magic-byte, batas 5 MB, sanitasi key anti-traversal; penyajian via `GET /api/v1/payment-proofs/:id` (pemilik atau super_admin).
- **Endpoint baru (user):** `POST /payments/:id/proof` (multipart/raw), `GET /payments/:id/instructions`.
- **Instruksi transfer** kini hanya memuat field yang benar-benar dikonfigurasi (bug "transfer ke ''" diperbaiki).
- **Super Admin:** setting `storage_driver` (enum local|r2) + `GET /admin/settings/storage` (status/kesehatan driver); ubah setting langsung meng-invalidasi cache driver.

### P1 — Halaman User Self-Service
- **Endpoint baru:** `GET /plans` (katalog publik dinamis + fitur/limit), `GET /payments`, `GET /payments/:id`, `GET /invoices`, `GET /subscriptions/me`, `POST/DELETE /subscriptions/me/downgrade`, `POST /subscriptions/me/cancel`.
- **Frontend:** halaman `user/settings/subscription` ditulis ulang menjadi hub billing (tab Paket/Pembayaran/Tagihan), komponen `billing/*` (PlanCard dinamis, PaymentRow + unggah bukti, CheckoutDialog, TransferInstructions), hook `use-billing.ts`; menu "Langganan" ditambahkan di Sidebar & Header. Harga/fitur/limit **tidak lagi hardcoded** (sumber: katalog backend).
- **Bug ditemukan & diperbaiki:** `BillingRepo.Update` tidak mempersist `metadata`, sehingga `scheduled_plan_code` (downgrade) hilang → kini `metadata` ikut di-update.

### Verifikasi
- `go vet ./...`, `go build ./...`, `go test ./internal/... ./pkg/...` hijau (paket `storage` + `payment` bertambah test: driver lokal, driver R2 via server S3 tiruan in-process, perpindahan driver runtime, ownership proof, downgrade soft-limit).
- Frontend: `pnpm lint` 0 error, `pnpm build` hijau, `pnpm test` 130/130 lulus.
- E2E live (via proxy frontend): checkout → unggah bukti (`pending_review`) → admin lihat antrean → approve → paket naik ke Pro (`status: active`); proof tersaji 200 ke pemilik & admin.
- Switch storage terbukti live: `local → r2` (unggahan baru masuk R2) → kembali ke `local`, objek R2 lama tetap terbaca (fallback); objek lokal & R2 sama-sama tersaji 200.
- Dokumentasi: `docs/storage-pembayaran.md` (panduan R2), `backend/.env.example` (+`STORAGE_*`, `R2_*`), README diperbarui.

## 2026-10-13 — P2: Sinkronisasi Dokumentasi (Tracking & Kontrak Error)

**Konteks:** Audit awal menemukan dokumentasi fase tidak konsisten dengan kode: ringkasan tracking
bertentangan dengan tabelnya, `upgrade_url` masih `/pricing` (kode memakai `/user/settings/subscription`),
dan kode error/status HTTP pada dokumen tidak sesuai implementasi.

### Diperbaiki
- **P04 tracking:** baris ringkasan `17 ⬜ / 0 ✅ / 0%` → `0 / 0 / 17 / 100%`; log pembaruan diisi
  (implementasi, aktivasi enforcement, UI swalayan); checklist DoD ditandai `[x]`.
- **P02 tracking:** header `BELUM DIMULAI` → `SELESAI (14/14, 100%)` (tabel sudah 14/14).
- **P05 tracking:** log riwayat menambahkan entri implementasi + pelengkapan UI swalayan.
- **`upgrade_url`:** diseragamkan ke `/user/settings/subscription` di seluruh dokumen P04 & P07
  (`planning/implementation-plan.md`, `rules/implementation-rules.md`, `teknis/enforcement-middleware.md`,
  `teknis/rbac-rollout.md`, `teknis/frontend-entitlement-mirror.md`).
- **Kontrak error disamakan dengan kode nyata:** fitur → `403 feature_not_in_plan` (+`feature_key`,
  `upgrade_hint`); limit paket → `402 limit_exceeded` (+`limit_key`, `limit_value`, `current_usage`);
  limit absen → `402 limit_not_in_plan`. Catatan ditambahkan: `429` hanya untuk rate limiter &
  kuota ledger token (`insufficient_quota`), bukan limit paket.
- **Audit:** `04-billing-checkout-flow-gap.md` ditandai **SELESAI**; `07-master-implementation-plan.md`
  TASK-PAY-01 & TASK-PAY-02 ditandai ✅.

### Verifikasi
- Grep akhir: **0** sisa `upgrade_url` `/pricing` atau `/settings/subscription` (kecuali catatan historis).
- Payload live dibuktikan cocok: `GET /reports` (user Free) → `403 feature_not_in_plan`,
  `upgrade_url=/user/settings/subscription`.
- `go build ./...` & `go vet ./...` hijau; `go test ./internal/... ./pkg/...` hijau; `pnpm test` 130/130.

## 2026-10-13 — Analisis Entitlement, Kuota, & Dinamisasi + Fix Bug Sidebar

**Konteks:** Tinjauan halaman Langganan memunculkan pertanyaan: bagaimana batas fitur/kuota
bekerja, bagaimana race condition-nya, dan permintaan agar semua metrik/plan/pricing dinamis
dari Super Admin (tanpa hardcode).

### Bug diperbaiki
- **BUG-01 (sidebar):** `/user/settings/subscription` menyalakan dua menu ("Langganan" &
  "Pengaturan") karena `startsWith` tumpang tindih. Dibuat `frontend/lib/nav.ts`
  (longest-prefix matching) + test `lib/nav.test.ts` (7 test); `Sidebar.tsx` & `Header.tsx`
  memakai `isNavItemActive()`. Verifikasi: `pnpm test` 137/137, `pnpm build` hijau.

### Temuan kritis (bukti kode + uji live)
- **GAP-01** OCR/vision tidak menulis ledger → `ocr_scans_per_cycle` tak pernah bertambah
  (live: 4 scan → 0 entri ledger).
- **GAP-02** Chat menagih 1 token flat, bukan token asli (live: 15 commit `amount=1`).
- **GAP-03** `ai_requests_per_cycle` tidak ditegakkan di mana pun.
- **GAP-04** Pricing engine tidak di-wire → `cost_idr` selalu 0.
- **GAP-05** `max_categories`/`max_wallets` tanpa usage resolver.
- **GAP-06** `pricing_rules` tanpa endpoint admin (hanya SQL).
- **GAP-07/08** User tak melihat sisa kuota; error 429 tak dipetakan ke toast upsell.
- **GAP-09** `ApplyScheduledDowngrade` tidak pernah dipanggil.
- **GAP-10** Token cleaner tidak di-start → reservasi menggantung terkunci.
- **GAP-11** Grace/dunning worker di-wire `nil` → transisi status tidak terjadi.

**Kesimpulan:** mekanisme inti (row lock, idempotency, cache berversi, fail-closed) solid;
yang bermasalah adalah kelengkapan cakupan akuntansi & beberapa worker yang tidak di-start.

### Dokumentasi
Dibuat folder `second_brain/0_Inbox/ide/2026-10-13_Analisis-Entitlement-Kuota-Dinamis/`:
- `README.md` (ringkasan eksekutif + peta temuan)
- `01-analisis-arsitektur-entitlement.md` (entitlement vs RBAC, resolusi, cache, isolasi plan, race)
- `02-kalkulasi-kuota-dan-ledger.md` (siklus, rumus, ledger 2-fase, skenario kuota menipis)
- `03-temuan-gap-dan-race-condition.md` (11 temuan + bukti + reproduksi)
- `04-dinamisasi-admin-tanpa-hardcode.md` (inventaris hardcode + desain endpoint/UI)
- `05-implementation-plan.md` (6 fase, checklist `[ ]`)
- `06-testing-plan.md` (matriks test, checklist `[ ]`, skrip smoke)

### Verifikasi
- `pnpm test` 137/137 ✅; `pnpm build` hijau ✅; `pnpm lint` 0 error ✅.
- `go build ./...` & `go vet ./...` hijau ✅.

## 2026-10-13 — F1: Buka Blokir Akuntansi Kuota AI

**Konteks:** F1 dari rencana `2026-10-13_Analisis-Entitlement-Kuota-Dinamis`. Tiga gap kritis
membuat pembatasan kuota AI tidak benar-benar berjalan.

### Diperbaiki
- **F1-01 (OCR/vision → ledger):** `AnalyzeReceipt` kini `Reserve(1, ModalityVision)` →
  call → `Commit(usage)`/`Release`. `ocr_scans_per_cycle` akhirnya menghitung, dan token
  vision dibebankan. Bukti live: 3 OCR → `vision reserve=1 ×3`, `vision commit ≈3750 ×3`;
  limit 5 → `402 limit_exceeded` (`current_usage:14`).
- **F1-02 (token asli chat):** `gatewayStream` menangkap `usage` dari chunk terminal
  (`stream_options.include_usage`), mengakumulasi lintas agent-loop, meneruskan pada event
  `done` (`ChatStreamEvent.Usage`); handler `Commit` token asli dengan fallback
  `metering.FromProvider`. Bukti live: 1 chat → `commit=9945` (sebelumnya selalu 1).
- **F1-03 (`ai_requests_per_cycle`):** tambah `CountAIRequestsInPeriod` + resolver + gate di
  `/ai/chat`, `/ai/receipt`, `/ai/voice`. Bukti live: usage 9, limit 8 → `402`.

### Berkas disentuh
- Backend: `internal/domain/chat.go`, `internal/usecase/ai_gateway_adapter.go`,
  `internal/usecase/ai_gateway_vision_voice.go`, `internal/usecase/ai_usecase.go`,
  `internal/delivery/http/handler/ai_handler.go`,
  `internal/delivery/http/middleware/usage_resolvers.go`,
  `internal/delivery/http/router.go`,
  `internal/repository/postgres/billing_repo.go`, `cmd/server/main.go`,
  `internal/storage/storage.go` (harden sanitasi key).
- Test baru: `middleware/usage_resolvers_test.go`, `repository/postgres/usage_counters_test.go`.
- Dokumen: checklist F1 di `05-implementation-plan.md` & `06-testing-plan.md` ditandai,
  status GAP-01/02/03 di `03-temuan-gap-dan-race-condition.md` & README.

### Verifikasi
- `go vet ./...`, `go build ./...` hijau.
- `go test ./internal/... ./pkg/...` hijau; `go test ./tests/...` (E2E) hijau.
- Integrasi `TEST_DATABASE_URL` → `TestBillingRepo_CountAIRequestsAndVisionScans` PASS.
- Uji live: OCR/chat/limit semuanya terbukti (lihat bukti di atas).

## 2026-10-14 — F2: Verifikasi Independen Aktivasi Pricing + Perbaikan Test Flaky

**Konteks:** F2 (aktifkan pricing) diminta dikerjakan. Saat recon ditemukan repo sudah memuat
commit yang mengimplementasikan F2/F3/F5/F6 (`c602eef`, `9b99da1`). Sesuai prinsip engagement
ini — **verifikasi, jangan percaya pesan commit** — F2 diverifikasi ulang terhadap instance live.

### Hasil verifikasi F2 (terbukti bekerja)
- **F2-01 (`cost_idr` terisi):** chat 9.956 token → `cost_idr=3733` =
  `floor(300×9.956×1.25×1.0)`; vision 3.729 token → `cost_idr=3495` =
  `floor(300×3.729×1.25×2.5)`. Sebelumnya selalu `0`.
- **F2-02 (rule live):** `PUT /admin/pricing-rules/:id` markup 1.25→2.0 → chat berikutnya
  `cost_idr=6006` (`floor(300×10.011×2.0)`); cost/token `0.3749`→`0.5999`. Revert 1.25 sukses.
  Engine di-refresh tiap 60s + hook admin.
- **Wiring:** `billing.NewPricingEngine` → `LedgerService` (bukan `nil`);
  `startPricingRefresh` aktif; `GetActiveRule` dipanggil saat boot.
- **F5 (bonus):** `/admin/pricing-rules` CRUD + `/super-admin/pricing` UI ada; mutasi teraudit.

### Perbaikan test flaky (bukan bug produk)
- `TestLedger_CleanerReleasesStale` — memakai `MaxAge: time.Nanosecond` yang balapan dengan
  `created_at` reserve (cutoff `now - maxAge`). Ditulis ulang: INSERT reserve bertanggal lampau
  eksplisit + reserve segar, lalu assert hanya yang lama di-release.
- `TestBillingRepo_TokenPeriodAndLedger` — idempotency key turunan `period.ID` (deterministik)
  bentrok antar-run pada DB yang sama. Diperbaiki dengan suffix UUID.
- Verifikasi: 3× `go test ./internal/repository/postgres/` berturut-turut **0 FAIL**.

### Catatan
- `TestP08_003_GatewaySustainedLoad` masih gagal (405 RPS vs target 500) — **environmental**,
  tidak menyentuh pricing/ledger, sudah ada sejak audit awal.
- **F4 (max_categories/max_wallets)** masih pending; `entitlement/catalog.go` menandainya
  `Enforced: false` secara eksplisit.

### Verifikasi
- `go vet ./...`, `go build ./...` hijau; `go test ./internal/... ./pkg/...` hijau.
- `go test ./tests/` (E2E) hijau kecuali P08 perf (pra-ada).
- Frontend: `pnpm build` hijau (route `/super-admin/pricing`), `pnpm test` 137/137.
## 2026-10-14 — Bugfix: AI Chat Flash / Reset ke Initial State
- **Root Cause**:
  1. `frontend/hooks/useAiChat.ts`: pemanggilan `fetch("/api/v1/ai/chat")` tidak menyertakan header `"X-Requested-With": "XMLHttpRequest"`. Middleware `OriginCheck` menolaknya dengan HTTP 403 Forbidden seketika (~10ms).
  2. `frontend/components/ai/ChatWindow.tsx`: `useEffect` penyeragaman pesan menempatkan `isStreaming` sebagai dependency. Ketika stream gagal dalam 0.1s (`isStreaming` jadi `false`), efek tersebut langsung menimpa state `messages` lokal dengan `sessionMessages` dari TanStack Query yang masih berupa array kosong `[]`, sehingga tampilan langsung kembali ke landing kosong ("Tanyakan apapun tentang keuangan Anda").
- **Solusi**:
  1. `frontend/hooks/useAiChat.ts` & `frontend/lib/auth-client.ts`: tambahkan `"X-Requested-With": "XMLHttpRequest"` pada request POST.
  2. `frontend/components/ai/ChatWindow.tsx` & `InlineAiAssistant.tsx`: gunakan `lastLoadedSessionIdRef` dan guard `sessionMessages.length >= messages.length` sehingga query kosong tidak pernah menghapus pesan yang sedang/baru saja dikirim. Inisialisasi `currentSessionId` dan `pendingAutoSend` via lazy `useState`.
- **Verifikasi**:
  - `pnpm test`: 173 passed.
  - `pnpm lint`: 0 errors.
  - Curl endpoint SSE streaming mengembalikan `200 OK` dengan event stream lengkap.

## 2026-10-14 — Dashboard Transaksi Terbaru: Penomoran & Layout Tabel Terstandarisasi
- **Fitur / UI**:
  - `frontend/components/transactions/TransactionTable.tsx`: Diganti dari tampilan list `TransactionRow` menjadi `DataTable` penuh dengan kolom: `#` (penomoran 1..10), `DESKRIPSI` (dengan tanggal & jam), `TIPE` (badge status warna), `KATEGORI` (badge nama kategori), dan `JUMLAH` (format Rupiah mono dengan prefix).
  - `frontend/components/transactions/TransactionColumns.tsx`: Opsi `hideActions` ditambahkan untuk menyembunyikan kolom aksi saat tabel dirender pada dashboard ringkasan.
  - `frontend/app/(dashboard)/user/page.tsx`: Limit transaksi dinaikkan menjadi maksimal 10 (`perPage: 10`), dihubungkan dengan `categoryMap` untuk resolusi nama kategori yang tepat.
- **Verifikasi**:
  - `pnpm vitest run`: 183 passed (26 test files).
  - `pnpm lint`: 0 errors.
  - `pnpm build`: Berhasil tanpa error (`31/31` pages).
## 2026-10-14 — F4: Limit Kategori/Dompet → Keputusan "UI Jujur"

**Konteks:** F4 meminta menegakkan `max_categories` & `max_wallets`. Recon menemukan kedua
limit **tidak dapat ditegakkan** dengan skema saat ini.

### Temuan (bukti)
- **`max_categories`:** kategori bersifat **global & admin-only** — `income_categories`,
  `expense_categories`, `savings_categories` **tidak punya kolom `user_id`**; mutasi hanya
  via `adminRoutes.POST /categories/*`. Tidak ada "kategori milik user" untuk dihitung.
- **`max_wallets`:** **tidak ada entitas dompet** — tidak ada tabel `wallets`/`accounts`;
  `savings_categories` bukan dompet dan juga global.

### Keputusan produk (dikonfirmasi user)
**UI jujur** — limit yang tidak ditegakkan **tidak ditampilkan** sebagai benefit paket, dan
admin diperingatkan. Backend jadi sumber kebenaran lewat flag `enforced`.

### Implementasi
- **Backend:** `payment.CatalogService` kini menyertakan `limit_meta` (key/label/unit/enforced)
  di `GET /api/v1/plans`, bersumber dari `entitlement.LimitCatalog`; `NewCatalogService`
  menerima katalog limit. Wiring di `cmd/server/main.go`.
- **Frontend:** `lib/plan-limits.ts` (`enforcedLimitEntries`, `limitLabel`) menyaring limit
  `enforced=false`; `PlanCard` hanya menampilkan limit yang berlaku; `PublicPlan.limit_meta`
  ditambah di `hooks/use-billing.ts`.
- **Admin:** `hooks/useEntitlementCatalog.ts` + peringatan di `PlanFormDialog` saat operator
  mengonfigurasi limit yang belum ditegakkan.

### Verifikasi
- Live: `GET /plans` → `max_categories.enforced=false`, `max_wallets.enforced=false`,
  sisanya `true`. Halaman Langganan hanya menampilkan 4 kuota berlaku.
- Test baru: backend `TestCatalog_UnenforcedLimitsCarryFlag` + `TestCatalog_NoLimitMetaWhenNotConfigured`;
  frontend `lib/plan-limits.test.ts` (7 test).
- `go vet`/`go build`/`go test ./internal/... ./pkg/...` hijau; `./tests/` E2E hijau;
  `pnpm build` hijau; `pnpm test` 183/183 (3 kegagalan awal = flaky timing pada
  `SearchCommand`/`NotificationBell` dari kerja paralel — lulus saat dijalankan ulang & terisolasi).

### Catatan pertanyaan pricing (refresh 60s)
Dijelaskan: `CostIDR` **tidak menyentuh DB** (murni in-memory di bawah RWMutex); refresh
adalah **1 query/menit per proses** (bukan per-user), jadi tidak tumbuh dengan jumlah user;
dan sudah ada jalur **realtime** via hook admin (`WithPricingReload`) — polling hanya jaring
pengaman untuk instance lain.

## 2026-10-14 — F6-04: Lengkapi UI Pemantauan Kuota Super Admin

**Konteks:** F6-04 sebelumnya hanya punya API (`GET /admin/quota-monitor`) tanpa UI —
tidak ada halaman, hook, atau menu. Dilengkapi menjadi fitur penuh.

### Temuan
- `/admin/quota-monitor` berfungsi (200, daftar user yang melewati ambang), tetapi **tidak
  ada UI sama sekali** → operator tidak bisa mengaksesnya.
- Grafik per-modalitas ternyata **sudah tersedia datanya** di `GET /admin/metrics/timeseries`
  (`tokens_chat/vision/voice/embedding`) → tidak perlu endpoint baru.

### Implementasi
- **Hook:** `hooks/useSuperAdminQuotaMonitor.ts` (threshold sebagai param, cache key per ambang).
- **Halaman:** `app/(dashboard)/super-admin/quota-monitor/page.tsx` — filter ambang via URL
  (`?threshold=`), tombol Segarkan, Suspense boundary (pola sama dengan halaman tokens).
- **Komponen:** `QuotaMonitorTable` (tabel + progress bar; band **kritis ≥90%** merah,
  mendekati batas kuning; ringkasan kritis/mendekati batas; empty/error state) dan
  `ModalityConsumptionChart` (bar bertumpuk chat/vision/voice/embedding).
- **Utilitas bersama:** `lib/quota-monitor.ts` (`quotaSeverity`, `summariseQuota`).
- **UI primitive:** `Progress` diberi prop `indicatorClassName` untuk warna band.
- **Sidebar:** menu "Pemantauan Kuota" ditambahkan.

### Verifikasi
- `pnpm build` hijau (route `/super-admin/quota-monitor` ter-generate, 32/32 halaman).
- `pnpm test` **193/193**; `pnpm lint` 0 error pada berkas baru.
- Test baru: `QuotaMonitorTable.dom.test.tsx` (6, termasuk axe a11y) + `lib/quota-monitor.test.ts` (4).
- Live: halaman 200; `?threshold=90` 200; data 33 baris (threshold=1); menu sidebar tampil.
- Backend `go vet`/`go test ./internal/... ./pkg/...` tetap hijau (tidak ada perubahan backend).

## 2026-10-14 — Audit Kuota/Biaya: 2 Bug Kritis Ditemukan & Diperbaiki

**Konteks:** Permintaan user untuk (a) audit semua query ledger tanpa filter `entry_type`,
(b) bandingkan scan di halaman Transaksi vs AI Chat, (c) verifikasi RBAC plan, (d) analisis
interaksi kuota OCR vs token, (e) dinamisasi biaya per-10k + markup.

### 🔴 BUG-OCR-01 — Double count kuota OCR (DIPERBAIKI)
`CountVisionScansInPeriod` menghitung **semua** baris modality=vision (reserve + commit),
sehingga 1 scan = 2 hitungan. Bukti live: 8 scan → tercatat 16 (user Pro hanya bisa ~50 scan,
bukan 100). Fix: filter `entry_type='reserve'`. Regression test ditambahkan.

### 🔴 BUG-NEG-01 — Kuota token bisa NEGATIF (DIPERBAIKI)
Dua lapis:
1. `/ai/receipt` **tidak** ter-gate `ai_tokens_per_cycle` (padahal OCR mengonsumsi token vision).
2. `Commit` tidak meng-clamp; reserve=estimasi 1 token, commit=token asli (~4.600).
Bukti live: kuota 150.000, sisa 100 → 1 OCR → `used_tokens = 154.474` (**−4.474**).
Fix: (a) tambah gate token untuk OCR; (b) `Commit` clamp ke `quota − used` + log peringatan.
Bukti: token habis + OCR → **402** (bukan 200); `used` tetap 150.000 (tidak negatif).

### ✅ Audit menyeluruh (16 lokasi query ledger)
Hanya `CountVisionScansInPeriod` yang bermasalah; 15 lainnya sudah benar memfilter
`entry_type` (`commit` untuk biaya/token, `reserve` untuk hitungan request).

### ✅ Verifikasi
- **Scan Transaksi vs AI Chat:** keduanya memanggil **endpoint sama** `POST /ai/receipt`
  (`ReceiptScanner`, `ReceiptUpload`, `InlineAiAssistant`) → ledger & RBAC identik.
  Live: 2 scan dari 2 jalur → ocr +2 (tepat), token & request ikut.
- **RBAC/entitlement:** user Free → `GET /reports` 403, `POST /ai/receipt` 403;
  user biasa ke `/admin/*` 403; anon 401. Semua rute AI ter-gate sesuai plan.

### 📄 Dokumentasi
Folder baru `second_brain/0_Inbox/ide/2026-10-14_Audit-Kuota-Biaya-Dinamis/`:
- `00-audit-kuota-biaya.md` (temuan lengkap + bukti + matriks gate)
- `01-implementation-testing-plan.md` (P0 selesai; P1–P3 rencana dengan checklist)

### Rekomendasi lanjutan
- P1: gate `ai_tokens_per_cycle` untuk `/ai/voice` (konsistensi).
- P2: satuan harga **per 10k** + harga beli provider + tampilan **margin** (permintaan user).
- P3: estimasi reserve lebih akurat (kurangi over-run clamp) + panel margin di Super Admin.

### Verifikasi teknis
`go vet`/`go build` hijau; `go test ./internal/... ./pkg/...` hijau; integrasi
`./internal/repository/postgres/` hijau (termasuk 2 test regression baru).
## 2026-10-14 — Perbaikan Export CSV, Penutupan Kebocoran Fitur RAG, & Audit Entitlement
- **Export CSV/PDF Transaksi & Laporan**:
  - `frontend/app/(dashboard)/user/transactions/page.tsx`: Tombol `Export CSV` sebelumnya memiliki handler mock `toast.info("Ekspor laporan belum tersedia...")`. Diperbaiki dengan memanggil `GET /api/v1/reports/export?format=csv` dan mengunduh berkas CSV asli dari server.
  - `frontend/app/(dashboard)/user/reports/page.tsx`: Tombol `Export PDF` dihubungkan ke `GET /api/v1/reports/export?format=pdf` untuk mengunduh laporan PDF.
- **Penutupan Kebocoran Fitur (Feature Leak) RAG**:
  - Ditemukan kebocoran fitur pada `backend/internal/usecase/ai_usecase.go`: fungsi `retrieveRelevantTransactions` menjalankan pencarian vektor semantik untuk semua user chat tanpa memverifikasi switch `rag_search`. User `Free` mendapatkan pencarian cerdas secara gratis meskipun plan menonaktifkannya.
  - Diperbaiki dengan menginjeksi `entitlement.Resolver` ke `AIUsecase` (`WithEntitlements`) dan memverifikasi `res.Feature("rag_search")` sebelum melakukan vector search. Ditambahkan unit test regresi `TestRetrieveRelevantTransactions_GatedByFeature`.
- **Verifikasi**:
  - `go test ./internal/...`: 27 packages ok.
  - `pnpm test`: 234 passed (31 test files).
  - `pnpm build`: Berhasil tanpa error (`32/32` pages).

## 2026-10-15 — Migrasi Sudut Lancip → Border Radius (Revisi Design System)
- **Konteks**: User melaporkan semua border (tabel, card, input, dll) lancip. Investigasi menemukan itu memang keputusan neo-brutalist yang dikunci (`--radius-*: 0px` di `app/globals.css`), TAPI 23 file memakai `rounded-md/lg/xl` sporadis → UI tidak konsisten.
- **Keputusan**: User memilih mengaktifkan border radius di seluruh UI (dicatat sebagai **ADR-011**).
- **Perubahan**:
  - `app/globals.css`: token `--radius-sm..4xl` = 4/6/8/12/16/24/32px, `--radius` = 6px; class `.neo-*` ikut dibulatkan.
  - `components/ui/*`: tambah radius ke button, input, badge, card, dialog, select, accordion, tabs, skeleton, table, sheet (dropdown/sidebar/tooltip sudah punya).
  - 93 file konsumen: **403 sisipan** `rounded-*` (342 `md`, 58 `lg`, 4 `sm`) hanya pada border kotak; border pembatas (`border-b-2`/`border-t-2`/`divide-*`) sengaja dilewati.
  - Bersih-bersih: 4 `rounded-none` dihapus, `rounded` telanjang → `rounded-md`, sisipan `rounded-none` redundan dihapus.
- **Verifikasi**: `pnpm test` → 271 passed (36 files); `pnpm build` → Compiled successfully. `pnpm lint` & `tsc` tetap sama dengan baseline (error pre-existing di file `*.test.tsx` & `useCallback` deps, bukan regresi).
- **Catatan**: shadow keras `Npx Npx 0` (ciri brutalist) dipertahankan.

## 2026-10-15 — Navigasi Bawah Mobile + Perbaikan Responsif AI Chat
- **Konteks**: User melaporkan (1) halaman `/user/ai` jelek saat responsif (header chat meluber, teks terpotong), (2) daftar sesi chat langsung terbuka penuh di mobile sehingga tampak "double sidebar" bersama Sheet navigasi global, (3) ingin pola app mobile: semua navigasi di bawah, bukan hamburger/sidebar.
- **Keputusan** (dikonfirmasi user): FAB AI Chat di kanan bawah (FAB klasik), hamburger/Sheet global **dihapus total** di mobile, daftar sesi chat tetap overlay penuh tapi **default tertutup**.
- **Perubahan**:
  - `components/layout/MobileBottomNav.tsx` (baru): bar tetap 5 item (`md:hidden`) + FAB AI Chat melayang; `pb-[env(safe-area-inset-bottom)]`, target sentuh `h-16`, `aria-current` per item, FAB disembunyikan saat sudah di route chat.
  - `lib/nav.ts`: `MOBILE_FAB_HREF` + `MOBILE_BOTTOM_NAV_ITEMS` (6 item dipecah 5 bar + 1 FAB).
  - `app/layout.tsx`: `export const viewport` dengan `viewportFit: "cover"` agar bar mengisi area home-indicator.
  - `app/(dashboard)/user/layout.tsx`: render `<MobileBottomNav />`; `main` diberi `pb-[calc(5rem+env(safe-area-inset-bottom))]`.
  - `components/layout/Header.tsx`: hapus `Sheet`/hamburger mobile + import mati; navigasi mobile kini sepenuhnya dari bottom bar.
  - `components/ai/ChatWindow.tsx`: `min-w-0` pada panel kanan & container scroll (fix overflow horizontal), header kompak (judul & label mode jadi ikon-saja di `<sm`), empty state `w-full`+`text-pretty`; state sidebar dipisah `desktopSidebarOpen`/`mobileSidebarOpen` (mobile default tertutup, tanpa flash overlay); rail sesi desktop di-gate CSS (`hidden md:flex`) agar tidak flash di mobile saat SSR.
- **Verifikasi**: `pnpm test` → 302 passed (39 files; 1 kegagalan awal di `lib/chat-sanitize.test.ts` **untracked** yang flaky antar-urutan, hijau saat dijalankan ulang & lulus saat isolasi); `pnpm build` → Compiled successfully; `tsc`/`eslint` tetap baseline. Cek browser 360/390/414/442px: tanpa scroll horizontal, bar 66px, FAB aktif, overlay sesi opt-in. Desktop 1568px tidak berubah (sidebar 240px, kartu chat 1280px, bar `display:none`).

## 2026-10-15 — AI Chat: Kuota Token, Kebocoran Payload, Sesi Ganda, & Layout
- **Laporan user**: chat sekali di akun Free langsung habis 5k token & tidak bisa lanjut; 1 chat jadi 2 sesi; response menampilkan JSON/toolcalls mentah; pesan `limit_exceeded` bocor; sidebar chat terpotong.
- **Temuan & perbaikan** (dokumen: `second_brain/0_Inbox/ide/2026-10-15_AI-Chat-Kuota-Render-Sesi/00-audit-ai-chat.md`):
  - **AI-TOK-01 (kritis)** — Reserve token **flat 2000** (`billing/estimate.go`) padahal 1 chat nyata ±5.000–10.000 token (system prompt + tools + RAG + riwayat 20 pesan); commit meng-clamp ke sisa kuota → 1 chat menguras seluruh kuota Free 5.000. **Fix**: estimasi proporsional `EstimateStore.ForChat(ctx, modality, promptChars)` (overhead 3.000 + prompt/4 + headroom 800, `ai_estimate_chat` tetap jadi lantai) + test.
  - **AI-TOK-02** — Commit ter-clamp senyap. **Fix (UI)**: badge sisa token + banner "kuota habis" di ChatWindow (dari `GET /me/entitlements`).
  - **AI-LEAK-01** — Model meng-echo payload internal (`{"chart":…}`, ```json) → tampil mentah. **Fix**: modul baru `frontend/lib/chat-sanitize.ts` + 8 unit test; dipakai di ChatMessage (assistant saja).
  - **AI-ERR-01** — `limit_exceeded` (402) tidak ditangani UI → bubble "…limit_exceeded". **Fix**: semua kode gate ke `notifyEntitlementError`; pesan kini spesifik per limit + angka pemakaian; `InlineAiAssistant` yang `catch {}` diberi toast.
  - **AI-SES-01** — Dua jalur pembuatan sesi (handleSend + efek `?q=` dgn `setTimeout 300ms`) tanpa guard → 2 sesi. **Fix**: `ensureSession()` dengan in-flight guard + hapus `setTimeout` + early-return `handleNewSession`.
  - **AI-UI-01** — Sidebar terpotong & scroll horizontal (`min-width:auto` + `truncate`). **Fix**: `min-w-0` + `overflow-x-hidden` + rail `w-60→w-64`.
- **Verifikasi**: `go vet`/`go build` bersih; `go test ./internal/billing/... ./internal/delivery/... ./internal/usecase/...` ok; `pnpm test` **289 passed (38 files)** naik dari 271; `pnpm lint` sama dengan baseline (tanpa regresi); `pnpm build` sukses.
- **Rekomendasi belum dikerjakan**: naikkan kuota Free / kuota chat terpisah; event `quota_clamped`; idempotency key `POST /ai/sessions`; satukan kode "token habis" (402 vs 429).

## 2026-10-15 — Dokumen Desain: Dinamisasi Kuota, Matriks Fitur, & Rasio Token per Plan
- **Permintaan user**: (1) kuota + matriks fitur per plan dinamis penuh dari Super Admin tanpa hardcode; (2) kalkulasi biaya/pemakaian token di Super Admin, dengan ide **rasio konversi** (mis. "per 5k token = 1k pemakaian plan") agar kuota tidak terkesan boros.
- **Keputusan user**: mekanisme = rasio token (biaya tetap dari token nyata); cakupan = per-plan; tampilan user = unit kuota saja; pengerjaan = **dokumen desain dulu**.
- **Hasil**: folder baru `second_brain/0_Inbox/ide/2026-10-15_Dinamisasi-Kuota-Rasio-Token/` berisi 10 file:
  `README.md`, `00-ringkasan-eksekutif.md`, `01-desain-teknis.md`, `02-skema-database-dan-migrasi.md`, `03-spesifikasi-api.md`, `04-desain-ui.md`, `05-model-biaya-dan-margin.md`, `06-rencana-implementasi.md`, `07-rencana-testing.md`, `08-risiko-dan-rollback.md`.
- **Temuan audit penting**:
  - Sudah ada fondasi dinamis: tabel `plan_limits`, `plan_entitlements`, `pricing_rules`, katalog entitlement, editor plan. Yang kurang: **rasio token** + editor berbasis dropdown katalog.
  - `token_ledger` punya trigger **append-only** (`token_ledger_no_update`) → backfill `raw_tokens` harus tanpa UPDATE (Pilihan B: helper `RawTokensOrAmount()`).
  - `pricing_rules.provider_cost_per_1k_idr` = **0** di rule `default` → margin belum terukur (kalkulator akan menampilkan 100%, menyesatkan).
  - `database/dump/fintrack_v2.sql` **belum memuat** kolom migrasi 0018 (`provider_cost_per_1k_idr`, `price_unit`) → wajib regenerate.
  - Dengan asumsi biaya provider Rp 150/1k token, plan **Pro & Business berpotensi rugi** bila user memakai 100% kuota → perlu evaluasi harga/kuota.
- **Angka kunci** (terverifikasi dengan rumus `PricingEngine` nyata): chat 9.945 token → jual Rp 3.729, provider Rp 1.491, margin Rp 2.238 (60%). Rasio 5 → 1.989 unit/chat → kuota Free 5.000 memberi ~2 chat (sebelumnya 0).
- **Status**: 📋 Proposal. Belum ada kode/skema yang diubah. Menunggu persetujuan 11 keputusan (P-1…P-11) sebelum implementasi.

## 2026-10-15 — Revisi Dokumen: Semua Kebijakan Jadi Dinamis (Tanpa Hardcode)
- **Permintaan user (revisi)**: *"saya ingin semuanya itu dinamis tanpa ada yang hardcode atau ditulis pada kode. semua keputusan, logic, flow, dan kalkulasi itu dari super admin dan dinamis. metrix juga harus dinamis."*
- **Jawaban keputusan user**: P-1 ceil; P-2 dinamis per plan; P-3 "kuota"; P-4 tetap dibagi; P-5 default 1; P-6 Pilihan B; P-7 NOT VALID >10jt; P-8 serahkan operator; P-9 default 10k dinamis; P-10 dinamis; P-11 default dinamis.
- **Audit hardcode baru** (temuan penting):
  - `admin/settings.go` → `writableSettings` (allowlist + validator) = **hardcode**.
  - `middleware/usage_resolvers.go` → `UsageFor()` switch limit_key→resolver = **hardcode**.
  - `entitlement/catalog.go` → `FeatureCatalog`/`LimitCatalog` = **hardcode**.
  - `billing/ledger.go` → mode pembulatan tetap; `billing/estimate.go` → overhead/chars konstan; `cmd/server/main.go` → sentinel `1e9`.
- **Solusi desain (dokumen baru `09-lapisan-konfigurasi-dinamis.md`)**: 3 tabel katalog jadi **data**, bukan kode:
  - `setting_definitions` (kind/min/max/enum/default/label/group) → ganti `writableSettings`.
  - `metric_definitions` (label/unit/resolver_key/enforced/ratio_adjusted) → ganti `LimitCatalog` + switch `UsageFor`.
  - `feature_definitions` → ganti `FeatureCatalog`.
  - Setting baru: `quota_rounding_mode`, `quota_default_ratio`, `quota_default_tokens_per_cycle`, `quota_unit_label`, `quota_unlimited_sentinel`, `quota_min_charge_units`, `ai_estimate_overhead_tokens`, `ai_estimate_output_headroom`, `ai_estimate_chars_per_token`, `pricing_default_provider_cost_per_1k_idr`.
  - **Seam diakui**: resolver usage (kemampuan), struktur rute, config bootstrap tetap di kode; kebijakan di DB.
  - **Fallback konstanta permanen** (P-12) → tabel kosong tidak mematikan sistem.
- **Dokumen diperbarui**: README (index + keputusan P-1..P-16 + tabel "sudah ada vs hardcode"), 00 (jawaban keputusan + prinsip no-hardcode), 01 (`ceilDiv`→`RoundUnits` dinamis + §12 lapisan konfigurasi), 02 (migrasi 0026 tabel katalog, seed rasio ditiadakan, backfill dihapus sesuai P-6), 03 (endpoint katalog + resolvers), 04 (editor settings/metrik/fitur + alur operator tanpa deploy), 05 (default biaya provider dinamis), 06 (fase F4 baru + renumber + DoD), 07 (test konfigurasi dinamis §8b), 08 (risiko R-11..R-16 + fail-safe + safety nets).
- **Verifikasi dokumen**: 11 file, semua code fence seimbang, tidak ada link internal rusak.
- **Status**: 📋 Proposal revisi. Belum ada kode/skema diubah. Estimasi naik 6–10 → **8–13 hari kerja**.

## 2026-10-15 — Revisi Dokumen: Invalidasi Cache Event-Driven (SSE) Gantikan TTL
- **Keputusan user**: P-12 fallback; **P-13 pakai SSE (bukan TTL)** — "biar lebih terjamin, murah, dan tidak memberatkan server/resources"; P-14 seam diakui; P-15 ada (default 1); P-16 soft delete.
- **Temuan penting**: infrastruktur SSE + Redis pub/sub + cache ber-versi **sudah lengkap** (dibangun untuk P07):
  - `realtime/hub.go` (SSE hub, heartbeat, resume `Last-Event-ID`, batas koneksi per user)
  - `realtime/publisher.go` (fan-out Redis lintas-instance + fallback hub lokal)
  - `realtime/buffer.go` (replay event terlewat)
  - `infrastructure/redis/cache.go` (`BumpVersion(scope)` = invalidasi O(1) lintas-instance)
  - `frontend/lib/realtime-dispatcher.ts` (event → invalidasi query TanStack, sudah pola `plan_updated`)
- **Desain P-13** (bagian baru `09 §12`): **cache ber-versi + invalidasi event-driven**, bukan TTL.
  - Alur: simpan → `BumpVersion("cfg")` + publish `config_changed` → Redis fan-out → semua instance drop cache lokal + klien muat ulang segera.
  - Event baru `config_changed` dengan payload `{scope, keys}` (scope: settings|metrics|features|plans|pricing).
  - Endpoint baru `POST /admin/config/reload` untuk perubahan di luar API.
  - **Jaring berlapis** (bukan hanya SSE): versi cache (tanpa pub/sub tetap terbaca) + buffer replay + **TTL 5 menit** sebagai jaring terakhir (bukan mekanisme utama).
  - Alasan lebih baik dari TTL: kesegaran ≈ milidetik, beban DB 1× saat berubah (bukan berulang tiap TTL), konsisten lintas-instance, tidak bergantung traffic.
- **Dokumen diperbarui**: 09 (§12 baru + §13/14/15/16 renumber + cache ber-versi), 00 (P-13), 01 (referensi cache), 03 (§5.0.3b event + §5.0.3c reload), 04 (U-8 + catatan kesegaran), 06 (tugas 4.3/4.4b), 07 (§8b.5 test invalidasi realtime + fail-safe SSE), 08 (R-17 + S-17/S-18 + P-13), README.
- **Verifikasi dokumen**: 11 file, code fence seimbang, tidak ada link rusak, tidak ada heading duplikat.
- **Status**: 📋 Proposal. Belum ada kode diubah.

## 2026-10-15 — Implementasi: Rasio Token per Plan + Konfigurasi Dinamis (ADR-012)
- **Keputusan user**: P-12 fallback permanen; **P-13 invalidasi event-driven (SSE + Redis)**; P-14 seam diakui; P-15 lantai tagih; P-16 soft delete.
- **Migrasi** (dijalankan + diuji idempoten pada DB):
  - `0028_plan_token_quota_ratio.sql`: `plans.token_quota_ratio`, `token_periods.quota_ratio`, `token_ledger.raw_tokens` (+ CHECK). Tanpa backfill (append-only, P-6).
  - `0029_config_catalog.sql`: 3 tabel katalog + seed (60 setting, 6 metrik, 8 fitur, 4 secret).
  - **Bentrok nomor ditemukan**: fitur voice (uncommitted) sudah memakai `0025`; migrasi saya di-renumber `0025→0028`, `0026→0029`.
- **Backend**:
  - `billing/rounding.go`: `RoundUnits` (ceil/floor/round dinamis) + `ApplyMinCharge` + 8 unit test.
  - `billing/ledger.go`: `Reserve`/`Commit`/`Refund` menerima **RAW**, konversi ke unit via rasio snapshot; `cost_idr` dari raw; `raw_tokens` tercatat. `QuotaResolver` mengembalikan rasio.
  - `billing/quotaconfig.go`: `SettingsQuotaConfig` (mode pembulatan, lantai, sentinel dari setting).
  - `configcatalog/catalog.go`: **BARU** — cache ber-versi (invalidasi event-driven) untuk setting/metrik/fitur + `ResolverKeyFor`.
  - `middleware/usage_resolvers.go`: `UsageFor` data-driven (`resolverRegistry` + `metric_definitions`), fallback konstanta.
  - `admin/config_catalog.go`: **BARU** — endpoint katalog, `/admin/resolvers`, `/admin/config/reload`, `POST /admin/plans/estimate` (kalkulator).
  - `contracts/events.go` + `realtime/publisher.go`: event `config_changed`.
  - `entitlement`: `Resolved.QuotaRatio` + `UsageEntry.Unit` dari katalog.
  - Test: 7 test invarian ledger (fake repo, tanpa DB) + 4 test handler + 8 test rounding — **semua PASS**.
- **Frontend**:
  - `hooks/useConfigCatalog.ts` + `hooks/useQuotaEstimate.ts`: **BARU**.
  - Halaman `/super-admin/kuota` (kalkulator) + `/super-admin/metrics` (katalog metrik).
  - `QuotaSettingsTab` render setting grup "kuota" **dari katalog DB** (bukan hardcoded).
  - `PlanFormDialog`: input rasio + payload; `PlanFormDialog` dropdown katalog.
  - Dispatcher SSE `config_changed` + cache-keys baru; label user "Token AI" → "Kuota AI"; catatan satuan.
  - `lib/api-error.ts`: **BARU** (narrowing, tanpa inline cast).
- **Verifikasi**:
  - Backend: `go vet ./...` bersih; `go build ./...` bersih; `go test ./internal/...` **semua ok**.
  - Frontend: `pnpm test` **338 passed (43 files)** (naik dari 289); `npx tsc --noEmit` bersih untuk file baru.
  - `pnpm lint`: 10 error, **semuanya di file fitur voice yang belum di-commit sebelum sesi ini** (bukan pekerjaan ini) — dibuktikan file tidak memuat kata kunci fitur ini.
- **Tidak dapat diverifikasi**: test integrasi DB (`TEST_DATABASE_URL`) — Docker Desktop mati (`Docker Desktop is unable to start`), port 5435/6379 tertutup. Test unit ledger menggantikan pembuktian konversi.

## 2026-10-15 (lanjutan) — Verifikasi End-to-End dengan DB Hidup
- **Docker Desktop berhasil dinyalakan** → verifikasi lengkap dapat dijalankan (sebelumnya diblokir).
- **Koreksi konfigurasi test**: Redis FinTrack ada di port **6380** (6379 milik project lain/dagangbot) → NOAUTH. Test e2e memakai DB **`fintrack_test`** (bukan `fintrack`).
- **Bug nyata ditemukan & diperbaiki** (bukti nyata, bukan klaim):
  1. `EstimateStore.settingInt` memperlakukan nilai `0` sebagai "unset" → overhead estimasi 0 diabaikan, reserve selalu 3800. Ditemukan test e2e `TestP08_001_TokenQuotaConcurrency` (0 sukses dari 20). Diperbaiki: baca nilai mentah via `GetString` agar membedakan "absen" dari "0". Setelah fix: **5 sukses / 15 ditolak** (sesuai harapan).
  2. Migrasi `0028`/`0029` belum dijalankan di DB `fintrack_test` → 3 test e2e gagal (404 "paket tidak ditemukan"). Setelah migrasi dijalankan di sana: **semua e2e PASS**.
  3. Fixture test (`oneTokenSettings`, `oneTokenEstimateSettings`) disesuaikan untuk overhead dinamis.
- **Hasil verifikasi final**:
  - `go vet ./...` bersih; `go build ./...` bersih; `go test ./internal/...` semua ok.
  - `go test ./internal/repository/postgres/ -run TestLedger` → **11/11 PASS** (concurrency + regresi).
  - `go test ./tests/` (e2e) → **PASS**.
  - `pnpm test` → **338 passed (43 files)**; `pnpm build` → **36/36 halaman** (termasuk halaman baru).
  - `pg_dump` → dump di-regenerate (kini memuat kolom migrasi 0018 yang tadinya hilang + 33 referensi fitur baru). Tidak ada secret bocor.
- **Sisa kegagalan (di luar pekerjaan ini)**: `TestVoiceRepo_*` gagal karena tabel `voice_daily_usage` belum ada — bagian fitur voice yang belum selesai (uncommitted). `pnpm lint` 10 error juga semuanya di file voice tersebut.
- **Renumber migrasi**: `0025→0028`, `0026→0029` (hindari bentrok dengan `0025_voice_transcriptions`).

## 2026-10-15 (Audit Paska-Implementasi) — Perbaikan Temuan Backend & Desain
- **Desain UI**: Pertanyaan pengguna mengenai border radius ditinjau. Konsep desain resmi adalah **neo-brutalist** (`--radius: 0px`, border 2px, hard offset shadow `4px 4px 0 0`) yang disepakati sejak brief awal.
- **Perbaikan Bug Kritis Hasil Reviewer (`agent://BackendAudit`)**:
  1. **Double Conversion AI Reserve Fallback**: `ai_handler.go` (`Chat`, `settleVision`, `settleVoice`) saat provider tidak melaporkan usage sebelumnya memakai `reservation.Amount` (units), yang ketika dikirim ke `Commit()` dibagi rasio untuk kedua kalinya. Ditambahkan field `RawTokens` ke struct `Reservation`; fallback kini memakai `RawTokens` mentah.
  2. **Cross-Period Quota Leak/Corruption**: `Commit` & `Release` di `ledger.go` sebelumnya mencari periode aktif saat ini via `s.now()`. Jika periode berganti di antara reserve dan commit, `reserved_tokens` periode baru menjadi negatif dan periode lama bocor. Diperbaiki: `GetPeriodByIDForUpdate(reserve.PeriodID)` dan `AdjustPeriod` diberi pengaman `GREATEST(0, ...)`.
  3. **Pengaturan Quota Tak Bisa Disimpan**: `admin/settings.go` menolak update setting kuota/estimasi dengan error `setting_not_writable`. Diperbaiki: `writableSettings` dilengkapi key kuota + resolusi dinamis via `catalog.SettingDefinition`.
  4. **Invalidasi & Event Plan**: `UpdatePlan` dan `DeactivatePlan` di `admin/plans.go` kini memanggil `publishConfigChanged` dan `invalidatePlan`.
  5. **Invalidasi Subscription Scope Entitlement**: `admin/subscriptions.go` sebelumnya hanya mem-bump scope `"u:"` bukan `"ent:"`. Diperbaiki dengan memanggil `h.svc.entInvalidator.InvalidateUser(ctx, userID)`.
  6. **Fallback Resolver Catalog**: `usage_resolvers.go` `UsageFor` tidak lagi menimpa deaktivasi metrik dari katalog dengan default hardcoded.
  7. **Missing Metric Seed**: `voice_requests_per_day` ditambahkan ke `0029_config_catalog.sql` dan dieksekusi di database `fintrack` & `fintrack_test`.
  8. **Error Handling Commit/Release**: Penanganan error `Commit` dan `Release` di `ai_handler.go` kini dicatat via logger (bukan `_ =`).
  9. **Timeseries Token Reporting**: `usage_repo.go` `KindAIToken` menggunakan `GREATEST(raw_tokens, ABS(amount))` agar konsisten dengan `admin/metrics.go`.
- **Verifikasi**:
  - `go build ./...` & `go vet ./...` lolos 100%.
  - `go test -count=1 ./internal/billing/... ./internal/admin/... ./internal/delivery/http/handler/... ./internal/delivery/http/middleware/...` semua PASS.
  - Frontend: `pnpm test` **347 passed (44 files)**, `pnpm build` sukses seluruh halaman prerender & dynamic.

## 2026-10-15 (Audit Voice & Vision) — Analisis Kalkulasi, Logika, & Desain Dinamis Super Admin
- **Audit Logika Voice & Vision**:
  - Voice: Logika kuota harian (`voice_requests_per_day`) via atomic conditional increment `ConsumeDailyUsage` pada tabel terpisah `voice_daily_usage` sudah aman dari race condition dan bypass penghapusan riwayat.
  - Vision: Logika pengkategorian baru (`loadReceiptCategories`, `receiptPrompt`, `normalizeReceiptResult`, `snapCategory`) sudah akurat mencocokkan kategori DB live (exact match -> substring -> fallback) dan menangani refusal model.
  - Temuan kalkulasi token: Reservasi kuota untuk Voice dan Vision saat ini masih flat/konstan (`def.Voice = 2000`, `def.Vision = 5000`), tidak adaptif terhadap panjang audio maupun ukuran file gambar struk.
- **Solusi Kalkulasi Dinamis (ADR & Desain)**:
  - Voice: `ForVoice(audioBytes)` mengestimasi durasi audio dari byte (`bytes_per_sec`), dikalikan `tokens_per_sec` + overhead + output headroom. Audio 2 detik tidak lagi memotong 2000 token.
  - Vision: `ForVision(imageBytes, promptChars)` mengestimasi token dari panjang prompt kategori + base image tokens (scaled jika > 500KB) + output headroom.
  - Konfigurasi Super Admin: 8 kunci konfigurasi baru dirancang untuk `setting_definitions`, tabel `settings`, serta integrasi simulasi pada panel `/super-admin/kuota`.
- **Dokumen Temuan Lengkap**: Dicatat di `second_brain/0_Inbox/ide/2026-10-15_Voice-Input-Smart-Routing/08-audit-kalkulasi-voice-vision-dinamis.md`.

## 2026-10-15 (Implementasi Voice & Vision Dinamis) — Estimasi Token Adaptif & Kontrol Super Admin
- **Migrasi 0031 (`0031_voice_vision_dynamic_estimation.sql`)**:
  - Mendaftarkan 8 setting ke `setting_definitions` dan `settings` (`ai_estimate_voice_bytes_per_sec`, `ai_estimate_voice_tokens_per_sec`, `ai_estimate_voice_overhead`, `ai_estimate_voice_output_headroom`, `ai_estimate_vision_image_tokens`, `ai_estimate_vision_output_headroom`, `ai_estimate_vision_large_file_bytes`, `ai_estimate_vision_large_file_extra`).
  - Diterapkan ke DB `fintrack` dan `fintrack_test`.
- **Backend (`billing/estimate.go`, `ai_handler.go`, `metering/estimator.go`)**:
  - `EstimateStore`: `ForVoice(audioBytes)` dan `ForVision(imageBytes, promptChars)` menghitung token dinamis berdasarkan muatan dan parameter DB (bukan nilai konstan).
  - `AIHandler`: `visionEstimate` dan `voiceEstimate` menghubungkan payload riil (`header.Size` dan byte audio hasil dekode) ke `Reserve`.
  - `metering`: `EstimateAudioFromBytes` menghitung fallback lokal adaptif; `applyMeteringSettings` dihubungkan ke `runtimeInvalidator` sehingga perubahan setting di Super Admin langsung hot-reload tanpa restart server.
  - `admin/config_catalog.go`: `PlanEstimate` menghitung simulasi Voice dan Vision serta mengembalikan biaya dan unit per panggilan.
- **Frontend (`useQuotaEstimate.ts`, `kuota/page.tsx`, `QuotaSettingsTab.tsx`)**:
  - `QuotaSettingsTab`: grup `ai_estimate` dirender otomatis berdampingan dengan `kuota`.
  - `QuotaCalculatorPage`: kalkulator menampilkan simulasi Voice (input slider durasi detik) dan Vision (scan struk).
- **Database Dump**:
  - `database/scripts/dump.sh` dijalankan ulang; dump ter-update memuat migrasi 0031.
- **Verifikasi**:
  - Unit tests `billing` (8/8 PASS termasuk `TestForVoice_*` dan `TestForVision_*`).
  - Backend `go build ./...` & `go vet ./...` exit 0; handler/admin/usecase tests PASS.
  - Frontend: `pnpm test` 347 passed (44 files), `pnpm build` sukses seluruh 36 routes.

## 2026-10-15 (Audit Bug Lanjutan & Perbaikan) — 6 Bug Nyata Ditemukan & Diperbaiki
- **Metode**: audit reviewer (`agent://DeepCodeAudit`) + audit manual, dengan pembuktian reproduksi (bukan klaim).
- **Bug #1 — PANIC nil-map (`usecase/ai_gateway_vision_voice.go`)**:
  - `json.Unmarshal("null", &map)` mengembalikan map **nil TANPA error**; penulisan `result["type"]=...` berikutnya memicu **panic: assignment to entry in nil map** → worker chat/vision crash.
  - Dibuktikan dengan repro `go run`. Diperbaiki: guard `if result == nil` → fallback shape aman. Regression test `TestParseReceiptResponse_JSONNullDoesNotPanic`.
- **Bug #2 — SetConfig menjatuhkan `AudioBytesPerSecond` (`aigateway/metering/estimator.go`)**:
  - `New`/`SetConfig` tidak menyalin field `AudioBytesPerSecond`, sehingga perubahan Super Admin atas `ai_estimate_voice_bytes_per_sec` **tidak pernah diterapkan** (selalu 4000). Regression test `TestSetConfig_AppliesAudioBytesPerSecond`.
- **Bug #3 — Gate hot-reload tidak mengenali key baru (`admin/settings.go`)**:
  - `hasMeteringKey` hanya memuat key `metering_*`; perubahan `ai_estimate_voice_*`/`ai_estimate_vision_*` **tidak memicu** `meteringReload`. Diperbaiki dengan menambahkan key terkait.
- **Bug #4 — Fitur dinamis inert karena floor flat (`billing/estimate.go` + migrasi 0031)**:
  - `ForVoice`/`ForVision` awalnya mengabaikan `ai_estimate_voice`/`ai_estimate_vision` sebagai floor (beda dari `ForChat`), sehingga kontrak "operator bisa menaikkan lantai" hilang.
  - Sebaliknya, seed lama (voice=2000, vision=5000) + `DefaultQuotaEstimate` (2000/5000) akan **mendominasi** kalkulasi dinamis → fitur praktis mati. Diperbaiki: floor dihormati + default diturunkan (Voice 200→, Vision 500→) di kode & migrasi 0031.
- **Bug #5 — Alokasi ganda base64 (`handler/ai_handler.go`)**:
  - `ProcessVoice` men-decode base64 hanya untuk mengukur panjang (5 MB → alokasi kedua penuh) padahal usecase men-decode lagi. Diganti hitung aritmetika `len*3/4`.
- **Bug #6 — Parameter mati & potensi crash UI**:
  - `VisionScans` tak terpakai di backend & `vision_scans` di frontend dihapus.
  - Guard `data.modalities?.voice && data.modalities?.vision` mencegah crash saat payload parsial/ter-cache.
- **Verifikasi**:
  - `go build ./...`, `go vet ./...` bersih.
  - `go test -count=1 ./internal/...` **semua ok** (30 paket); `go test ./tests/` (e2e) ok.
  - Frontend: `pnpm test` 347 passed (44 files); `pnpm build` sukses 36 route.
  - Dump DB di-regenerate (memuat setting baru).

## 2026-10-15 (UX Discoverability) — Deep-link Tab Pengaturan + Item Sidebar "Estimasi Token AI"
- **Masalah**: user tidak menemukan konfigurasi estimasi token Voice/Vision karena tersembunyi di dalam tab "Kuota" pada halaman Pengaturan Sistem; tidak ada entri menu langsung.
- **Perbaikan**:
  - `hooks/use-hash.ts`: hook `useHash()` baru (useSyncExternalStore) untuk deep-link berbasis hash.
  - `super-admin/settings/page.tsx`: Tabs kini **controlled** dan tersinkron hash — `/super-admin/settings#kuota` langsung membuka tab Kuota; ganti tab menulis hash ke URL (shareable + back button).
  - `SuperAdminSidebar.tsx`: entri baru **"Estimasi Token AI"** (`/super-admin/settings#kuota`, ikon Gauge) + deteksi aktif hash-aware (dua entri yang berbagi pathname tidak menyala bersamaan).
  - Klik "Pengaturan" (tanpa hash) mereset ke tab Global.
- **Verifikasi**:
  - `npx tsc --noEmit` bersih untuk file terkait.
  - Test baru `hooks/use-hash.dom.test.tsx` (3 test) PASS.
  - Frontend `pnpm test` **352 passed (45 files)**; `pnpm build` sukses.

## 2026-10-15 (Bug NaN Pagination + Sidebar Bergrup) — 2 Perbaikan UI Super Admin
- **Bug #1 — `NaN` pada pagination (`Received NaN for the children attribute`)**:
  - **Akar masalah**: helper `response.Paginated` mengirim `meta` **tanpa `total_pages`**; `DataTablePagination` menghitung `Math.max(undefined,1)` → `NaN` → `<Button>{NaN}</Button>`. Dibuktikan lewat API live (`meta: {total,page,per_page}` tanpa `total_pages`).
  - **Fix backend**: `pkg/response/response.go` — `Meta` kini punya `TotalPages`, dihitung di `Paginated` (`ceil(total/perPage)`, minimal 1). Berlaku untuk SEMUA endpoint paginated.
  - **Fix frontend**: `DataTablePagination` meng-coerce `page`/`perPage`/`total`/`totalPages` ke angka aman (fallback derive dari `total/perPage`) sehingga payload lama pun tidak merender NaN.
  - **Bukti**: API live kini mengembalikan `total_pages` di `subscriptions` (1), `audit-logs` (26), `payments` (6), `tokens/users` (36), `tokens/ledger` (121).
  - **Test**: `pkg/response` (`TestPaginated`, `TestPaginated_TotalPagesFloorsAtOne`), `data-table-pagination.dom.test.tsx` (4 test: undefined/NaN/valid/klik).
- **Bug #2 — Sidebar Super Admin tidak terkelola**:
  - 15 item dalam satu daftar datar → sulit dipindai.
  - **Fix**: dikelompokkan jadi 4 grup collapsible (**Ringkasan, Langganan & Pembayaran, Token & AI, Sistem**), **default terbuka semua**, toggle via `Record<string,boolean>` (state lokal). Titik indikator oranye menandai grup yang memuat halaman aktif.
- **Verifikasi**:
  - Backend `go build ./...`, `go vet ./...` bersih; `go test -count=1 ./...` **semua ok**; `go test ./tests/` (e2e) ok.
  - Frontend `pnpm test` **356 passed (46 files)**; `pnpm build` sukses.
