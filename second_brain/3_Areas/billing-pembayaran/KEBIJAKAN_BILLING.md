# Kebijakan Billing (Mengikat) — FinTrack v2

> Aturan di bawah adalah **kebijakan yang mengikat**. Perubahan hanya lewat keputusan baru bertanggal.

## 1. Uang & Presisi

- Semua nilai uang disimpan sebagai **integer rupiah** (`int`), tanpa desimal.
- Token disimpan sebagai **integer**.
- Konversi/format hanya di layer tampilan (`Money` component).
- Pembulatan prorata: **ke bawah** (menguntungkan user), didokumentasikan eksplisit.

## 2. Siklus Langganan

- Siklus = `current_period_start + billing_period_days` hari kalender, TZ **Asia/Jakarta**.
- Contoh: mulai 5 Agu 00:00 → berakhir 4 Sep 23:59:59.
- Kuota token **direset** pada awal setiap siklus.
- **Tidak ada rollover** token antar-siklus.

## 3. Kuota Token & Ledger

- Setiap request AI: **reserve** → call → **commit** (usage nyata) / **release** (gagal).
- `token_ledger` **append-only**; tidak ada `UPDATE`/`DELETE` pada entri.
- `idempotency_key` **UNIQUE** — retry tidak menggandakan.
- Kuota habis → **soft-limit**: blokir request AI berikutnya (402), request berjalan tidak dipotong.

## 4. Harga Dinamis

```
harga = base_per_1k_idr × (token / 1000) × markup_multiplier × (1 + markup_percent/100) × modality_multiplier
```

- `base_per_1k_idr`, `markup_multiplier`, `markup_percent`, `modality_multipliers` diatur super admin.
- Dapat berbeda per plan (`scope=plan`) atau global (`scope=global`).

## 5. Upgrade

- Berlaku **instan** setelah pembayaran sukses.
- Prorata: `selisih = (harga_baru − harga_lama) × sisa_hari / total_hari`, dibulatkan ke bawah.
- Kuota disesuaikan proporsional; event `upgrade` dicatat.

## 6. Downgrade & Cancel

- Berlaku **akhir siklus** (`cancel_at_period_end`).
- **Tanpa refund**.
- Saat siklus berakhir: plan baru aktif; **soft-limit** bila pemakaian melebihi kuota baru.
- **Data user tidak dihapus.**

## 7. Renewal & Grace

- Hybrid: auto bila metode mendukung; selain itu manual.
- Reminder H-3, H-1, H-0.
- Grace period (default 3 hari, dari `settings`) — fitur tetap jalan.
- Lewat grace → status `past_due` → dunning → turun ke plan free.

## 8. Pembayaran

- Midtrans (Snap/hosted) atau manual (bukti transfer).
- Pembayaran sah **hanya** setelah webhook terverifikasi (Midtrans) atau approval super admin (manual).
- Switch gateway berlaku untuk **checkout baru**; transaksi in-flight tetap gateway asalnya.

## 9. Invoice

- Dasar: nomor, tanggal, item, total, status; **siap PPN** (kolom `tax_idr` ada, default 0).
- Nomor unik & berurutan.

## 10. Audit & Kepatuhan

- Semua mutasi billing → audit log (aktor, aksi, before→after).
- Retensi invoice: sesuai kebijakan pajak (umumnya 10 tahun); konflik dengan hak hapus PDP → anonimisasi.

## Referensi

- `RUNBOOK_DUNNING.md`, `RUNBOOK_PEMBAYARAN_MANUAL.md`, `RUNBOOK_REFUND.md`
- `../../2_Projects/P04_Subscription-Token-Core/teknis/token-ledger.md`
