# 02 — Kritik & Saran (Best Practice Industri)

Dokumen ini adalah kritik jujur terhadap keinginan pemilik produk, disertai rujukan praktik industri
global dan pandangan para ahli. Tujuannya bukan menolak ide, melainkan membuatnya **aman, benar, dan
dapat dipertahankan** saat sudah berjalan di produksi dengan uang nyata.

---

## Kritik 1 — Uang tidak boleh memakai float

**Masalah:** jika harga, invoice, atau saldo disimpan sebagai `float`/`double`, akan muncul galat
pembulatan (mis. `0.1 + 0.2 = 0.30000000000000004`) yang merusak invoice dan pembukuan.

**Saran:** simpan semua nilai uang sebagai **integer minor unit** (rupiah tanpa desimal untuk IDR →
`15000` = Rp15.000). Token juga integer. Konversi ke tampilan hanya di layer presentasi.

**Rujukan:** IEEE-754 tidak merepresentasikan desimal uang secara eksak; dokumentasi API Stripe secara
eksplisit menyarankan integer untuk amount. Ini standar industri pembayaran.

---

## Kritik 2 — Metering token harus transaksional (reserve → call → commit)

**Masalah:** pola naif "cek kuota → panggil AI → kurangi kuota" **bocor** saat concurrency. Dengan 20
request paralel dan kuota sisa 5, ke-20 request bisa lolos karena semuanya membaca sisa 5.

**Saran:** pola **reserve → commit → release** dengan transaksi DB + **idempotency key**:
- Reserve menaikkan `reserved` (kunci sementara) → request ditolak **sebelum** biaya keluar.
- Commit menagih jumlah nyata (dari `usage` provider); Release mengembalikan reserve saat gagal.
- Idempotency key mencegah retry menagih dua kali.

**Rujukan:** pola reserve/commit adalah dasar sistem billing (mis. Lago, Stripe usage records) dan
mirip dengan SPK queue + advisory lock yang sudah dipakai FinTrack v1 (pelajaran internal).

---

## Kritik 3 — Kunci AI terpusat memperbesar blast radius

**Masalah:** memusatkan kunci di super admin berarti satu kebocoran/gangguan = seluruh layanan AI
terdampak. Ini adalah trade-off nyata, bukan gratis.

**Saran:** wajib ada (a) **pool multi-endpoint** dengan failover, (b) **circuit breaker** + cooldown,
(c) **kuota & rate limit per kunci**, (d) **kill switch**, (e) **health check** berkala, (f) rotasi
kunci terjadwal, (g) audit akses kunci. Jangan pernah bergantung pada satu kunci/endpoint.

**Rujukan:** pola *Circuit Breaker* (Nygard, *Release It!*) dan *bulkhead* untuk isolasi kegagalan.

---

## Kritik 4 — "Realtime tanpa reload" butuh kualifikasi

**Masalah:** tidak semua realtime bisa "instan tanpa request". Untuk aksi **user sendiri**, update cache
lokal sudah cukup (instan). Untuk perubahan dari **luar** (admin ubah plan, webhook bayar), klien tidak
punya cara tahu tanpa push atau polling.

**Saran:** pisahkan dua kasus:
- Aksi sendiri → **surgical cache update** (TanStack Query `setQueryData`/optimistic).
- Perubahan eksternal → **SSE push** (auth cookie, resume via `Last-Event-ID`, heartbeat, reconnect).

Jangan pernah menjadikan klien sebagai otoritas kebenaran.

**Rujukan:** prinsip *server-authoritative state*; pola optimistic update React Query (TkDodo).

---

## Kritik 5 — Cache invalidation itu sulit

**Masalah:** "caching realtime" yang salah invalidasi = user melihat data basi atau, lebih buruk, **bisa
memakai fitur yang belum dibayar** (cache entitlement basi).

**Saran:** kunci cache harus **granular per-entitas** dan **berversi**. Data billing/entitlement
memakai TTL pendek + bump versi pada mutasi, bukan cache panjang. Backend cache harus **bersama**
(Redis), bukan in-memory per-proses, agar konsisten lintas instance.

**Rujukan:** "There are only two hard things in Computer Science" (attributed to Phil Karlton); pola
cache-aside + versioned keys.

---

## Kritik 6 — Downgrade tidak boleh menghapus data

**Masalah:** jika kuota plan baru lebih kecil dari pemakaian, menghapus transaksi/kategori user adalah
malapraktik (data finansial user suci dan bisa jadi kewajiban hukum).

**Saran:** **soft-limit** — blokir fitur baru/kelebihan, jangan hapus data. Beri jalan ekspor.

**Rujukan:** prinsip *data minimization ≠ data destruction*; GDPR/UU PDP mengatur penghapusan hanya
atas permintaan subjek data, bukan akibat downgrade.

---

## Kritik 7 — Jangan percaya callback pembayaran dari klien

**Masalah:** mengandalkan halaman "sukses" di browser untuk mengaktifkan langganan bisa dimanipulasi.

**Saran:** sumber kebenaran pembayaran = **webhook server-to-server** yang (a) diverifikasi
**signature**, (b) diverifikasi ulang via **Status API**, (c) **idempoten** (replay-safe). Halaman sukses
hanya UI.

**Rujukan:** Midtrans mengirim notifikasi dengan `order_id + status_code + gross_amount + server_key`
untuk signature; verifikasi ulang via API adalah rekomendasi resmi Midtrans. Ini juga praktik
anti-fraud dasar (OWASP).

---

## Kritik 8 — Data kartu jangan menyentuh server kita

**Masalah:** menyimpan/memproses data kartu menuntut kepatuhan PCI-DSS penuh — mahal dan berisiko.

**Saran:** pakai **Midtrans Snap / halaman hosted** (setara **PCI-DSS SAQ-A**). Simpan hanya
`order_id`/token, bukan PAN/CVV.

**Rujukan:** PCI-DSS SAQ-A berlaku bila data kartu sepenuhnya ditangani pihak ketiga.

---

## Kritik 9 — Siklus per-hari & prorasi butuh konvensi eksplisit

**Masalah:** "prorata" dan "per-hari" ambigu (hari kalender? 24 jam? pembulatan ke atas/bawah? zona
waktu?).

**Saran:** tetapkan di spec: **hari kalender**, TZ **Asia/Jakarta**, `current_period_start + N hari`,
prorata = `(harga_baru − harga_lama) × sisa_hari / total_hari` dibulatkan **ke bawah** ke rupiah. Tulis
eksplisit agar tidak jadi sengketa.

**Rujukan:** praktik billing SaaS (Stripe proration) selalu mendefinisikan konvensi pembulatan secara
eksplisit.

---

## Kritik 10 — Aksi super admin wajib auditable

**Masalah:** tanpa audit, sengketa billing ("saya tidak pernah ubah plan ini") tak bisa diselesaikan,
dan penyalahgunaan tidak terdeteksi.

**Saran:** `audit_logs` **append-only** mencatat aktor, aksi, entitas, nilai **sebelum→sesudah**, IP, UA.
Berlaku untuk semua aksi billing, plan, entitlement, kunci AI, dan approval pembayaran manual.

**Rujukan:** prinsip *non-repudiation* & *least privilege* (NIST); SOX/GDPR mendorong jejak audit.

---

## Kritik 11 — Kepatuhan Indonesia (UU PDP)

**Masalah:** data billing (nama, email, riwayat pembayaran) = **data pribadi**. Ada kewajiban hukum.

**Saran:** tetapkan dasar hukum pemrosesan, kebijakan retensi, hak akses/penghapusan, dan
keamanan. Ada **tegangan** antara retensi invoice (pajak, umumnya 10 tahun) dan hak hapus PDP —
selesaikan dengan kebijakan tertulis (mis. anonymize, bukan hapus, saat retensi konflik).

**Rujukan:** UU No. 27/2022 tentang Pelindungan Data Pribadi.

---

## Kritik 12 — Zero downtime bukan slogan

**Masalah:** "tanpa down" tidak terwujud hanya karena ingin. Migrasi yang mengubah kolom bisa memutus
sesi yang berjalan.

**Saran:** disiplin **expand → migrate → contract** (Parallel Change), rolling deploy, **feature flag**,
dan health gate. Kolom baru = nullable dulu; hapus kolom lama di migrasi terpisah setelah kode lama mati.

**Rujukan:** *Parallel Change* / *Strangler Fig* (Martin Fowler).

---

## Kritik 13 — Observability wajib

**Masalah:** tanpa metrik, insiden (gateway down, pembayaran gagal, token membengkak) tidak terdeteksi.

**Saran:** metrik minimal: token usage per user/plan, error rate & latency gateway, failover count,
payment success rate, dunning rate, entitlement cache hit/miss. Plus alerting ambang batas.

**Rujukan:** praktik SRE (*four golden signals*: latency, traffic, errors, saturation).

---

## Kritik 14 — YAGNI: jangan bangun yang belum perlu

**Masalah:** godaan menambah rollover token, plan keluarga, PPN, multi-mata-uang, auto-recurring paksa,
pitch deck, dsb. di fase awal.

**Saran:** tunda semuanya. Fokus ke jalur inti (pilar 1–5). Pemilik produk sudah tepat menolak
rollover & multi-tenant. **Pertahankan disiplin itu.**

**Rujukan:** *YAGNI* & *You Aren't Gonna Need It* (XP, Ron Jeffries); *Worse is Better*.

---

## Ringkasan prioritas saran

| Prioritas | Saran | Fase |
|---|---|---|
| 🔴 Kritis | Integer money; ledger reserve/commit; webhook signature+idempoten; backend otoritatif | P03–P05 |
| 🔴 Kritis | Pool + circuit breaker + kill switch (blast radius) | P03 |
| 🟠 Tinggi | Redis + versioned cache; SSE + pub/sub | P02, P07 |
| 🟠 Tinggi | Audit log append-only; expand/contract | P02 |
| 🟡 Sedang | Konvensi prorata/hari; UU PDP; observability | P05, P02, P08 |
| 🟢 Disiplin | YAGNI; jangan hapus data saat downgrade | semua |

## Rujukan utama

- Martin Fowler — *StranglerFigApplication*, *ParallelChange*
- Michael Nygard — *Release It!* (Circuit Breaker, Bulkhead)
- Stripe API Docs — *Idempotent requests*, *Proration*, *integer amounts*
- Midtrans Docs — *Webhook signature*, *Snap* (PCI-DSS SAQ-A)
- OWASP — *Client-side trust*, anti-fraud
- UU No. 27/2022 (PDP) — kewajiban data pribadi
- Google SRE Book — *Four Golden Signals*
