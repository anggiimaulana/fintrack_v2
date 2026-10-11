# Runbook — Pembayaran Manual (Review Bukti Transfer)

## 1. Tujuan

Memverifikasi pembayaran manual (transfer bank/e-wallet) dengan aman, cepat, dan auditable.

## 2. Alur

```
1. User memilih pembayaran manual → sistem terbitkan invoice + instruksi transfer
2. User transfer → unggah bukti (gambar/PDF)
3. payments(status=pending_review, proof_url=...)
4. Masuk antrean review super admin
5. Super admin memeriksa: nominal, tanggal, tujuan, kecocokan dengan invoice
6. Approve → payments.status=paid → aktivasi/perpanjang langganan
   Reject → sertakan alasan → notifikasi user
7. Catat audit (siapa, kapan, keputusan)
```

## 3. Kriteria Verifikasi

| Cek | Detail |
|---|---|
| Nominal | Harus **≥** total invoice (integer rupiah) |
| Tujuan | Rekening/e-wallet resmi platform |
| Tanggal | Dalam rentang wajar (mis. ≤ 3 hari) |
| Keunikan | Bukti belum dipakai untuk invoice lain (anti-duplikasi) |
| Kejelasan | Nomor referensi/berita transfer terbaca |

## 4. Aturan

1. **Approve wajib tercatat** (audit: aktor, waktu, invoice, nominal).
2. **Idempoten** — approve dua kali tidak menggandakan aktivasi.
3. **Jangan** menyimpan data kartu; bukti transfer disimpan aman (akses terbatas).
4. Reject **wajib** disertai alasan.
5. SLA review: usulan ≤ 1×24 jam.

## 5. Anomali

- Bukti duplikat / nominal tidak cocok → reject + catatan.
- Indikasi pemalsuan → tolak, catat insiden (lihat `../keamanan/RESPONS_INSIDEN.md`).

## 6. Metrik

- Antrean review (jumlah & umur), waktu approve rata-rata, rasio reject.

## Referensi

- `KEBIJAKAN_BILLING.md`
- `../../2_Projects/P05_Payments-Billing/teknis/manual-payment.md`
- `../../2_Projects/P06_Super-Admin-Dashboard/teknis/payment-review-ui.md`
