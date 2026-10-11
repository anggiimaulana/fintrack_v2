# Runbook — Refund

## 1. Kebijakan

- Refund **manual** (tidak otomatis) pada fase awal.
- Default: **tidak ada refund** untuk downgrade; refund hanya untuk kasus khusus
  (kesalahan sistem, duplikasi pembayaran, kebijakan pemilik produk).
- Semua refund **wajib** tercatat audit.

## 2. Skenario

| Skenario | Perlakuan |
|---|---|
| Duplikasi pembayaran | Refund penuh nominal duplikat |
| Kesalahan sistem | Refund sesuai kebijakan pemilik produk |
| Upgrade lalu berubah pikiran | Tidak ada refund (downgrade berlaku akhir siklus) |
| Chargeback dari gateway | Ikuti prosedur gateway; catat & audit |

## 3. Prosedur

```
1. Terima permintaan refund + alasan
2. Verifikasi: pembayaran ada, nominal, tanggal, alasan valid
3. Keputusan pemilik produk (approve/tolak)
4. Bila approve → proses refund via gateway (atau transfer manual)
5. Catat: payments (entri refund), audit log, notifikasi user
6. Jangan hapus riwayat pembayaran asli
```

## 4. Aturan

1. Refund **tidak menghapus** riwayat; dibuat sebagai entri tersendiri.
2. Refund **tidak otomatis** mengurangi data user.
3. Setiap refund → audit (aktor, waktu, nominal, alasan).
4. Untuk chargeback: ikuti prosedur Midtrans; siapkan bukti.

## 5. Dampak ke Langganan

- Refund penuh atas pembayaran langganan → langganan dapat dinonaktifkan (sesuai kebijakan).
- Refund sebagian (mis. duplikasi) → langganan tetap aktif.

## Referensi

- `KEBIJAKAN_BILLING.md`
- `../keamanan/KEAMANAN_PEMBAYARAN.md`
