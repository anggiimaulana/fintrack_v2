# Keamanan Pembayaran — FinTrack v2

## 1. Prinsip Wajib

1. **Jangan pernah** menyimpan/memproses data kartu (PAN/CVV). Gunakan **Midtrans Snap/hosted** →
   kepatuhan **PCI-DSS SAQ-A**.
2. **Jangan percaya klien.** Halaman "sukses" di browser hanyalah UI.
3. Sumber kebenaran pembayaran = **webhook server-to-server** terverifikasi.
4. Semua mutasi pembayaran **idempoten** (idempotency_key UNIQUE).

## 2. Alur Verifikasi Webhook Midtrans

```
1. Terima notifikasi (order_id, status_code, gross_amount, signature_key)
2. Verifikasi signature:
      sha512(order_id + status_code + gross_amount + server_key)
   → bandingkan dengan signature_key
3. Jika signature invalid → tolak + catat audit (potensi serangan)
4. Cek idempotency: gateway_ref / idempotency_key sudah ada?
   → jika ya: abaikan (replay) → balas 200
5. Verifikasi ulang status via Midtrans Status API (jangan percaya payload mentah)
6. Jika settlement/capture valid → tandai paid → aktifkan/perpanjang langganan
7. Publikasikan event realtime (SSE) ke user
8. Balas 200
```

## 3. Aturan Idempotensi

- `payments.idempotency_key` **UNIQUE**; `gateway_ref` (order_id) **UNIQUE**.
- Replay webhook tidak boleh menagih/mengaktifkan dua kali.
- Retry dari gateway diperlakukan sebagai operasi yang sama.

## 4. Pembayaran Manual (Bukti Transfer)

- User unggah bukti → `payments(gateway=manual, proof_url, status=pending_review)`.
- Masuk **antrean review super admin** → approve/reject (wajib alasan bila reject).
- Approve → `paid` → aktivasi langganan; **wajib audit** (siapa approve, kapan).
- Validasi file: tipe, ukuran, pemindaian (bila tersedia).

## 5. Switch Gateway (aman)

- `settings.payment_gateway` (global) + `plans.gateway_override`.
- Berlaku untuk **checkout baru**; transaksi in-flight **tetap** terikat gateway asalnya.
- Webhook dari gateway lama tetap diproses selama ada transaksi pending.

## 6. Anti-Fraud & Anomali

- Deteksi pola: banyak pembayaran gagal, nominal tak wajar, replay berulang, IP mencurigakan.
- Batasi rate pembuatan checkout.
- Catat semua kegagalan verifikasi signature ke audit log.

## 7. Refund

- Kebijakan: refund **manual** (dicatat), tidak otomatis di fase awal.
- Refund tidak menghapus riwayat; dibuat sebagai entri tersendiri + audit.

## 8. Data yang Disimpan

| Disimpan | Tidak disimpan |
|---|---|
| order_id / gateway_ref, amount, status, method, proof_url | PAN, CVV, data kartu apa pun |
| waktu, user_id, subscription_id | kredensial gateway di log |

## Referensi

- `../../2_Projects/P05_Payments-Billing/teknis/midtrans-webhook.md`
- Midtrans Docs (signature & Snap)
- OWASP (client-side trust, anti-fraud)
